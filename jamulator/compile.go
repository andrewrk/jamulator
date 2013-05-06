package jamulator

// generates a module compatible with runtime/rom.h

import (
	"bytes"
	"fmt"
	"github.com/axw/gollvm/llvm"
	"os"
)

type Compilation struct {
	Warnings []string
	Errors   []string
	Flags    CompileFlags

	program         *Program
	mod             llvm.Module
	builder         llvm.Builder
	wram            llvm.Value // 2KB WRAM
	rX              llvm.Value // X index register
	rY              llvm.Value // Y index register
	rA              llvm.Value // accumulator
	rSP             llvm.Value // stack pointer
	rPC             llvm.Value // program counter
	rSNeg           llvm.Value // whether the last arithmetic result is negative
	rSOver          llvm.Value // whether the last arithmetic result overflowed
	rSBrk           llvm.Value // break
	rSDec           llvm.Value // decimal
	rSInt           llvm.Value // irq interrupt disable
	rSZero          llvm.Value // whether the last arithmetic result is zero
	rSCarry         llvm.Value // carry
	runtimePanicMsg llvm.Value // we print this when a runtime error occurs

	// ABI
	mainFn         llvm.Value
	printfFn       llvm.Value
	putCharFn      llvm.Value
	exitFn         llvm.Value
	cycleFn        llvm.Value
	ppuStatusFn    llvm.Value
	ppuCtrlFn      llvm.Value
	ppuMaskFn      llvm.Value
	ppuAddrFn      llvm.Value
	setPpuDataFn   llvm.Value
	oamAddrFn      llvm.Value
	setOamDataFn   llvm.Value
	setPpuScrollFn llvm.Value

	labeledData   map[string]llvm.Value
	labeledBlocks map[string]llvm.BasicBlock
	currentValue  *bytes.Buffer
	currentLabel  string
	mode          int
	// map label name to basic block
	basicBlocks  map[string]llvm.BasicBlock
	currentBlock *llvm.BasicBlock
	// label names to look for
	nmiLabelName   string
	resetLabelName string
	irqLabelName   string
	nmiBlock       *llvm.BasicBlock
	resetBlock     *llvm.BasicBlock
	irqBlock       *llvm.BasicBlock
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
	IncludeDebugFlag
)

func (c *Compilation) dataStop() {
	if len(c.currentLabel) == 0 {
		return
	}
	if c.currentValue.Len() == 0 {
		c.currentLabel = "";
		return
	}
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
		if len(c.currentLabel) == 0 {
			return
		}
		err := c.currentValue.WriteByte(byte(*t))
		if err != nil {
			c.Errors = append(c.Errors, err.Error())
			return
		}
	case *StringDataItem:
		// trash the data
		if len(c.currentLabel) == 0 {
			return
		}
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
	if v&0x80 == 0x80 {
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
	x80 := llvm.ConstInt(llvm.Int8Type(), 0x80, false)
	masked := c.builder.CreateAnd(v, x80, "")
	isNeg := c.builder.CreateICmp(llvm.IntEQ, masked, x80, "")
	c.builder.CreateStore(isNeg, c.rSNeg)
}

func (c *Compilation) dynTestAndSetZero(v llvm.Value) {
	zeroConst := llvm.ConstInt(llvm.Int8Type(), 0, false)
	isZero := c.builder.CreateICmp(llvm.IntEQ, v, zeroConst, "")
	c.builder.CreateStore(isZero, c.rSZero)
}

func (c *Compilation) store(addr int, i8 llvm.Value) {
	//c.debugPrintf(fmt.Sprintf("static store $%04x %s\n", addr, "#$%02x"), []llvm.Value{i8})
	// homebrew ABI
	switch addr {
	case 0x2008: // putchar
		i32 := c.builder.CreateZExt(i8, llvm.Int32Type(), "")
		c.builder.CreateCall(c.putCharFn, []llvm.Value{i32}, "")
		return
	case 0x2009: // exit
		i32 := c.builder.CreateZExt(i8, llvm.Int32Type(), "")
		c.builder.CreateCall(c.exitFn, []llvm.Value{i32}, "")
		return
	}
	switch {
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("writing to memory address 0x%04x is unsupported", addr))
	case 0x0000 <= addr && addr < 0x2000:
		// 2KB working RAM. mask because mirrored
		maskedAddr := addr & (0x800 - 1)
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int8Type(), 0, false),
			llvm.ConstInt(llvm.Int8Type(), uint64(maskedAddr), false),
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		c.builder.CreateStore(i8, ptr)
	case 0x2000 <= addr && addr < 0x4000:
		// PPU registers. mask because mirrored
		switch addr & (0x8 - 1) {
		case 0: // ppuctrl
			c.debugPrint("ppu_write_control\n")
			c.builder.CreateCall(c.ppuCtrlFn, []llvm.Value{i8}, "")
		case 1: // ppumask
			c.debugPrint("ppu_write_mask\n")
			c.builder.CreateCall(c.ppuMaskFn, []llvm.Value{i8}, "")
		case 3: // oamaddr
			c.debugPrint("ppu_write_oamaddr\n")
			c.builder.CreateCall(c.oamAddrFn, []llvm.Value{i8}, "")
		case 4: // oamdata
			c.debugPrint("ppu_write_oamdata\n")
			c.builder.CreateCall(c.setOamDataFn, []llvm.Value{i8}, "")
		case 5: // ppuscroll
			c.debugPrint("ppu_write_scroll\n")
			c.builder.CreateCall(c.setPpuScrollFn, []llvm.Value{i8}, "")
		case 6: // ppuaddr
			c.debugPrint("ppu_write_address\n")
			c.builder.CreateCall(c.ppuAddrFn, []llvm.Value{i8}, "")
		case 7: // ppudata
			c.debugPrintf("ppu_write_data $%02x\n", []llvm.Value{i8})
			c.builder.CreateCall(c.setPpuDataFn, []llvm.Value{i8}, "")
		default:
			panic("unreachable")
		}
	}

}

