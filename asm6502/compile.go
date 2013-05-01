package asm6502

import (
	"github.com/axw/gollvm/llvm"
	"os"
	"fmt"
	"bytes"
)

type Compilation struct {
	Warnings []string
	Errors []string

	program *Program
	mod llvm.Module
	builder llvm.Builder
	rX llvm.Value
	rY llvm.Value
	rA llvm.Value
	rSNeg llvm.Value // whether the last arithmetic result is negative
	rSZero llvm.Value // whether the last arithmetic result is zero
	rSDec llvm.Value // decimal
	rSInt llvm.Value // interrupt disable

	// ABI
	mainFn llvm.Value
	putCharFn llvm.Value
	exitFn llvm.Value
	ppuStatusFn llvm.Value

	labeledData map[string] llvm.Value
	labeledBlocks map[string] llvm.BasicBlock
	currentValue *bytes.Buffer
	currentLabel string
	mode int
	// map label name to basic block
	basicBlocks map[string] llvm.BasicBlock
	currentBlock *llvm.BasicBlock
	// label names to look for
	nmiLabelName string
	resetLabelName string
	irqLabelName string
	nmiBlock *llvm.BasicBlock
	resetBlock *llvm.BasicBlock
	irqBlock *llvm.BasicBlock
}

type Compiler interface {
	Compile(*Compilation)
}

const (
	dataStmtMode = iota
	basicBlocksMode
	compileMode
)


type CompileFlags int
const (
	DisableOptFlag CompileFlags = 1 << iota
	DumpModuleFlag
	DumpModulePreFlag
)


func (c *Compilation) dataStop() {
	if len(c.currentLabel) == 0 { return }
	if c.currentValue.Len() == 0 { return }
	text := llvm.ConstString(c.currentValue.String(), false)
	strGlobal := llvm.AddGlobal(c.mod, text.Type(), c.currentLabel)
	strGlobal.SetLinkage(llvm.PrivateLinkage)
	strGlobal.SetInitializer(text)
	c.labeledData[c.currentLabel] = strGlobal
	c.currentLabel = ""
}

func (c *Compilation) dataStart(stmt *LabeledStatement) {
	c.currentLabel = stmt.LabelName
	c.currentValue = new(bytes.Buffer)
}

func (c *Compilation) Visit(n Node) {
	switch c.mode {
	case dataStmtMode:
		c.visitForDataStmts(n)
	case basicBlocksMode:
		c.visitForBasicBlocks(n)
	case compileMode:
		c.visitForCompile(n)
	}
}

func (c *Compilation) visitForBasicBlocks(n Node) {
	switch t := n.(type) {
	case *LabeledStatement:
		t.CompileLabels(c)
	}
}

func (c *Compilation) visitForCompile(n Node) {
	switch t := n.(type) {
	case Compiler:
		t.Compile(c)
	}
}

func (c *Compilation) visitForDataStmts(n Node) {
	switch t := n.(type) {
	case DataList:
	case *IntegerDataItem:
		// trash the data
		if len(c.currentLabel) == 0 { return }
		err := c.currentValue.WriteByte(byte(*t))
		if err != nil {
			c.Errors = append(c.Errors, err.Error())
			return
		}
	case *StringDataItem:
		// trash the data
		if len(c.currentLabel) == 0 { return }
		_, err := c.currentValue.WriteString(string(*t))
		if err != nil {
			c.Errors = append(c.Errors, err.Error())
			return
		}
	case *DataStatement:
		if len(c.currentLabel) == 0 {
			c.Warnings = append(c.Warnings, fmt.Sprintf("trashing data at 0x%04x", t.Offset))
			return
		}
	case *LabeledStatement:
		c.dataStop()
		c.dataStart(t)
	default:
		c.dataStop()
	}
}

func (c *Compilation) VisitEnd(n Node) {}

func (c *Compilation) testAndSetZero(v int) {
	if v == 0 {
		c.setZero()
		return
	}
	c.clearZero()
}

func (c *Compilation) setZero() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 1, false), c.rSZero)
}

func (c *Compilation) clearZero() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 0, false), c.rSZero)
}

func (c *Compilation) setDec() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 1, false), c.rSDec)
}

func (c *Compilation) clearDec() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 0, false), c.rSDec)
}

func (c *Compilation) setInt() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 1, false), c.rSInt)
}

func (c *Compilation) clearInt() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 0, false), c.rSInt)
}

func (c *Compilation) testAndSetNeg(v int) {
	if v & 0x80 == 0x80 {
		c.setNeg()
		return
	}
	c.clearNeg()
}

