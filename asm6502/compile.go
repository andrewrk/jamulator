package asm6502

// TODO: handle interrupts
// TODO: load/save state

import (
	"bytes"
	"fmt"
	"github.com/axw/gollvm/llvm"
	"os"
)

type Compilation struct {
	Warnings []string
	Errors   []string

	program         *Program
	mod             llvm.Module
	builder         llvm.Builder
	wram            llvm.Value // 2KB WRAM
	rX              llvm.Value // X index register
	rY              llvm.Value // Y index register
	rA              llvm.Value // accumulator
	rSP             llvm.Value // stack pointer
	rSNeg           llvm.Value // whether the last arithmetic result is negative
	rSZero          llvm.Value // whether the last arithmetic result is zero
	rSDec           llvm.Value // decimal
	rSInt           llvm.Value // interrupt disable
	runtimePanicMsg llvm.Value // we print this when a runtime error occurs

	// ABI
	mainFn         llvm.Value
	putsFn         llvm.Value
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
)

func (c *Compilation) dataStop() {
	if len(c.currentLabel) == 0 {
		return
	}
	if c.currentValue.Len() == 0 {
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
			c.builder.CreateCall(c.ppuCtrlFn, []llvm.Value{i8}, "")
		case 1: // ppumask
			c.builder.CreateCall(c.ppuMaskFn, []llvm.Value{i8}, "")
		case 3: // oamaddr
			c.builder.CreateCall(c.oamAddrFn, []llvm.Value{i8}, "")
		case 4: // oamdata
			c.builder.CreateCall(c.setOamDataFn, []llvm.Value{i8}, "")
		case 5: // ppuscroll
			c.builder.CreateCall(c.setPpuScrollFn, []llvm.Value{i8}, "")
		case 6: // ppuaddr
			c.builder.CreateCall(c.ppuAddrFn, []llvm.Value{i8}, "")
		case 7: // ppudata
			c.builder.CreateCall(c.setPpuDataFn, []llvm.Value{i8}, "")
		default:
			panic("unreachable")
		}
	}

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
		return c.builder.CreateLoad(ptr, "")
	case 0x2000 <= addr && addr < 0x4000:
		// PPU registers. mask because mirrored
		switch addr & (0x8 - 1) {
		case 2:
			return c.builder.CreateCall(c.ppuStatusFn, []llvm.Value{}, "")
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
	c.builder.CreateCall(c.putsFn, []llvm.Value{ptr}, "")
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

func (c *Compilation) cycle(count int) {
	v := llvm.ConstInt(llvm.Int8Type(), uint64(count), false)
	c.builder.CreateCall(c.cycleFn, []llvm.Value{v}, "")
}

func (i *ImmediateInstruction) Compile(c *Compilation) {
	v := llvm.ConstInt(llvm.Int8Type(), uint64(i.Value), false)
	switch i.OpCode {
	case 0xa2: // ldx
		c.builder.CreateStore(v, c.rX)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2)
	case 0xa0: // ldy
		c.builder.CreateStore(v, c.rY)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2)
	case 0xa9: // lda
		c.builder.CreateStore(v, c.rA)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2)
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
		c.cycle(2)
	case 0x58: // cli
		c.clearInt()
		c.cycle(2)
	//case 0xb8: // clv
	case 0xca: // dex
		c.increment(c.rX, -1)
		c.cycle(2)
	case 0x88: // dey
		c.increment(c.rY, -1)
		c.cycle(2)
	case 0xe8: // inx
		c.increment(c.rX, 1)
		c.cycle(2)
	case 0xc8: // iny
		c.increment(c.rY, 1)
		c.cycle(2)
	//case 0x4a: // lsr
	case 0xea: // nop
		c.cycle(2)
	//case 0x48: // pha
	//case 0x08: // php
	//case 0x68: // pla
	//case 0x28: // plp
	//case 0x2a: // rol
	//case 0x6a: // ror
	case 0x40: // rti
		c.Warnings = append(c.Warnings, "interrupts not supported - ignoring RTI instruction")
		c.builder.CreateUnreachable()
		//c.cycle(6)
	//case 0x60: // rts
	//case 0x38: // sec
	case 0xf8: // sed
		c.setDec()
		c.cycle(2)
	case 0x78: // sei
		c.setInt()
		c.cycle(2)
	case 0xaa: // tax
		c.transfer(c.rA, c.rX)
		c.cycle(2)
	case 0xa8: // tay
		c.transfer(c.rA, c.rY)
		c.cycle(2)
	case 0xba: // tsx
		c.transfer(c.rSP, c.rX)
		c.cycle(2)
	case 0x8a: // txa
		c.transfer(c.rX, c.rA)
		c.cycle(2)
	case 0x9a: // txs
		c.transfer(c.rX, c.rSP)
		c.cycle(2)
	case 0x98: // tya
		c.transfer(c.rY, c.rA)
		c.cycle(2)
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

		cycleCount := 4
		addr := c.program.Labels[i.LabelName]
		if addr > 0xff {
			cycleCount += 1
		}
		c.cycle(cycleCount)
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
		// branch instruction - cycle before execution
		c.cycle(3)
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
		// we put the cycle before the execution because
		// it's a branch...
		c.cycle(2)
		thenBlock := c.labeledBlocks[i.LabelName]
		elseBlock := c.createBlock("else")
		isZero := c.builder.CreateLoad(c.rSZero, "")
		c.builder.CreateCondBr(isZero, thenBlock, elseBlock)
		c.selectBlock(elseBlock)
	//case 0x90: // bcc
	//case 0xb0: // bcs
	case 0x30: // bmi
		c.cycle(2)
		thenBlock := c.labeledBlocks[i.LabelName]
		elseBlock := c.createBlock("else")
		isNeg := c.builder.CreateLoad(c.rSNeg, "")
		c.builder.CreateCondBr(isNeg, thenBlock, elseBlock)
		c.selectBlock(elseBlock)
	case 0xd0: // bne
		c.cycle(2)
		thenBlock := c.createBlock("then")
		elseBlock := c.labeledBlocks[i.LabelName]
		isZero := c.builder.CreateLoad(c.rSZero, "")
		c.builder.CreateCondBr(isZero, thenBlock, elseBlock)
		c.selectBlock(thenBlock)
	case 0x10: // bpl
		c.cycle(2)
		thenBlock := c.createBlock("then")
		elseBlock := c.labeledBlocks[i.LabelName]
		isNeg := c.builder.CreateLoad(c.rSNeg, "")
		c.builder.CreateCondBr(isNeg, thenBlock, elseBlock)
		c.selectBlock(thenBlock)
	//case 0x50: // bvc
	//case 0x70: // bvs
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s <label> lacks Compile() implementation", i.OpName))
	}
}