func (c *Compilation) dynLoad(addr llvm.Value, minAddr int, maxAddr int) llvm.Value {
	// returns the byte at addr, with runtime checks for the range between minAddr and maxAddr
	// currently only can do WRAM stuff
	// TODO: support the full address range
	switch {
	case maxAddr < 0x0800:
		// no runtime checks needed.
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int8Type(), 0, false),
			addr,
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		return c.builder.CreateLoad(ptr, "")
	case maxAddr < 0x4000:
		// address masking needed.
		maskedAddr := c.builder.CreateAnd(addr, llvm.ConstInt(llvm.Int16Type(), 0x800-1, false), "")
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int8Type(), 0, false),
			maskedAddr,
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		return c.builder.CreateLoad(ptr, "")
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("dynLoad $%04x < x < $%04x currently only supports max address below $4000", minAddr, maxAddr))
		return llvm.ConstInt(llvm.Int8Type(), 0, false)
	}
	panic("unreachable")
}

func (c *Compilation) load(addr int) llvm.Value {
	switch {
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("reading from $%04x not implemented", addr))
		return llvm.ConstNull(llvm.Int8Type())
	case 0x0000 <= addr && addr < 0x2000:
		// 2KB working RAM. mask because mirrored
		maskedAddr := addr & (0x800 - 1)
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int8Type(), 0, false),
			llvm.ConstInt(llvm.Int8Type(), uint64(maskedAddr), false),
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		v := c.builder.CreateLoad(ptr, "")
		//c.debugPrintf(fmt.Sprintf("static load $%04x %s\n", addr, "#$%02x"), []llvm.Value{v})
		return v
	case 0x2000 <= addr && addr < 0x4000:
		// PPU registers. mask because mirrored
		switch addr & (0x8 - 1) {
		case 2:
			c.debugPrint("ppu_read_status\n")
			v := c.builder.CreateCall(c.ppuStatusFn, []llvm.Value{}, "")
			//c.debugPrintf(fmt.Sprintf("static load $%04x %s\n", addr, "#$%02x"), []llvm.Value{v})
			return v
		default:
			c.Errors = append(c.Errors, fmt.Sprintf("reading from $%04x not implemented", addr))
			return llvm.ConstNull(llvm.Int8Type())
		}
	}
	panic("unreachable")
}

func (c *Compilation) increment(reg llvm.Value, delta int) {
	v := c.builder.CreateLoad(reg, "")
	var newValue llvm.Value
	if delta < 0 {
		c1 := llvm.ConstInt(llvm.Int8Type(), uint64(-delta), false)
		newValue = c.builder.CreateSub(v, c1, "")
	} else {
		c1 := llvm.ConstInt(llvm.Int8Type(), uint64(delta), false)
		newValue = c.builder.CreateAdd(v, c1, "")
	}
	c.builder.CreateStore(newValue, reg)
	c.dynTestAndSetNeg(newValue)
	c.dynTestAndSetZero(newValue)
}

func (c *Compilation) transfer(source llvm.Value, dest llvm.Value) {
	v := c.builder.CreateLoad(source, "")
	c.builder.CreateStore(v, dest)
	c.dynTestAndSetNeg(v)
	c.dynTestAndSetZero(v)
}

func (c *Compilation) createBlock(name string) llvm.BasicBlock {
	bb := llvm.InsertBasicBlock(*c.currentBlock, name)
	bb.MoveAfter(*c.currentBlock)
	return bb
}

func (c *Compilation) selectBlock(bb llvm.BasicBlock) {
	c.builder.SetInsertPointAtEnd(bb)
	c.currentBlock = &bb
}

func (c *Compilation) createPanic() {
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	ptr := c.builder.CreatePointerCast(c.runtimePanicMsg, bytePointerType, "")
	c.builder.CreateCall(c.printfFn, []llvm.Value{ptr}, "")
	exitCode := llvm.ConstInt(llvm.Int32Type(), 1, false)
	c.builder.CreateCall(c.exitFn, []llvm.Value{exitCode}, "")
	c.builder.CreateUnreachable()
}

// returns the else block, sets the current block to the if block
func (c *Compilation) createIf(cond llvm.Value) llvm.BasicBlock {
	elseBlock := c.createBlock("else")
	thenBlock := c.createBlock("then")
	c.builder.CreateCondBr(cond, thenBlock, elseBlock)
	c.selectBlock(thenBlock)
	return elseBlock
}

func (c *Compilation) pullFromStack() llvm.Value {
	// increment stack pointer
	sp := c.builder.CreateLoad(c.rSP, "")
	spPlusOne := c.builder.CreateAdd(sp, llvm.ConstInt(llvm.Int8Type(), 1, false), "")
	c.builder.CreateStore(spPlusOne, c.rSP)
	// read the value at stack pointer
	spZExt := c.builder.CreateZExt(sp, llvm.Int16Type(), "")
	addr := c.builder.CreateAdd(spZExt, llvm.ConstInt(llvm.Int16Type(), 0x100, false), "")
	indexes := []llvm.Value{
		llvm.ConstInt(llvm.Int16Type(), 0, false),
		addr,
	}
	ptr := c.builder.CreateGEP(c.wram, indexes, "")
	return c.builder.CreateLoad(ptr, "")
}

func (c *Compilation) pushToStack(v llvm.Value) {
	// write the value to the address at current stack pointer
	sp := c.builder.CreateLoad(c.rSP, "")
	spZExt := c.builder.CreateZExt(sp, llvm.Int16Type(), "")
	addr := c.builder.CreateAdd(spZExt, llvm.ConstInt(llvm.Int16Type(), 0x100, false), "")
	indexes := []llvm.Value{
		llvm.ConstInt(llvm.Int16Type(), 0, false),
		addr,
	}
	ptr := c.builder.CreateGEP(c.wram, indexes, "")
	c.builder.CreateStore(v, ptr)
	// stack pointer = stack pointer - 1
	spMinusOne := c.builder.CreateSub(sp, llvm.ConstInt(llvm.Int8Type(), 1, false), "")
	c.builder.CreateStore(spMinusOne, c.rSP)
}