func (c *Compilation) setNeg() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 1, false), c.rSNeg)
}

func (c *Compilation) clearNeg() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 0, false), c.rSNeg)
}

func (c *Compilation) dynTestAndSetNeg(v llvm.Value) {
	x80 := llvm.ConstInt(llvm.Int8Type(), uint64(0x80), false)
	masked := c.builder.CreateAnd(v, x80, "")
	isNeg := c.builder.CreateICmp(llvm.IntEQ, masked, x80, "")
	c.builder.CreateStore(isNeg, c.rSNeg)
}

func (c *Compilation) dynTestAndSetZero(v llvm.Value) {
	zeroConst := llvm.ConstInt(llvm.Int8Type(), uint64(0), false)
	isZero := c.builder.CreateICmp(llvm.IntEQ, v, zeroConst, "")
	c.builder.CreateStore(isZero, c.rSZero)
}

func (c *Compilation) store(addr int, i8 llvm.Value) {
	i32 := c.builder.CreateZExt(i8, llvm.Int32Type(), "")
	switch addr {
	case 0x2008: // putchar
		c.builder.CreateCall(c.putCharFn, []llvm.Value{i32}, "")
	case 0x2009: // exit
		c.builder.CreateCall(c.exitFn, []llvm.Value{i32}, "")
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("writing to memory address 0x%04x is unsupported", addr))
	}
}

func (i *ImmediateInstruction) Compile(c *Compilation) {
	v := llvm.ConstInt(llvm.Int8Type(), uint64(i.Value), false)
	switch i.OpCode {
	case 0xa2: // ldx
		c.builder.CreateStore(v, c.rX)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
	case 0xa0: // ldy
		c.builder.CreateStore(v, c.rY)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
	case 0xa9: // lda
		c.builder.CreateStore(v, c.rA)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
	//case 0x69: // adc
	//case 0x29: // and
	//case 0xc9: // cmp
	//case 0xe0: // cpx
	//case 0xc0: // cpy
	//case 0x49: // eor
	//case 0x09: // ora
	//case 0xe9: // sbc
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s immediate lacks Compile() implementation", i.OpName))
	}
}

func (i *ImpliedInstruction) Compile(c *Compilation) {
	switch i.OpCode {
	//case 0x0a: // asl
	//case 0x00: // brk
	//case 0x18: // clc
	case 0xd8: // cld
		c.clearDec()
	case 0x58: // cli
		c.clearInt()
	//case 0xb8: // clv
	//case 0xca: // dex
	//case 0x88: // dey
	case 0xe8: // inx
		oldX := c.builder.CreateLoad(c.rX, "")
		c1 := llvm.ConstInt(llvm.Int8Type(), uint64(1), false)
		newX := c.builder.CreateAdd(oldX, c1, "")
		c.builder.CreateStore(newX, c.rX)
		c.dynTestAndSetNeg(newX)
		c.dynTestAndSetZero(newX)
	//case 0xc8: // iny
	//case 0x4a: // lsr
	case 0xea: // nop
		// do nothing
	//case 0x48: // pha
	//case 0x08: // php
	//case 0x68: // pla
	//case 0x28: // plp
	//case 0x2a: // rol
	//case 0x6a: // ror
	case 0x40: // rti
		c.Warnings = append(c.Warnings, "interrupts not supported - ignoring RTI instruction")
		c.builder.CreateUnreachable()
	//case 0x60: // rts
	//case 0x38: // sec
	case 0xf8: // sed
		c.setDec()
	case 0x78: // sei
		c.setInt()
	//case 0xaa: // tax
	//case 0xa8: // tay
	//case 0xba: // tsx
	//case 0x8a: // txa
	//case 0x9a: // txs
	//case 0x98: // tya
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s implied lacks Compile() implementation", i.OpName))
	}
}