func (i *DirectInstruction) Compile(c *Compilation) {
	switch i.Payload[0] {
	case 0xa5, 0xad: // lda (zpg, abs)
		v := c.load(i.Value)
		c.builder.CreateStore(v, c.rA)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		if i.Payload[0] == 0xa5 {
			c.cycle(3)
		} else {
			c.cycle(4)
		}
	case 0xc6, 0xce: // dec (zpg, abs)
		oldValue := c.load(i.Value)
		c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
		newValue := c.builder.CreateSub(oldValue, c1, "")
		c.store(i.Value, newValue)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetNeg(newValue)
		if i.Payload[0] == 0xc6 {
			c.cycle(5)
		} else {
			c.cycle(6)
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
			c.cycle(3)
		} else {
			c.cycle(4)
		}
	case 0x86, 0x8e: // stx (zpg, abs)
		c.store(i.Value, c.builder.CreateLoad(c.rX, ""))
		if i.Payload[0] == 0x86 {
			c.cycle(3)
		} else {
			c.cycle(4)
		}
	case 0x84, 0x8c: // sty (zpg, abs)
		c.store(i.Value, c.builder.CreateLoad(c.rY, ""))
		if i.Payload[0] == 0x84 {
			c.cycle(3)
		} else {
			c.cycle(4)
		}
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s direct lacks Compile() implementation", i.OpName))
	}
}

func (i *IndirectXInstruction) Compile(c *Compilation) {
	c.Errors = append(c.Errors, "IndirectXInstruction lacks Compile() implementation")
}

func (i *IndirectYInstruction) Compile(c *Compilation) {
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
		c.cycle(6)

	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s ($%02x), Y lacks Compile() implementation", i.OpName, i.Value))
	}
}

func (i *IndirectInstruction) Compile(c *Compilation) {
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
	c.labeledData = map[string]llvm.Value{}
	c.labeledBlocks = map[string]llvm.BasicBlock{}

	// 2KB memory
	memType := llvm.ArrayType(llvm.Int8Type(), 0x800)
	c.wram = llvm.AddGlobal(c.mod, memType, "wram")
	c.wram.SetLinkage(llvm.PrivateLinkage)
	c.wram.SetInitializer(llvm.ConstNull(memType))

	// runtime panic msg
	text := llvm.ConstString("panic: attempted to write to invalid memory address", false)
	c.runtimePanicMsg = llvm.AddGlobal(c.mod, text.Type(), "panicMsg")
	c.runtimePanicMsg.SetLinkage(llvm.PrivateLinkage)
	c.runtimePanicMsg.SetInitializer(text)

	// first pass to generate data declarations
	c.mode = dataStmtMode
	p.Ast.Ast(c)

	// declare i32 @putchar(i32)
	putCharType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{llvm.Int32Type()}, false)
	c.putCharFn = llvm.AddFunction(c.mod, "putchar", putCharType)
	c.putCharFn.SetLinkage(llvm.ExternalLinkage)

	// declare i32 @puts(i8*)
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	putsType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{bytePointerType}, false)
	c.putsFn = llvm.AddFunction(c.mod, "puts", putsType)
	c.putsFn.SetFunctionCallConv(llvm.CCallConv)
	c.putsFn.SetLinkage(llvm.ExternalLinkage)

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

	// main function / entry point
	mainType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
	c.mainFn = llvm.AddFunction(c.mod, "rom_start", mainType)
	c.mainFn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(c.mainFn, "Entry")
	c.builder.SetInsertPointAtEnd(entry)
	c.rX = c.builder.CreateAlloca(llvm.Int8Type(), "X")
	c.rY = c.builder.CreateAlloca(llvm.Int8Type(), "Y")
	c.rA = c.builder.CreateAlloca(llvm.Int8Type(), "A")
	c.rSP = c.builder.CreateAlloca(llvm.Int8Type(), "SP")
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

	if flags&DumpModulePreFlag != 0 {
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