func (c *Compilation) cycle(count int, pc int, size int) {
	c.debugPrint(fmt.Sprintf("cycles %d\n", count))

	c.builder.CreateStore(llvm.ConstInt(llvm.Int16Type(), uint64(pc + size), false), c.rPC)

	v := llvm.ConstInt(llvm.Int8Type(), uint64(count), false)
	c.builder.CreateCall(c.cycleFn, []llvm.Value{v}, "")
}

func (c *Compilation) debugPrint(str string) {
	c.debugPrintf(str, []llvm.Value{})
}

func (c *Compilation) debugPrintf(str string, values []llvm.Value) {
	if c.Flags&IncludeDebugFlag == 0 {
		return
	}
	text := llvm.ConstString(str, true)
	glob := llvm.AddGlobal(c.mod, text.Type(), "debugPrintStr")
	glob.SetLinkage(llvm.PrivateLinkage)
	glob.SetInitializer(text)
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	ptr := c.builder.CreatePointerCast(glob, bytePointerType, "")
	args := []llvm.Value{ptr}
	for _, v := range values {
		args = append(args, v)
	}
	c.builder.CreateCall(c.printfFn, args, "")
}

func (c *Compilation) createBranch(cond llvm.Value, labelName string, instrAddr int, instrSize int) {
	branchBlock := c.labeledBlocks[labelName]
	thenBlock := c.createBlock("then")
	elseBlock := c.createBlock("else")
	c.builder.CreateCondBr(cond, thenBlock, elseBlock)
	// if the condition is met, the cycle count is 3 or 4, depending
	// on whether the page boundary is crossed.
	c.selectBlock(thenBlock)
	addr := c.program.Labels[labelName]
	if instrAddr&0xff00 == addr&0xff00 {
		c.cycle(3, instrAddr, instrSize)
	} else {
		c.cycle(4, instrAddr, instrSize)
	}
	c.builder.CreateBr(branchBlock)
	// the else block is when the code does *not* branch.
	// in this case, the cycle count is 2.
	c.selectBlock(elseBlock)
	c.cycle(2, instrAddr, instrSize)
}

func (c *Compilation) cyclesForAbsoluteIndexed(baseAddr int, index16 llvm.Value, offset int, size int) {
	// if address & 0xff00 != (address + x) & 0xff00
	baseAddrMasked := baseAddr & 0xff00
	baseAddrMaskedValue := llvm.ConstInt(llvm.Int16Type(), uint64(baseAddrMasked), false)

	baseAddrValue := llvm.ConstInt(llvm.Int16Type(), uint64(baseAddr), false)
	addrPlusX := c.builder.CreateAdd(baseAddrValue, index16, "")
	xff00 := llvm.ConstInt(llvm.Int16Type(), uint64(0xff00), false)
	maskedAddrPlusX := c.builder.CreateAnd(addrPlusX, xff00, "")

	eq := c.builder.CreateICmp(llvm.IntEQ, baseAddrMaskedValue, maskedAddrPlusX, "")
	ldaDoneBlock := c.createBlock("LDA_done")
	pageBoundaryCrossedBlock := c.createIf(eq)
	// executed if page boundary is not crossed
	c.cycle(4, offset, size)
	c.builder.CreateBr(ldaDoneBlock)
	// executed if page boundary crossed
	c.selectBlock(pageBoundaryCrossedBlock)
	c.cycle(5, offset, size)
	c.builder.CreateBr(ldaDoneBlock)
	// done
	c.selectBlock(ldaDoneBlock)
}

func (i *ImmediateInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	v := llvm.ConstInt(llvm.Int8Type(), uint64(i.Value), false)
	switch i.OpCode {
	case 0xa2: // ldx
		c.builder.CreateStore(v, c.rX)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, i.Offset, i.Size)
	case 0xa0: // ldy
		c.builder.CreateStore(v, c.rY)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, i.Offset, i.Size)
	case 0xa9: // lda
		c.builder.CreateStore(v, c.rA)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, i.Offset, i.Size)
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

func (c *Compilation) pullWordFromStack() llvm.Value {
	low := c.pullFromStack()
	high := c.pullFromStack()
	low16 := c.builder.CreateZExt(low, llvm.Int16Type(), "")
	high16 := c.builder.CreateZExt(high, llvm.Int16Type(), "")
	word := c.builder.CreateShl(high16, llvm.ConstInt(llvm.Int16Type(), 8, false), "")
	return c.builder.CreateAnd(word, low16, "")
}

func (i *ImpliedInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	switch i.OpCode {
	//case 0x0a: // asl
	//case 0x00: // brk
	//case 0x18: // clc
	case 0xd8: // cld
		c.clearDec()
		c.cycle(2, i.Offset, i.Size)
	case 0x58: // cli
		c.clearInt()
		c.cycle(2, i.Offset, i.Size)
	//case 0xb8: // clv
	case 0xca: // dex
		c.increment(c.rX, -1)
		c.cycle(2, i.Offset, i.Size)
	case 0x88: // dey
		c.increment(c.rY, -1)
		c.cycle(2, i.Offset, i.Size)
	case 0xe8: // inx
		c.increment(c.rX, 1)
		c.cycle(2, i.Offset, i.Size)
	case 0xc8: // iny
		c.increment(c.rY, 1)
		c.cycle(2, i.Offset, i.Size)
	//case 0x4a: // lsr
	case 0xea: // nop
		c.cycle(2, i.Offset, i.Size)
	//case 0x48: // pha
	//case 0x08: // php
	//case 0x68: // pla
	case 0x28: // plp
		c.pullStatusReg()
		c.cycle(4, i.Offset, i.Size)
	//case 0x2a: // rol
	//case 0x6a: // ror
	case 0x40: // rti
		c.pullStatusReg()
		pc := c.pullWordFromStack()
		c.builder.CreateStore(pc, c.rPC)
		c.cycle(6, i.Offset, i.Size)
		c.builder.CreateRetVoid()
		c.currentBlock = nil
	case 0x60: // rts
		pc := c.pullWordFromStack()
		pc = c.builder.CreateAdd(pc, llvm.ConstInt(llvm.Int16Type(), 1, false), "")
		c.builder.CreateStore(pc, c.rPC)
		c.builder.CreateRetVoid()
		c.currentBlock = nil
	//case 0x38: // sec
	case 0xf8: // sed
		c.setDec()
		c.cycle(2, i.Offset, i.Size)
	case 0x78: // sei
		c.setInt()
		c.cycle(2, i.Offset, i.Size)
	case 0xaa: // tax
		c.transfer(c.rA, c.rX)
		c.cycle(2, i.Offset, i.Size)
	case 0xa8: // tay
		c.transfer(c.rA, c.rY)
		c.cycle(2, i.Offset, i.Size)
	case 0xba: // tsx
		c.transfer(c.rSP, c.rX)
		c.cycle(2, i.Offset, i.Size)
	case 0x8a: // txa
		c.transfer(c.rX, c.rA)
		c.cycle(2, i.Offset, i.Size)
	case 0x9a: // txs
		c.transfer(c.rX, c.rSP)
		c.cycle(2, i.Offset, i.Size)
	case 0x98: // tya
		c.transfer(c.rY, c.rA)
		c.cycle(2, i.Offset, i.Size)
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s implied lacks Compile() implementation", i.OpName))
	}
}