func (i *DirectWithLabelIndexedInstruction) Compile(c *Compilation) {
	switch i.OpCode {
	case 0xbd: // lda l, X
		dataPtr := c.labeledData[i.LabelName]
		index := c.builder.CreateLoad(c.rX, "")
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int8Type(), 0, false),
			index,
		}
		ptr := c.builder.CreateGEP(dataPtr, indexes, "")
		v := c.builder.CreateLoad(ptr, "")
		c.builder.CreateStore(v, c.rA)
		c.dynTestAndSetNeg(v)
		c.dynTestAndSetZero(v)
	//case 0x7d: // adc l, X
	//case 0x3d: // and l, X
	//case 0x1e: // asl l, X
	//case 0xdd: // cmp l, X
	//case 0xde: // dec l, X
	//case 0x5d: // eor l, X
	//case 0xfe: // inc l, X
	//case 0xbc: // ldy l, X
	//case 0x5e: // lsr l, X
	//case 0x1d: // ora l, X
	//case 0x3e: // rol l, X
	//case 0x7e: // ror l, X
	//case 0xfd: // sbc l, X
	//case 0x9d: // sta l, X

	//case 0x79: // adc l, Y
	//case 0x39: // and l, Y
	//case 0xd9: // cmp l, Y
	//case 0x59: // eor l, Y
	//case 0xb9: // lda l, Y
	//case 0xbe: // ldx l, Y
	//case 0x19: // ora l, Y
	//case 0xf9: // sbc l, Y
	//case 0x99: // sta l, Y
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s <label>, %s lacks Compile() implementation", i.OpName, i.RegisterName))
	}
}

func (i *DirectIndexedInstruction) Compile(c *Compilation) {
	c.Errors = append(c.Errors, "DirectIndexedInstruction lacks Compile() implementation")
}

func (i *DirectWithLabelInstruction) Compile(c *Compilation) {
	switch i.OpCode {
	//case 0x6d: // adc
	//case 0x2d: // and
	//case 0x0e: // asl
	//case 0x2c: // bit
	//case 0xcd: // cmp
	//case 0xec: // cpx
	//case 0xcc: // cpy
	//case 0xce: // dec
	//case 0x4d: // eor
	//case 0xee: // inc
	case 0x4c: // jmp
		destBlock := c.labeledBlocks[i.LabelName]
		c.builder.CreateBr(destBlock)
		c.currentBlock = nil
	//case 0x20: // jsr
	//case 0xad: // lda
	//case 0xae: // ldx
	//case 0xac: // ldy
	//case 0x4e: // lsr
	//case 0x0d: // ora
	//case 0x2e: // rol
	//case 0x6e: // ror
	//case 0xed: // sbc
	//case 0x8d: // sta
	//case 0x8e: // stx
	//case 0x8c: // sty

	case 0xf0: // beq
		thenBlock := c.labeledBlocks[i.LabelName]
		elseBlock := llvm.InsertBasicBlock(*c.currentBlock, "else")
		isZero := c.builder.CreateLoad(c.rSZero, "")
		c.builder.CreateCondBr(isZero, thenBlock, elseBlock)
		c.builder.SetInsertPointAtEnd(elseBlock)
		elseBlock.MoveAfter(*c.currentBlock)
		c.currentBlock = &elseBlock
	//case 0x90: // bcc
	//case 0xb0: // bcs
	case 0x30: // bmi
		thenBlock := c.labeledBlocks[i.LabelName]
		elseBlock := llvm.InsertBasicBlock(*c.currentBlock, "else")
		isNeg := c.builder.CreateLoad(c.rSNeg, "")
		c.builder.CreateCondBr(isNeg, thenBlock, elseBlock)
		c.builder.SetInsertPointAtEnd(elseBlock)
		elseBlock.MoveAfter(*c.currentBlock)
		c.currentBlock = &elseBlock
	case 0xd0: // bne
		thenBlock := llvm.InsertBasicBlock(*c.currentBlock, "then")
		elseBlock := c.labeledBlocks[i.LabelName]
		isZero := c.builder.CreateLoad(c.rSZero, "")
		c.builder.CreateCondBr(isZero, thenBlock, elseBlock)
		c.builder.SetInsertPointAtEnd(thenBlock)
		thenBlock.MoveAfter(*c.currentBlock)
		c.currentBlock = &thenBlock
	case 0x10: // bpl
		thenBlock := llvm.InsertBasicBlock(*c.currentBlock, "then")
		elseBlock := c.labeledBlocks[i.LabelName]
		isNeg := c.builder.CreateLoad(c.rSNeg, "")
		c.builder.CreateCondBr(isNeg, thenBlock, elseBlock)
		c.builder.SetInsertPointAtEnd(thenBlock)
		thenBlock.MoveAfter(*c.currentBlock)
		c.currentBlock = &thenBlock
	//case 0x50: // bvc
	//case 0x70: // bvs
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s <label> lacks Compile() implementation", i.OpName))
	}
}