func (i *DirectWithLabelIndexedInstruction) ResolveRender(c *Compilation) string {
	// render, but replace the label with the address
	addr := c.program.Labels[i.LabelName]
	return fmt.Sprintf("%s $%04x, %s\n", i.OpName, addr, i.RegisterName)
}

func (i *DirectWithLabelInstruction) ResolveRender(c *Compilation) string {
	// render, but replace the label with the address
	addr := c.program.Labels[i.LabelName]
	return fmt.Sprintf("%s $%04x\n", i.OpName, addr)
}

func (i *DirectWithLabelIndexedInstruction) Compile(c *Compilation) {
	c.debugPrint(i.ResolveRender(c))
	switch i.OpCode {
	case 0xbd: // lda l, X
		dataPtr := c.labeledData[i.LabelName]
		index := c.builder.CreateLoad(c.rX, "")
		index16 := c.builder.CreateZExt(index, llvm.Int16Type(), "")
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int16Type(), 0, false),
			index16,
		}
		ptr := c.builder.CreateGEP(dataPtr, indexes, "")
		v := c.builder.CreateLoad(ptr, "")
		c.builder.CreateStore(v, c.rA)
		c.dynTestAndSetNeg(v)
		c.dynTestAndSetZero(v)
		c.cyclesForAbsoluteIndexed(c.program.Labels[i.LabelName], index16, i.Offset, i.Size)
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
	switch i.Payload[0] {
	// abs y
	//case 0x79: // adc
	//case 0x39: // and
	//case 0xd9: // cmp
	//case 0x59: // eor
	//case 0xb9: // lda
	//case 0xbe: // ldx
	//case 0x19: // ora
	//case 0xf9: // sbc
	//case 0x99: // sta
	// zpg y
	//case 0xb6: // ldx
	//case 0x96: // stx
	// abs x
	//case 0x7d: // adc
	//case 0x3d: // and
	//case 0x1e: // asl
	//case 0xdd: // cmp
	//case 0xde: // dec
	//case 0x5d: // eor
	//case 0xfe: // inc
	case 0xbd: // lda
		x := c.builder.CreateLoad(c.rX, "")
		x16 := c.builder.CreateZExt(x, llvm.Int16Type(), "")
		addr := c.builder.CreateAdd(x16, llvm.ConstInt(llvm.Int16Type(), uint64(i.Value), false), "")
		v := c.dynLoad(addr, i.Value, i.Value + 0xff)
		c.builder.CreateStore(v, c.rA)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		c.cyclesForAbsoluteIndexed(i.Value, x16, i.Offset, i.GetSize())
	//case 0xbc: // ldy
	//case 0x5e: // lsr
	//case 0x1d: // ora
	//case 0x3e: // rol
	//case 0x7e: // ror
	//case 0xfd: // sbc
	//case 0x9d: // sta
	// zpg x
	//case 0x75: // adc
	//case 0x35: // and
	//case 0x16: // asl
	//case 0xd5: // cmp
	//case 0xd6: // dec
	//case 0x55: // eor
	//case 0xf6: // inc
	//case 0xb5: // lda
	//case 0xb4: // ldy
	//case 0x56: // lsr
	//case 0x15: // ora
	//case 0x36: // rol
	//case 0x76: // ror
	//case 0xf5: // sbc
	//case 0x95: // sta
	//case 0x94: // sty
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s lacks Compile() implementation", i.Render()))
	}
}

func (i *DirectWithLabelInstruction) Compile(c *Compilation) {
	c.debugPrint(i.ResolveRender(c))
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
		// branch instruction - cycle before execution
		c.cycle(3, i.Offset, i.Size)
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
		isZero := c.builder.CreateLoad(c.rSZero, "")
		c.createBranch(isZero, i.LabelName, i.Offset, i.Size)
	//case 0x90: // bcc
	//case 0xb0: // bcs
	case 0x30: // bmi
		isNeg := c.builder.CreateLoad(c.rSNeg, "")
		c.createBranch(isNeg, i.LabelName, i.Offset, i.Size)
	case 0xd0: // bne
		isZero := c.builder.CreateLoad(c.rSZero, "")
		notZero := c.builder.CreateNot(isZero, "")
		c.createBranch(notZero, i.LabelName, i.Offset, i.Size)
	case 0x10: // bpl
		isNeg := c.builder.CreateLoad(c.rSNeg, "")
		notNeg := c.builder.CreateNot(isNeg, "")
		c.createBranch(notNeg, i.LabelName, i.Offset, i.Size)
	//case 0x50: // bvc
	//case 0x70: // bvs
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s <label> lacks Compile() implementation", i.OpName))
	}
}

func (i *DirectInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	switch i.Payload[0] {
	case 0xa5, 0xad: // lda (zpg, abs)
		v := c.load(i.Value)
		c.builder.CreateStore(v, c.rA)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		if i.Payload[0] == 0xa5 {
			c.cycle(3, i.Offset, i.GetSize())
		} else {
			c.cycle(4, i.Offset, i.GetSize())
		}
	case 0xc6, 0xce: // dec (zpg, abs)
		oldValue := c.load(i.Value)
		c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
		newValue := c.builder.CreateSub(oldValue, c1, "")
		c.store(i.Value, newValue)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetNeg(newValue)
		if i.Payload[0] == 0xc6 {
			c.cycle(5, i.Offset, i.GetSize())
		} else {
			c.cycle(6, i.Offset, i.GetSize())
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
	case 0x85, 0x8d: // sta (zpg, abs)
		c.store(i.Value, c.builder.CreateLoad(c.rA, ""))
		if i.Payload[0] == 0x85 {
			c.cycle(3, i.Offset, i.GetSize())
		} else {
			c.cycle(4, i.Offset, i.GetSize())
		}
	case 0x86, 0x8e: // stx (zpg, abs)
		c.store(i.Value, c.builder.CreateLoad(c.rX, ""))
		if i.Payload[0] == 0x86 {
			c.cycle(3, i.Offset, i.GetSize())
		} else {
			c.cycle(4, i.Offset, i.GetSize())
		}
	case 0x84, 0x8c: // sty (zpg, abs)
		c.store(i.Value, c.builder.CreateLoad(c.rY, ""))
		if i.Payload[0] == 0x84 {
			c.cycle(3, i.Offset, i.GetSize())
		} else {
			c.cycle(4, i.Offset, i.GetSize())
		}
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s direct lacks Compile() implementation", i.OpName))
	}
}

func (i *IndirectXInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	c.Errors = append(c.Errors, "IndirectXInstruction lacks Compile() implementation")
}

func (i *IndirectYInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	switch i.Payload[0] {
	//case 0x71: // adc
	//case 0x31: // and
	//case 0xd1: // cmp
	//case 0x51: // eor
	//case 0xb1: // lda
	//case 0x11: // ora
	//case 0xf1: // sbc
	case 0x91: // sta
		ptrByte1 := c.load(i.Value)
		ptrByte2 := c.load(i.Value + 1)
		ptrByte1w := c.builder.CreateZExt(ptrByte1, llvm.Int16Type(), "")
		ptrByte2w := c.builder.CreateZExt(ptrByte2, llvm.Int16Type(), "")
		shiftAmt := llvm.ConstInt(llvm.Int16Type(), 8, false)
		word := c.builder.CreateShl(ptrByte2w, shiftAmt, "")
		word = c.builder.CreateOr(word, ptrByte1w, "")
		rY := c.builder.CreateLoad(c.rY, "")
		rYw := c.builder.CreateZExt(rY, llvm.Int16Type(), "")
		word = c.builder.CreateAdd(word, rYw, "")
		rA := c.builder.CreateLoad(c.rA, "")

		// debug statement, TODO remove this
		//c.debugPrintf("dyn store %04x\n", []llvm.Value{word})

		// runtime memory check
		staDoneBlock := c.createBlock("STA_done")
		x2000 := llvm.ConstInt(llvm.Int16Type(), 0x2000, false)
		inWRam := c.builder.CreateICmp(llvm.IntULT, word, x2000, "")
		notInWRamBlock := c.createIf(inWRam)
		// this generated code runs if the write is happening in the WRAM range
		maskedAddr := c.builder.CreateAnd(word, llvm.ConstInt(llvm.Int16Type(), 0x800-1, false), "")
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int16Type(), 0, false),
			maskedAddr,
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		c.builder.CreateStore(rA, ptr)
		c.builder.CreateBr(staDoneBlock)
		// this generated code runs if the write is > WRAM range
		c.selectBlock(notInWRamBlock)
		x4000 := llvm.ConstInt(llvm.Int16Type(), 0x4000, false)
		inPpuRam := c.builder.CreateICmp(llvm.IntULT, word, x4000, "")
		notInPpuRamBlock := c.createIf(inPpuRam)
		// this generated code runs if the write is in the PPU RAM range
		maskedAddr = c.builder.CreateAnd(word, llvm.ConstInt(llvm.Int16Type(), 0x8-1, false), "")
		badPpuAddrBlock := c.createBlock("BadPPUAddr")
		sw := c.builder.CreateSwitch(maskedAddr, badPpuAddrBlock, 7)
		// this generated code runs if the write is in a bad PPU RAM addr
		c.selectBlock(badPpuAddrBlock)
		c.createPanic()

		ppuCtrlBlock := c.createBlock("ppuctrl")
		sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0, false), ppuCtrlBlock)
		c.selectBlock(ppuCtrlBlock)
		c.builder.CreateCall(c.ppuCtrlFn, []llvm.Value{rA}, "")
		c.builder.CreateBr(staDoneBlock)

		ppuMaskBlock := c.createBlock("ppumask")
		sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 1, false), ppuMaskBlock)
		c.selectBlock(ppuMaskBlock)
		c.builder.CreateCall(c.ppuMaskFn, []llvm.Value{rA}, "")
		c.builder.CreateBr(staDoneBlock)

		oamAddrBlock := c.createBlock("oamaddr")
		sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 3, false), oamAddrBlock)
		c.selectBlock(oamAddrBlock)
		c.builder.CreateCall(c.oamAddrFn, []llvm.Value{rA}, "")
		c.builder.CreateBr(staDoneBlock)

		oamDataBlock := c.createBlock("oamdata")
		sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 4, false), oamDataBlock)
		c.selectBlock(oamDataBlock)
		c.builder.CreateCall(c.setOamDataFn, []llvm.Value{rA}, "")
		c.builder.CreateBr(staDoneBlock)

		ppuScrollBlock := c.createBlock("ppuscroll")
		sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 5, false), ppuScrollBlock)
		c.selectBlock(ppuScrollBlock)
		c.builder.CreateCall(c.setPpuScrollFn, []llvm.Value{rA}, "")
		c.builder.CreateBr(staDoneBlock)

		ppuAddrBlock := c.createBlock("ppuaddr")
		sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 6, false), ppuAddrBlock)
		c.selectBlock(ppuAddrBlock)
		c.builder.CreateCall(c.ppuAddrFn, []llvm.Value{rA}, "")
		c.builder.CreateBr(staDoneBlock)

		ppuDataBlock := c.createBlock("ppudata")
		sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 7, false), ppuDataBlock)
		c.selectBlock(ppuDataBlock)
		c.builder.CreateCall(c.setPpuDataFn, []llvm.Value{rA}, "")
		c.builder.CreateBr(staDoneBlock)

		// this generated code runs if the write is > PPU RAM range
		c.selectBlock(notInPpuRamBlock)
		c.createPanic()

		// done. X_X
		c.selectBlock(staDoneBlock)
		c.cycle(6, i.Offset, i.GetSize())

	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s ($%02x), Y lacks Compile() implementation", i.OpName, i.Value))
	}
}