func (i *DirectInstruction) Compile(c *Compilation) {
	switch i.Payload[0] {
	case 0xa5: fallthrough // lda zpg
	case 0xad: // lda abs
		switch i.Value {
		case 0x2002:
			v := c.builder.CreateCall(c.ppuStatusFn, []llvm.Value{}, "")
			c.dynTestAndSetZero(v)
			c.dynTestAndSetNeg(v)
		default:
			c.Errors = append(c.Errors, "only LDA $2002 is supported")
		}
	//case 0x90: // bcc rel
	//case 0xb0: // bcs rel
	//case 0xf0: // beq rel
	//case 0x30: // bmi rel
	//case 0xd0: // bne rel
	//case 0x10: // bpl rel
	//case 0x50: // bvc rel
	//case 0x70: // bvs rel

	//case 0x65: // adc zpg
	//case 0x25: // and zpg
	//case 0x06: // asl zpg
	//case 0x24: // bit zpg
	//case 0xc5: // cmp zpg
	//case 0xe4: // cpx zpg
	//case 0xc4: // cpy zpg
	//case 0xc6: // dec zpg
	//case 0x45: // eor zpg
	//case 0xe6: // inc zpg
	//case 0xa6: // ldx zpg
	//case 0xa4: // ldy zpg
	//case 0x46: // lsr zpg
	//case 0x05: // ora zpg
	//case 0x26: // rol zpg
	//case 0x66: // ror zpg
	//case 0xe5: // sbc zpg

	//case 0x6d: // adc abs
	//case 0x2d: // and abs
	//case 0x0e: // asl abs
	//case 0x2c: // bit abs
	//case 0xcd: // cmp abs
	//case 0xec: // cpx abs
	//case 0xcc: // cpy abs
	//case 0xce: // dec abs
	//case 0x4d: // eor abs
	//case 0xee: // inc abs
	//case 0x4c: // jmp abs
	//case 0x20: // jsr abs
	//case 0xae: // ldx abs
	//case 0xac: // ldy abs
	//case 0x4e: // lsr abs
	//case 0x0d: // ora abs
	//case 0x2e: // rol abs
	//case 0x6e: // ror abs
	//case 0xed: // sbc abs
	case 0x85: fallthrough // sta zpg
	case 0x8d: // sta abs
		c.store(i.Value, c.builder.CreateLoad(c.rA, ""))
	case 0x86: fallthrough // stx zpg
	case 0x8e: // stx abs
		c.store(i.Value, c.builder.CreateLoad(c.rX, ""))
	case 0x84: fallthrough // sty zpg
	case 0x8c: // sty abs
		c.store(i.Value, c.builder.CreateLoad(c.rY, ""))
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s direct lacks Compile() implementation", i.OpName))
	}
}

func (i *IndirectXInstruction) Compile(c *Compilation) {
	c.Errors = append(c.Errors, "IndirectXInstruction lacks Compile() implementation")
}

func (i *IndirectYInstruction) Compile(c *Compilation) {
	c.Errors = append(c.Errors, "IndirectYInstruction lacks Compile() implementation")
}

func (i *IndirectInstruction) Compile(c *Compilation) {
	c.Errors = append(c.Errors, "IndirectInstruction lacks Compile() implementation")
}

func (s *LabeledStatement) Compile(c *Compilation) {
	// if we've already processed it as data, move on
	_, ok := c.labeledData[s.LabelName]
	if ok { return }

	bb := c.labeledBlocks[s.LabelName]
	if c.currentBlock != nil {
		c.builder.CreateBr(bb)
	}
	c.currentBlock = &bb
	c.builder.SetInsertPointAtEnd(bb)

	switch s.LabelName {
	case c.nmiLabelName:
		c.currentBlock = nil
	case c.irqLabelName:
		c.currentBlock = nil
	}
}

func (s *LabeledStatement) CompileLabels(c *Compilation) {
	// if we've already processed it as data, move on
	_, ok := c.labeledData[s.LabelName]
	if ok { return }

	bb := llvm.AddBasicBlock(c.mainFn, s.LabelName)
	c.labeledBlocks[s.LabelName] = bb

	switch s.LabelName {
	case c.nmiLabelName:
		c.nmiBlock = &bb
	case c.resetLabelName:
		c.resetBlock = &bb
	case c.irqLabelName:
		c.irqBlock = &bb
	}
}

func (c *Compilation) setUpEntryPoint(p *Program, addr int, s *string) {
	n, ok := p.offsets[addr]
	if !ok {
		c.Errors = append(c.Errors, fmt.Sprintf("Missing 0x%04x entry point"))
		return
	}
	stmt, ok := n.(*DataWordStatement)
	if !ok {
		c.Errors = append(c.Errors, fmt.Sprintf("Entry point at 0x%04x must be a dc.w"))
		return
	}
	call, ok := stmt.dataList[0].(*LabelCall)
	if !ok {
		c.Errors = append(c.Errors, fmt.Sprintf("Entry point at 0x%04x must be a dc.w with a label"))
		return
	}
	*s = call.LabelName
}

func (p *Program) Compile(filename string, flags CompileFlags) (c *Compilation) {
	llvm.InitializeNativeTarget()

	c = new(Compilation)
	c.program = p
	c.Warnings = []string{}
	c.Errors = []string{}
	c.mod = llvm.NewModule("asm_module")
	c.builder = llvm.NewBuilder()
	defer c.builder.Dispose()
	c.labeledData = map[string] llvm.Value{}
	c.labeledBlocks = map[string] llvm.BasicBlock{}

	// first pass to generate data declarations
	c.mode = dataStmtMode
	p.Ast.Ast(c)

	// declare i32 @putchar(i32)
	putCharType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{llvm.Int32Type()}, false)
	c.putCharFn = llvm.AddFunction(c.mod, "putchar", putCharType)
	c.putCharFn.SetLinkage(llvm.ExternalLinkage)

	// declare void @exit(i32) noreturn nounwind
	exitType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int32Type()}, false)
	c.exitFn = llvm.AddFunction(c.mod, "exit", exitType)
	c.exitFn.AddFunctionAttr(llvm.NoReturnAttribute|llvm.NoUnwindAttribute)
	c.exitFn.SetLinkage(llvm.ExternalLinkage)

	// declare i8 @ppustatus()
	c.ppuStatusFn = llvm.AddFunction(c.mod, "ppustatus", llvm.FunctionType(llvm.Int8Type(), []llvm.Type{}, false))
	c.ppuStatusFn.SetLinkage(llvm.ExternalLinkage)

	// main function / entry point
	mainType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{}, false)
	c.mainFn = llvm.AddFunction(c.mod, "main", mainType)
	c.mainFn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(c.mainFn, "Entry")
	c.builder.SetInsertPointAtEnd(entry)
	c.rX = c.builder.CreateAlloca(llvm.Int8Type(), "X")
	c.rY = c.builder.CreateAlloca(llvm.Int8Type(), "Y")
	c.rA = c.builder.CreateAlloca(llvm.Int8Type(), "A")
	c.rSNeg = c.builder.CreateAlloca(llvm.Int1Type(), "S_neg")
	c.rSZero = c.builder.CreateAlloca(llvm.Int1Type(), "S_zero")
	c.rSDec = c.builder.CreateAlloca(llvm.Int1Type(), "S_dec")
	c.rSInt = c.builder.CreateAlloca(llvm.Int1Type(), "S_int")

	// set up entry points
	c.setUpEntryPoint(p, 0xfffa, &c.nmiLabelName)
	c.setUpEntryPoint(p, 0xfffc, &c.resetLabelName)
	c.setUpEntryPoint(p, 0xfffe, &c.irqLabelName)

	// second pass to build basic blocks
	c.mode = basicBlocksMode
	p.Ast.Ast(c)

	// finally, one last pass for codegen
	c.mode = compileMode
	p.Ast.Ast(c)

	// hook up entry points
	if c.nmiBlock == nil {
		c.Errors = append(c.Errors, "missing nmi entry point")
		return
	}
	if c.resetBlock == nil {
		c.Errors = append(c.Errors, "missing reset entry point")
		return
	}
	if c.irqBlock == nil {
		c.Errors = append(c.Errors, "missing irq entry point")
		return
	}

	// hook up the first entry block to the reset block
	c.builder.SetInsertPointAtEnd(entry)
	c.builder.CreateBr(*c.resetBlock)

	if flags & DumpModulePreFlag != 0 {
		c.mod.Dump()
	}
	err := llvm.VerifyModule(c.mod, llvm.ReturnStatusAction)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}

	engine, err := llvm.NewJITCompiler(c.mod, 3)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}
	defer engine.Dispose()

	if flags & DisableOptFlag == 0 {
		pass := llvm.NewPassManager()
		defer pass.Dispose()

		pass.Add(engine.TargetData())
		pass.AddConstantPropagationPass()
		pass.AddInstructionCombiningPass()
		pass.AddPromoteMemoryToRegisterPass()
		pass.AddGVNPass()
		pass.AddCFGSimplificationPass()
		pass.AddDeadStoreEliminationPass()
		pass.AddGlobalDCEPass()
		pass.Run(c.mod)
	}

	if flags & DumpModuleFlag != 0 {
		c.mod.Dump()
	}

	fd, err := os.Create(filename)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}

	err = llvm.WriteBitcodeToFile(c.mod, fd)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}

	err = fd.Close()
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}

	return
}