func (i *IndirectInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	c.Errors = append(c.Errors, "IndirectInstruction lacks Compile() implementation")
}

func (s *LabeledStatement) Compile(c *Compilation) {
	// if we've already processed it as data, move on
	_, ok := c.labeledData[s.LabelName]
	if ok {
		return
	}

	bb := c.labeledBlocks[s.LabelName]
	if c.currentBlock != nil {
		c.builder.CreateBr(bb)
	}
	c.currentBlock = &bb
	c.builder.SetInsertPointAtEnd(bb)
}

func (s *LabeledStatement) CompileLabels(c *Compilation) {
	// if we've already processed it as data, move on
	_, ok := c.labeledData[s.LabelName]
	if ok {
		return
	}

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
		c.Errors = append(c.Errors, fmt.Sprintf("Missing 0x%04x entry point", addr))
		return
	}
	stmt, ok := n.(*DataWordStatement)
	if !ok {
		c.Errors = append(c.Errors, fmt.Sprintf("Entry point at 0x%04x must be a dc.w", addr))
		return
	}
	call, ok := stmt.dataList[0].(*LabelCall)
	if !ok {
		c.Errors = append(c.Errors, fmt.Sprintf("Entry point at 0x%04x must be a dc.w with a label", addr))
		return
	}
	*s = call.LabelName
}

func (c *Compilation) createReadChrFn(chrRom [][]byte) {
	//uint8_t rom_chr_bank_count;
	bankCountConst := llvm.ConstInt(llvm.Int8Type(), uint64(len(chrRom)), false)
	bankCountGlobal := llvm.AddGlobal(c.mod, bankCountConst.Type(), "rom_chr_bank_count")
	bankCountGlobal.SetLinkage(llvm.ExternalLinkage)
	bankCountGlobal.SetInitializer(bankCountConst)

	//uint8_t* rom_chr_data;
	dataLen := 0x2000 * len(chrRom)
	chrDataValues := make([]llvm.Value, 0, dataLen)
	int8type := llvm.Int8Type()
	for _, bank := range chrRom {
		for _, b := range bank {
			chrDataValues = append(chrDataValues, llvm.ConstInt(int8type, uint64(b), false))
		}
	}
	chrDataConst := llvm.ConstArray(llvm.ArrayType(llvm.Int8Type(), dataLen), chrDataValues)
	chrDataGlobal := llvm.AddGlobal(c.mod, chrDataConst.Type(), "rom_chr_data")
	chrDataGlobal.SetLinkage(llvm.PrivateLinkage)
	chrDataGlobal.SetInitializer(chrDataConst)
	// declare void @memcpy(void* dest, void* source, i32 size)
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	memcpyType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{bytePointerType, bytePointerType, llvm.Int32Type()}, false)
	memcpyFn := llvm.AddFunction(c.mod, "memcpy", memcpyType)
	memcpyFn.SetLinkage(llvm.ExternalLinkage)
	// void rom_read_chr(uint8_t* dest)
	readChrType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{bytePointerType}, false)
	readChrFn := llvm.AddFunction(c.mod, "rom_read_chr", readChrType)
	readChrFn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(readChrFn, "Entry")
	c.builder.SetInsertPointAtEnd(entry)
	if dataLen > 0 {
		x2000 := llvm.ConstInt(llvm.Int32Type(), uint64(dataLen), false)
		source := c.builder.CreatePointerCast(chrDataGlobal, bytePointerType, "")
		c.builder.CreateCall(memcpyFn, []llvm.Value{readChrFn.Param(0), source, x2000}, "")
	}
	c.builder.CreateRetVoid()
}

func (c *Compilation) createNamedGlobal(intType llvm.Type, name string) llvm.Value {
	val := llvm.ConstInt(intType, 0, false)
	glob := llvm.AddGlobal(c.mod, val.Type(), name)
	glob.SetLinkage(llvm.PrivateLinkage)
	glob.SetInitializer(val)
	return glob
}

func (c *Compilation) createByteRegister(name string) llvm.Value {
	return c.createNamedGlobal(llvm.Int8Type(), name)
}

func (c *Compilation) createWordRegister(name string) llvm.Value {
	return c.createNamedGlobal(llvm.Int16Type(), name)
}

func (c *Compilation) createBitRegister(name string) llvm.Value {
	return c.createNamedGlobal(llvm.Int1Type(), name)
}

func (c *Compilation) createFunctionDeclares() {
	// declare i32 @putchar(i32)
	putCharType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{llvm.Int32Type()}, false)
	c.putCharFn = llvm.AddFunction(c.mod, "putchar", putCharType)
	c.putCharFn.SetLinkage(llvm.ExternalLinkage)

	// declare i32 @printf(i8*, ...)
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	printfType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{bytePointerType}, true)
	c.printfFn = llvm.AddFunction(c.mod, "printf", printfType)
	c.printfFn.SetFunctionCallConv(llvm.CCallConv)
	c.printfFn.SetLinkage(llvm.ExternalLinkage)

	// declare void @exit(i32) noreturn nounwind
	exitType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int32Type()}, false)
	c.exitFn = llvm.AddFunction(c.mod, "exit", exitType)
	c.exitFn.AddFunctionAttr(llvm.NoReturnAttribute | llvm.NoUnwindAttribute)
	c.exitFn.SetLinkage(llvm.ExternalLinkage)

	// cycle should be called after every instruction with how many cycles the instruction took
	c.cycleFn = llvm.AddFunction(c.mod, "rom_cycle", llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false))
	c.cycleFn.SetLinkage(llvm.ExternalLinkage)

	c.ppuStatusFn = llvm.AddFunction(c.mod, "rom_ppustatus", llvm.FunctionType(llvm.Int8Type(), []llvm.Type{}, false))
	c.ppuStatusFn.SetLinkage(llvm.ExternalLinkage)

	c.ppuCtrlFn = llvm.AddFunction(c.mod, "rom_ppuctrl", llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false))
	c.ppuCtrlFn.SetLinkage(llvm.ExternalLinkage)

	c.ppuMaskFn = llvm.AddFunction(c.mod, "rom_ppumask", llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false))
	c.ppuMaskFn.SetLinkage(llvm.ExternalLinkage)

	c.ppuAddrFn = llvm.AddFunction(c.mod, "rom_ppuaddr", llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false))
	c.ppuAddrFn.SetLinkage(llvm.ExternalLinkage)

	c.setPpuDataFn = llvm.AddFunction(c.mod, "rom_setppudata", llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false))
	c.setPpuDataFn.SetLinkage(llvm.ExternalLinkage)

	c.oamAddrFn = llvm.AddFunction(c.mod, "rom_oamaddr", llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false))
	c.oamAddrFn.SetLinkage(llvm.ExternalLinkage)

	c.setOamDataFn = llvm.AddFunction(c.mod, "rom_setoamdata", llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false))
	c.setOamDataFn.SetLinkage(llvm.ExternalLinkage)

	c.setPpuScrollFn = llvm.AddFunction(c.mod, "rom_setppuscroll", llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false))
	c.setPpuScrollFn.SetLinkage(llvm.ExternalLinkage)
}

func (c *Compilation) createRegisters() {
	c.rX = c.createByteRegister("X")
	c.rY = c.createByteRegister("Y")
	c.rA = c.createByteRegister("A")
	c.rSP = c.createByteRegister("SP")
	c.rPC = c.createWordRegister("PC")
	c.rSNeg = c.createBitRegister("S_neg")
	c.rSOver = c.createBitRegister("S_over")
	c.rSBrk = c.createBitRegister("S_brk")
	c.rSDec = c.createBitRegister("S_dec")
	c.rSInt = c.createBitRegister("S_int")
	c.rSZero = c.createBitRegister("S_zero")
	c.rSCarry = c.createBitRegister("S_carry")
}

func (c *Compilation) addInterruptCode() {
	c.builder.SetInsertPointBefore(c.nmiBlock.FirstInstruction())
	// * push PC high onto stack
	pc := c.builder.CreateLoad(c.rPC, "")
	pcHigh16 := c.builder.CreateLShr(pc, llvm.ConstInt(llvm.Int16Type(), 8, false), "")
	pcHigh := c.builder.CreateTrunc(pcHigh16, llvm.Int8Type(), "")
	c.pushToStack(pcHigh)
	// * push PC low onto stack
	pcLow16 := c.builder.CreateAnd(pc, llvm.ConstInt(llvm.Int16Type(), 0xff, false), "")
	pcLow := c.builder.CreateTrunc(pcLow16, llvm.Int8Type(), "")
	c.pushToStack(pcLow)
	// * push processor status onto stack
	c.pushStatusReg()
}

func (c *Compilation) pullStatusReg() {
	status := c.pullFromStack()
	// and
	s7 := c.builder.CreateAnd(status, llvm.ConstInt(llvm.Int8Type(), 0x80, false), "")
	s6 := c.builder.CreateAnd(status, llvm.ConstInt(llvm.Int8Type(), 0x40, false), "")
	s4 := c.builder.CreateAnd(status, llvm.ConstInt(llvm.Int8Type(), 0x10, false), "")
	s3 := c.builder.CreateAnd(status, llvm.ConstInt(llvm.Int8Type(), 0x08, false), "")
	s2 := c.builder.CreateAnd(status, llvm.ConstInt(llvm.Int8Type(), 0x04, false), "")
	s1 := c.builder.CreateAnd(status, llvm.ConstInt(llvm.Int8Type(), 0x02, false), "")
	s0 := c.builder.CreateAnd(status, llvm.ConstInt(llvm.Int8Type(), 0x01, false), "")
	// icmp
	zero := llvm.ConstInt(llvm.Int8Type(), 0, false)
	s7 = c.builder.CreateICmp(llvm.IntNE, s7, zero, "")
	s6 = c.builder.CreateICmp(llvm.IntNE, s6, zero, "")
	s4 = c.builder.CreateICmp(llvm.IntNE, s4, zero, "")
	s3 = c.builder.CreateICmp(llvm.IntNE, s3, zero, "")
	s2 = c.builder.CreateICmp(llvm.IntNE, s2, zero, "")
	s1 = c.builder.CreateICmp(llvm.IntNE, s1, zero, "")
	s0 = c.builder.CreateICmp(llvm.IntNE, s0, zero, "")
	// store
	c.builder.CreateStore(s7, c.rSNeg)
	c.builder.CreateStore(s6, c.rSOver)
	c.builder.CreateStore(s4, c.rSBrk)
	c.builder.CreateStore(s3, c.rSDec)
	c.builder.CreateStore(s2, c.rSInt)
	c.builder.CreateStore(s1, c.rSZero)
	c.builder.CreateStore(s0, c.rSCarry)
}

func (c *Compilation) pushStatusReg() {
	// zextend
	s7z := c.builder.CreateZExt(c.builder.CreateLoad(c.rSNeg, ""), llvm.Int8Type(), "")
	s6z := c.builder.CreateZExt(c.builder.CreateLoad(c.rSOver, ""), llvm.Int8Type(), "")
	s4z := c.builder.CreateZExt(c.builder.CreateLoad(c.rSBrk, ""), llvm.Int8Type(), "")
	s3z := c.builder.CreateZExt(c.builder.CreateLoad(c.rSDec, ""), llvm.Int8Type(), "")
	s2z := c.builder.CreateZExt(c.builder.CreateLoad(c.rSInt, ""), llvm.Int8Type(), "")
	s1z := c.builder.CreateZExt(c.builder.CreateLoad(c.rSZero, ""), llvm.Int8Type(), "")
	s0z := c.builder.CreateZExt(c.builder.CreateLoad(c.rSCarry, ""), llvm.Int8Type(), "")
	// shift
	s7z = c.builder.CreateShl(s7z, llvm.ConstInt(llvm.Int8Type(), 7, false), "")
	s6z = c.builder.CreateShl(s6z, llvm.ConstInt(llvm.Int8Type(), 6, false), "")
	s4z = c.builder.CreateShl(s4z, llvm.ConstInt(llvm.Int8Type(), 4, false), "")
	s3z = c.builder.CreateShl(s3z, llvm.ConstInt(llvm.Int8Type(), 3, false), "")
	s2z = c.builder.CreateShl(s2z, llvm.ConstInt(llvm.Int8Type(), 2, false), "")
	s1z = c.builder.CreateShl(s1z, llvm.ConstInt(llvm.Int8Type(), 1, false), "")
	// or
	s0z = c.builder.CreateOr(s0z, s1z, "")
	s0z = c.builder.CreateOr(s0z, s2z, "")
	s0z = c.builder.CreateOr(s0z, s3z, "")
	s0z = c.builder.CreateOr(s0z, s4z, "")
	s0z = c.builder.CreateOr(s0z, s6z, "")
	s0z = c.builder.CreateOr(s0z, s7z, "")
	c.pushToStack(s0z)
}

func (p *Program) CompileToFile(file *os.File, flags CompileFlags) (*Compilation, error) {
	llvm.InitializeNativeTarget()

	c := new(Compilation)
	c.Flags = flags
	c.program = p
	c.mod = llvm.NewModule("asm_module")
	c.builder = llvm.NewBuilder()
	defer c.builder.Dispose()
	c.labeledData = map[string]llvm.Value{}
	c.labeledBlocks = map[string]llvm.BasicBlock{}

	// 2KB memory
	memType := llvm.ArrayType(llvm.Int8Type(), 0x800)
	c.wram = llvm.AddGlobal(c.mod, memType, "wram")
	c.wram.SetLinkage(llvm.PrivateLinkage)
	c.wram.SetInitializer(llvm.ConstNull(memType))

	//uint8_t rom_mirroring;
	mirroringConst := llvm.ConstInt(llvm.Int8Type(), uint64(p.Mirroring), false)
	mirroringGlobal := llvm.AddGlobal(c.mod, mirroringConst.Type(), "rom_mirroring")
	mirroringGlobal.SetLinkage(llvm.ExternalLinkage)
	mirroringGlobal.SetInitializer(mirroringConst)

	c.createReadChrFn(p.ChrRom)

	// runtime panic msg
	text := llvm.ConstString("panic: attempted to write to invalid memory address", false)
	c.runtimePanicMsg = llvm.AddGlobal(c.mod, text.Type(), "panicMsg")
	c.runtimePanicMsg.SetLinkage(llvm.PrivateLinkage)
	c.runtimePanicMsg.SetInitializer(text)

	// first pass to generate data declarations
	c.mode = dataStmtMode
	p.Ast.Ast(c)

	c.createFunctionDeclares()
	c.createRegisters()

	// main function / entry point
	mainType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false)
	c.mainFn = llvm.AddFunction(c.mod, "rom_start", mainType)
	c.mainFn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(c.mainFn, "Entry")

	// set up entry points
	c.setUpEntryPoint(p, 0xfffa, &c.nmiLabelName)
	c.setUpEntryPoint(p, 0xfffc, &c.resetLabelName)
	c.setUpEntryPoint(p, 0xfffe, &c.irqLabelName)

	// second pass to build basic blocks
	c.builder.SetInsertPointAtEnd(entry)
	c.mode = basicBlocksMode
	p.Ast.Ast(c)

	// finally, one last pass for codegen
	c.mode = compileMode
	p.Ast.Ast(c)

	// hook up entry points
	if c.nmiBlock == nil {
		c.Errors = append(c.Errors, "missing nmi entry point")
		return c, nil
	}
	if c.resetBlock == nil {
		c.Errors = append(c.Errors, "missing reset entry point")
		return c, nil
	}
	if c.irqBlock == nil {
		c.Errors = append(c.Errors, "missing irq entry point")
		return c, nil
	}

	// entry jump table
	c.selectBlock(entry)
	c.builder.SetInsertPointAtEnd(entry)
	badInterruptBlock := c.createBlock("BadInterrupt")
	sw := c.builder.CreateSwitch(c.mainFn.Param(0), badInterruptBlock, 3)
	c.selectBlock(badInterruptBlock)
	c.createPanic()
	sw.AddCase(llvm.ConstInt(llvm.Int8Type(), 1, false), *c.nmiBlock)
	sw.AddCase(llvm.ConstInt(llvm.Int8Type(), 2, false), *c.resetBlock)
	sw.AddCase(llvm.ConstInt(llvm.Int8Type(), 3, false), *c.irqBlock)

	c.addInterruptCode()


	if flags&DumpModulePreFlag != 0 {
		c.mod.Dump()
	}
	err := llvm.VerifyModule(c.mod, llvm.ReturnStatusAction)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return c, nil
	}

	engine, err := llvm.NewJITCompiler(c.mod, 3)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return c, nil
	}
	defer engine.Dispose()

	if flags&DisableOptFlag == 0 {
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

	if flags&DumpModuleFlag != 0 {
		c.mod.Dump()
	}

	err = llvm.WriteBitcodeToFile(c.mod, file)

	if err != nil {
		return c, err
	}

	return c, nil
}

func (p *Program) CompileToFilename(filename string, flags CompileFlags) (*Compilation, error) {
	fd, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	c, err := p.CompileToFile(fd, flags)
	err2 := fd.Close()

	if err != nil {
		return nil, err
	}
	if err2 != nil {
		return nil, err2
	}
	return c, nil
}
