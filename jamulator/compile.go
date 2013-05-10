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

	labeledData   map[string]llvm.Value
	labeledBlocks map[string]llvm.BasicBlock
	// used for the entry jump table so we can do JSR
	labelIds        map[string]int
	entryLabelCount int

	currentValue *bytes.Buffer
	currentLabel string
	mode         int
	currentBlock *llvm.BasicBlock
	// label names to look for
	nmiLabelName   string
	resetLabelName string
	irqLabelName   string
	nmiBlock       *llvm.BasicBlock
	resetBlock     *llvm.BasicBlock
	irqBlock       *llvm.BasicBlock

	// ABI
	mainFn    llvm.Value
	printfFn  llvm.Value
	putCharFn llvm.Value
	exitFn    llvm.Value
	cycleFn   llvm.Value
	// PPU
	ppuReadStatusFn    llvm.Value
	ppuReadOamDataFn    llvm.Value
	ppuReadDataFn    llvm.Value
	ppuCtrlFn      llvm.Value
	ppuMaskFn      llvm.Value
	ppuAddrFn      llvm.Value
	setPpuDataFn   llvm.Value
	oamAddrFn      llvm.Value
	setOamDataFn   llvm.Value
	setPpuScrollFn llvm.Value
	ppuWriteDma    llvm.Value
	// APU
	apuWriteSquare1CtrlFn      llvm.Value
	apuWriteSquare1SweepsFn    llvm.Value
	apuWriteSquare1LowFn       llvm.Value
	apuWriteSquare1HighFn      llvm.Value
	apuWriteSquare2CtrlFn      llvm.Value
	apuWriteSquare2SweepsFn    llvm.Value
	apuWriteSquare2LowFn       llvm.Value
	apuWriteSquare2HighFn      llvm.Value
	apuWriteTriangleCtrlFn     llvm.Value
	apuWriteTriangleLowFn      llvm.Value
	apuWriteTriangleHighFn     llvm.Value
	apuWriteNoiseBaseFn        llvm.Value
	apuWriteNoisePeriodFn      llvm.Value
	apuWriteNoiseLengthFn      llvm.Value
	apuWriteDmcFlagsFn         llvm.Value
	apuWriteDmcDirectLoadFn    llvm.Value
	apuWriteDmcSampleAddressFn llvm.Value
	apuWriteDmcSampleLengthFn  llvm.Value
	apuWriteCtrlFlags1Fn       llvm.Value
	apuWriteCtrlFlags2Fn       llvm.Value
	// pads
	padWrite1Fn llvm.Value
	padWrite2Fn llvm.Value
	padRead1Fn llvm.Value
	padRead2Fn llvm.Value
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
		c.currentLabel = ""
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

func (c *Compilation) setCarry() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 1, false), c.rSCarry)
}

func (c *Compilation) clearCarry() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 0, false), c.rSCarry)
}

func (c *Compilation) clearOverflow() {
	c.builder.CreateStore(llvm.ConstInt(llvm.Int1Type(), 0, false), c.rSOver)
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

func (c *Compilation) dynTestAndSetCarryLShr(val llvm.Value) {
	masked := c.builder.CreateAnd(val, llvm.ConstInt(llvm.Int8Type(), 0x1, false), "")
	c0 := llvm.ConstInt(llvm.Int8Type(), 0, false)
	isCarry := c.builder.CreateICmp(llvm.IntNE, masked, c0, "")
	c.builder.CreateStore(isCarry, c.rSCarry)
}

func (c *Compilation) dynTestAndSetCarryShl(val llvm.Value) {
	c0 := llvm.ConstInt(llvm.Int8Type(), 0, false)
	x80 := llvm.ConstInt(llvm.Int8Type(), 0x80, false)
	masked := c.builder.CreateAnd(val, x80, "")
	isCarry := c.builder.CreateICmp(llvm.IntNE, masked, c0, "")
	c.builder.CreateStore(isCarry, c.rSCarry)
}

func (c *Compilation) dynTestAndSetCarrySubtraction(left llvm.Value, right llvm.Value) {
	// set the carry bit if result is positive or zero
	isCarry := c.builder.CreateICmp(llvm.IntUGE, left, right, "")
	c.builder.CreateStore(isCarry, c.rSCarry)
}

func (c *Compilation) dynTestAndSetCarrySubtraction3(a llvm.Value, v llvm.Value, carry llvm.Value) {
	// set the carry bit if result is positive or zero
	a32 := c.builder.CreateZExt(a, llvm.Int32Type(), "")
	carry32 := c.builder.CreateZExt(carry, llvm.Int32Type(), "")
	v32 := c.builder.CreateZExt(v, llvm.Int32Type(), "")
	aMinusV32 := c.builder.CreateSub(a32, v32, "")
	newA32 := c.builder.CreateSub(aMinusV32, carry32, "")
	c0 := llvm.ConstInt(llvm.Int32Type(), 0, false)
	isCarry := c.builder.CreateICmp(llvm.IntSGE, newA32, c0, "")
	c.builder.CreateStore(isCarry, c.rSCarry)
}

func (c *Compilation) performRor(val llvm.Value) llvm.Value {
	c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
	c7 := llvm.ConstInt(llvm.Int8Type(), 7, false)
	shifted := c.builder.CreateLShr(val, c1, "")
	carryBit := c.builder.CreateLoad(c.rSCarry, "")
	carry := c.builder.CreateZExt(carryBit, llvm.Int8Type(), "")
	carryShifted := c.builder.CreateShl(carry, c7, "")
	newValue := c.builder.CreateAnd(shifted, carryShifted, "")
	c.dynTestAndSetZero(newValue)
	c.dynTestAndSetNeg(newValue)
	c.dynTestAndSetCarryLShr(val)
	return newValue
}

func (c *Compilation) performRol(val llvm.Value) llvm.Value {
	c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
	shifted := c.builder.CreateShl(val, c1, "")
	carryBit := c.builder.CreateLoad(c.rSCarry, "")
	carry := c.builder.CreateZExt(carryBit, llvm.Int8Type(), "")
	newValue := c.builder.CreateAnd(shifted, carry, "")
	c.dynTestAndSetZero(newValue)
	c.dynTestAndSetNeg(newValue)
	c.dynTestAndSetCarryShl(val)
	return newValue
}

func (c *Compilation) performAsl(val llvm.Value) llvm.Value {
	c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
	newValue := c.builder.CreateShl(val, c1, "")
	c.dynTestAndSetZero(newValue)
	c.dynTestAndSetNeg(newValue)
	c.dynTestAndSetCarryShl(val)
	return newValue
}

func (c *Compilation) performAdc(val llvm.Value) {
	a := c.builder.CreateLoad(c.rA, "")
	aPlusV := c.builder.CreateAdd(a, val, "")
	carryBit := c.builder.CreateLoad(c.rSCarry, "")
	carry := c.builder.CreateZExt(carryBit, llvm.Int8Type(), "")
	newA := c.builder.CreateAdd(aPlusV, carry, "")
	c.builder.CreateStore(newA, c.rA)
	c.dynTestAndSetNeg(newA)
	c.dynTestAndSetZero(newA)
	c.dynTestAndSetOverflowAddition(a, val, newA)
	c.dynTestAndSetCarryAddition(a, val, carry)
}

func (c *Compilation) performSbc(val llvm.Value) {
	a := c.builder.CreateLoad(c.rA, "")
	aMinusV := c.builder.CreateSub(a, val, "")
	carryBit := c.builder.CreateLoad(c.rSCarry, "")
	carry := c.builder.CreateZExt(carryBit, llvm.Int8Type(), "")
	newA := c.builder.CreateSub(aMinusV, carry, "")
	c.builder.CreateStore(newA, c.rA)
	c.dynTestAndSetNeg(newA)
	c.dynTestAndSetZero(newA)
	c.dynTestAndSetOverflowSubtraction(a, val, carry)
	c.dynTestAndSetCarrySubtraction3(a, val, carry)
}

func (c *Compilation) performBit(val llvm.Value) {
	a := c.builder.CreateLoad(c.rA, "")
	c0 := llvm.ConstInt(llvm.Int8Type(), 0, false)
	x40 := llvm.ConstInt(llvm.Int8Type(), 0x40, false)
	x80 := llvm.ConstInt(llvm.Int8Type(), 0x80, false)

	anded := c.builder.CreateAnd(val, a, "")
	isZero := c.builder.CreateICmp(llvm.IntEQ, anded, c0, "")
	c.builder.CreateStore(isZero, c.rSZero)

	maskedX80 := c.builder.CreateAnd(val, x80, "")
	isNeg := c.builder.CreateICmp(llvm.IntNE, maskedX80, c0, "")
	c.builder.CreateStore(isNeg, c.rSNeg)

	maskedX40 := c.builder.CreateAnd(val, x40, "")
	isOver := c.builder.CreateICmp(llvm.IntNE, maskedX40, c0, "")
	c.builder.CreateStore(isOver, c.rSOver)
}

func (c *Compilation) dynStore(addr llvm.Value, minAddr int, maxAddr int, val llvm.Value) {
	if minAddr != 0 || maxAddr != 0xffff {
		c.Warnings = append(c.Warnings, "TODO: dynStore is unoptimized")
	}

	// runtime memory check
	storeDoneBlock := c.createBlock("StoreDone")
	x2000 := llvm.ConstInt(llvm.Int16Type(), 0x2000, false)
	inWRam := c.builder.CreateICmp(llvm.IntULT, addr, x2000, "")
	notInWRamBlock := c.createIf(inWRam)
	// this generated code runs if the write is happening in the WRAM range
	maskedAddr := c.builder.CreateAnd(addr, llvm.ConstInt(llvm.Int16Type(), 0x800-1, false), "")
	indexes := []llvm.Value{
		llvm.ConstInt(llvm.Int16Type(), 0, false),
		maskedAddr,
	}
	ptr := c.builder.CreateGEP(c.wram, indexes, "")
	c.builder.CreateStore(val, ptr)
	c.builder.CreateBr(storeDoneBlock)
	// this generated code runs if the write is > WRAM range
	c.selectBlock(notInWRamBlock)
	x4000 := llvm.ConstInt(llvm.Int16Type(), 0x4000, false)
	inPpuRam := c.builder.CreateICmp(llvm.IntULT, addr, x4000, "")
	notInPpuRamBlock := c.createIf(inPpuRam)
	// this generated code runs if the write is in the PPU RAM range
	maskedAddr = c.builder.CreateAnd(addr, llvm.ConstInt(llvm.Int16Type(), 0x8-1, false), "")
	badPpuAddrBlock := c.createBlock("BadPPUAddr")
	sw := c.builder.CreateSwitch(maskedAddr, badPpuAddrBlock, 7)
	// this generated code runs if the write is in a bad PPU RAM addr
	c.selectBlock(badPpuAddrBlock)
	c.createPanic()

	ppuCtrlBlock := c.createBlock("ppuctrl")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0, false), ppuCtrlBlock)
	c.selectBlock(ppuCtrlBlock)
	c.builder.CreateCall(c.ppuCtrlFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuMaskBlock := c.createBlock("ppumask")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 1, false), ppuMaskBlock)
	c.selectBlock(ppuMaskBlock)
	c.builder.CreateCall(c.ppuMaskFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	oamAddrBlock := c.createBlock("oamaddr")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 3, false), oamAddrBlock)
	c.selectBlock(oamAddrBlock)
	c.builder.CreateCall(c.oamAddrFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	oamDataBlock := c.createBlock("oamdata")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 4, false), oamDataBlock)
	c.selectBlock(oamDataBlock)
	c.builder.CreateCall(c.setOamDataFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuScrollBlock := c.createBlock("ppuscroll")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 5, false), ppuScrollBlock)
	c.selectBlock(ppuScrollBlock)
	c.builder.CreateCall(c.setPpuScrollFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuAddrBlock := c.createBlock("ppuaddr")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 6, false), ppuAddrBlock)
	c.selectBlock(ppuAddrBlock)
	c.builder.CreateCall(c.ppuAddrFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuDataBlock := c.createBlock("ppudata")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 7, false), ppuDataBlock)
	c.selectBlock(ppuDataBlock)
	c.builder.CreateCall(c.setPpuDataFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	// this generated code runs if the write is > PPU RAM range
	c.selectBlock(notInPpuRamBlock)
	c.createPanic()

	// done. X_X
	c.selectBlock(storeDoneBlock)
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
		ptr := c.wramPtr(addr)
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
	case 0x4000 <= addr && addr <= 0x4017:
		switch addr {
		default:
			c.Errors = append(c.Errors, fmt.Sprintf("writing to memory address 0x%04x is unsupported", addr))
		case 0x4000:
			c.builder.CreateCall(c.apuWriteSquare1CtrlFn, []llvm.Value{i8}, "")
		case 0x4001:
			c.builder.CreateCall(c.apuWriteSquare1SweepsFn, []llvm.Value{i8}, "")
		case 0x4002:
			c.builder.CreateCall(c.apuWriteSquare1LowFn, []llvm.Value{i8}, "")
		case 0x4003:
			c.builder.CreateCall(c.apuWriteSquare1HighFn, []llvm.Value{i8}, "")
		case 0x4004:
			c.builder.CreateCall(c.apuWriteSquare2CtrlFn, []llvm.Value{i8}, "")
		case 0x4005:
			c.builder.CreateCall(c.apuWriteSquare2SweepsFn, []llvm.Value{i8}, "")
		case 0x4006:
			c.builder.CreateCall(c.apuWriteSquare2LowFn, []llvm.Value{i8}, "")
		case 0x4007:
			c.builder.CreateCall(c.apuWriteSquare2HighFn, []llvm.Value{i8}, "")
		case 0x4008:
			c.builder.CreateCall(c.apuWriteTriangleCtrlFn, []llvm.Value{i8}, "")
		case 0x400a:
			c.builder.CreateCall(c.apuWriteTriangleLowFn, []llvm.Value{i8}, "")
		case 0x400b:
			c.builder.CreateCall(c.apuWriteTriangleHighFn, []llvm.Value{i8}, "")
		case 0x400c:
			c.builder.CreateCall(c.apuWriteNoiseBaseFn, []llvm.Value{i8}, "")
		case 0x400e:
			c.builder.CreateCall(c.apuWriteNoisePeriodFn, []llvm.Value{i8}, "")
		case 0x400f:
			c.builder.CreateCall(c.apuWriteNoiseLengthFn, []llvm.Value{i8}, "")
		case 0x4010:
			c.builder.CreateCall(c.apuWriteDmcFlagsFn, []llvm.Value{i8}, "")
		case 0x4011:
			c.builder.CreateCall(c.apuWriteDmcDirectLoadFn, []llvm.Value{i8}, "")
		case 0x4012:
			c.builder.CreateCall(c.apuWriteDmcSampleAddressFn, []llvm.Value{i8}, "")
		case 0x4013:
			c.builder.CreateCall(c.apuWriteDmcSampleLengthFn, []llvm.Value{i8}, "")
		case 0x4014:
			c.builder.CreateCall(c.ppuWriteDma, []llvm.Value{i8}, "")
		case 0x4015:
			c.builder.CreateCall(c.apuWriteCtrlFlags1Fn, []llvm.Value{i8}, "")
		case 0x4016:
			c.builder.CreateCall(c.padWrite1Fn, []llvm.Value{i8}, "")
		case 0x4017:
			c.builder.CreateCall(c.padWrite2Fn, []llvm.Value{i8}, "")
			c.builder.CreateCall(c.apuWriteCtrlFlags2Fn, []llvm.Value{i8}, "")
		}
	}

}

func (c *Compilation) dynLoad(addr llvm.Value, minAddr int, maxAddr int) llvm.Value {
	// returns the byte at addr, with runtime checks for the range between minAddr and maxAddr
	// currently only can do WRAM stuff
	if maxAddr < 0x0800 {
		// no runtime checks needed.
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int8Type(), 0, false),
			addr,
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		return c.builder.CreateLoad(ptr, "")
	}
	if maxAddr < 0x2000 {
		// address masking needed, but it's definitely in WRAM
		maskedAddr := c.builder.CreateAnd(addr, llvm.ConstInt(llvm.Int16Type(), 0x800-1, false), "")
		indexes := []llvm.Value{
			llvm.ConstInt(llvm.Int8Type(), 0, false),
			maskedAddr,
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		return c.builder.CreateLoad(ptr, "")
	}
	if minAddr != 0 || maxAddr != 0xffff {
		c.Warnings = append(c.Warnings, "TODO: dynLoad is unoptimized")
	}
	result := c.builder.CreateAlloca(llvm.Int8Type(), "load_result")
	loadDoneBlock := c.createBlock("LoadDone")
	x2000 := llvm.ConstInt(llvm.Int16Type(), 0x2000, false)
	inWRam := c.builder.CreateICmp(llvm.IntULT, addr, x2000, "")
	notInWRamBlock := c.createIf(inWRam)
	// this generated code runs if the write is happening in the WRAM range
	maskedAddr := c.builder.CreateAnd(addr, llvm.ConstInt(llvm.Int16Type(), 0x800-1, false), "")
	indexes := []llvm.Value{
		llvm.ConstInt(llvm.Int16Type(), 0, false),
		maskedAddr,
	}
	ptr := c.builder.CreateGEP(c.wram, indexes, "")
	v := c.builder.CreateLoad(ptr, "")
	c.builder.CreateStore(v, result)
	c.builder.CreateBr(loadDoneBlock)
	// this generated code runs if the write is > WRAM range
	c.selectBlock(notInWRamBlock)
	x4000 := llvm.ConstInt(llvm.Int16Type(), 0x4000, false)
	inPpuRam := c.builder.CreateICmp(llvm.IntULT, addr, x4000, "")
	notInPpuRamBlock := c.createIf(inPpuRam)
	// this generated code runs if the write is in the PPU RAM range
	maskedAddr = c.builder.CreateAnd(addr, llvm.ConstInt(llvm.Int16Type(), 0x8-1, false), "")
	badPpuAddrBlock := c.createBlock("BadPPUAddr")
	sw := c.builder.CreateSwitch(maskedAddr, badPpuAddrBlock, 3)
	// this generated code runs if the write is in a bad PPU RAM addr
	c.selectBlock(badPpuAddrBlock)
	c.createPanic()

	ppuReadStatusBlock := c.createBlock("ppu_read_status")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 2, false), ppuReadStatusBlock)
	c.selectBlock(ppuReadStatusBlock)
	v = c.builder.CreateCall(c.ppuReadStatusFn, []llvm.Value{}, "")
	c.builder.CreateStore(v, result)
	c.builder.CreateBr(loadDoneBlock)

	ppuReadOamDataBlock := c.createBlock("ppu_read_oamdata")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 4, false), ppuReadOamDataBlock)
	c.selectBlock(ppuReadOamDataBlock)
	v = c.builder.CreateCall(c.ppuReadOamDataFn, []llvm.Value{}, "")
	c.builder.CreateStore(v, result)
	c.builder.CreateBr(loadDoneBlock)

	ppuReadDataBlock := c.createBlock("ppu_read_data")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 7, false), ppuReadDataBlock)
	c.selectBlock(ppuReadDataBlock)
	v = c.builder.CreateCall(c.ppuReadDataFn, []llvm.Value{}, "")
	c.builder.CreateStore(v, result)
	c.builder.CreateBr(loadDoneBlock)

	// this generated code runs if the write is > PPU RAM range
	c.selectBlock(notInPpuRamBlock)
	c.createPanic()

	// done. X_X
	c.selectBlock(loadDoneBlock)
	return c.builder.CreateLoad(result, "")
}

func (c *Compilation) wramPtr(addr int) llvm.Value {
	// 2KB working RAM. mask because mirrored
	if addr < 0 || addr >= 0x2000 {
		c.Errors = append(c.Errors, fmt.Sprintf("$%04x is not in wram", addr))
	}
	maskedAddr := addr & (0x800 - 1)
	indexes := []llvm.Value{
		llvm.ConstInt(llvm.Int8Type(), 0, false),
		llvm.ConstInt(llvm.Int8Type(), uint64(maskedAddr), false),
	}
	return c.builder.CreateGEP(c.wram, indexes, "")
}

func (c *Compilation) load(addr int) llvm.Value {
	switch {
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("reading from $%04x not implemented", addr))
		return llvm.ConstNull(llvm.Int8Type())
	case 0x0000 <= addr && addr < 0x2000:
		ptr := c.wramPtr(addr)
		v := c.builder.CreateLoad(ptr, "")
		return v
	case 0x2000 <= addr && addr < 0x4000:
		// PPU registers. mask because mirrored
		switch addr & (0x8 - 1) {
		case 2:
			c.debugPrint("ppu_read_status\n")
			return c.builder.CreateCall(c.ppuReadStatusFn, []llvm.Value{}, "")
		case 4:
			c.debugPrint("ppu_read_oamdata\n")
			return c.builder.CreateCall(c.ppuReadOamDataFn, []llvm.Value{}, "")
		case 7:
			c.debugPrint("ppu_read_data\n")
			return c.builder.CreateCall(c.ppuReadDataFn, []llvm.Value{}, "")
		default:
			c.Errors = append(c.Errors, fmt.Sprintf("reading from $%04x not implemented", addr))
			return llvm.ConstNull(llvm.Int8Type())
		}
	case addr == 0x4016:
		return c.builder.CreateCall(c.padRead1Fn, []llvm.Value{}, "")
	case addr == 0x4017:
		return c.builder.CreateCall(c.padRead2Fn, []llvm.Value{}, "")
	}
	panic("unreachable")
}

// loads a little endian word
func (c *Compilation) loadWord(addr int) llvm.Value {
	ptrByte1 := c.load(addr)
	ptrByte2 := c.load(addr + 1)
	ptrByte1w := c.builder.CreateZExt(ptrByte1, llvm.Int16Type(), "")
	ptrByte2w := c.builder.CreateZExt(ptrByte2, llvm.Int16Type(), "")
	shiftAmt := llvm.ConstInt(llvm.Int16Type(), 8, false)
	word := c.builder.CreateShl(ptrByte2w, shiftAmt, "")
	return c.builder.CreateOr(word, ptrByte1w, "")
}

func (c *Compilation) incrementVal(v llvm.Value, delta int) llvm.Value {
	if delta < 0 {
		c1 := llvm.ConstInt(llvm.Int8Type(), uint64(-delta), false)
		return c.builder.CreateSub(v, c1, "")
	}
	c1 := llvm.ConstInt(llvm.Int8Type(), uint64(delta), false)
	return c.builder.CreateAdd(v, c1, "")
}

func (c *Compilation) incrementMem(addr int, delta int) {
	oldValue := c.load(addr)
	newValue := c.incrementVal(oldValue, delta)
	c.store(addr, newValue)
	c.dynTestAndSetZero(newValue)
	c.dynTestAndSetNeg(newValue)
}

func (c *Compilation) increment(ptr llvm.Value, delta int) {
	oldValue := c.builder.CreateLoad(ptr, "")
	newValue := c.incrementVal(oldValue, delta)
	c.builder.CreateStore(newValue, ptr)
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

func (c *Compilation) pullWordFromStack() llvm.Value {
	low := c.pullFromStack()
	high := c.pullFromStack()
	low16 := c.builder.CreateZExt(low, llvm.Int16Type(), "")
	high16 := c.builder.CreateZExt(high, llvm.Int16Type(), "")
	word := c.builder.CreateShl(high16, llvm.ConstInt(llvm.Int16Type(), 8, false), "")
	return c.builder.CreateAnd(word, low16, "")
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

func (c *Compilation) pushWordToStack(word llvm.Value) {
	high16 := c.builder.CreateLShr(word, llvm.ConstInt(llvm.Int16Type(), 8, false), "")
	high := c.builder.CreateTrunc(high16, llvm.Int8Type(), "")
	c.pushToStack(high)
	low16 := c.builder.CreateAnd(word, llvm.ConstInt(llvm.Int16Type(), 0xff, false), "")
	low := c.builder.CreateTrunc(low16, llvm.Int8Type(), "")
	c.pushToStack(low)
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

func (c *Compilation) cycle(count int, pc int) {
	c.debugPrint(fmt.Sprintf("cycles %d\n", count))

	if pc >= 0 {
		c.builder.CreateStore(llvm.ConstInt(llvm.Int16Type(), uint64(pc), false), c.rPC)
	}

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

func (c *Compilation) createBranch(cond llvm.Value, labelName string, instrAddr int) {
	branchBlock := c.labeledBlocks[labelName]
	thenBlock := c.createBlock("then")
	elseBlock := c.createBlock("else")
	c.builder.CreateCondBr(cond, thenBlock, elseBlock)
	// if the condition is met, the cycle count is 3 or 4, depending
	// on whether the page boundary is crossed.
	c.selectBlock(thenBlock)
	addr, ok := c.program.Labels[labelName]
	if !ok {
		panic(fmt.Sprintf("label %s not defined", labelName))
	}
	if instrAddr&0xff00 == addr&0xff00 {
		c.cycle(3, addr)
	} else {
		c.cycle(4, addr)
	}
	c.builder.CreateBr(branchBlock)
	// the else block is when the code does *not* branch.
	// in this case, the cycle count is 2.
	c.selectBlock(elseBlock)
	c.cycle(2, instrAddr+2) // branch instructions are 2 bytes
}

func (c *Compilation) absoluteIndexedLoadData(destPtr llvm.Value, dataLabelName string, indexPtr llvm.Value, pc int) {
	dataPtr, ok := c.labeledData[dataLabelName]
	if !ok {
		_, ok := c.labeledBlocks[dataLabelName]
		if ok {
			c.Errors = append(c.Errors, fmt.Sprintf("label %s is instruction; expected data", dataLabelName))
		} else {
			c.Errors = append(c.Errors, fmt.Sprintf("unknown label %s", dataLabelName))
		}
		return
	}
	index := c.builder.CreateLoad(indexPtr, "")
	index16 := c.builder.CreateZExt(index, llvm.Int16Type(), "")
	indexes := []llvm.Value{
		llvm.ConstInt(llvm.Int16Type(), 0, false),
		index16,
	}
	ptr := c.builder.CreateGEP(dataPtr, indexes, "")
	v := c.builder.CreateLoad(ptr, "")
	c.builder.CreateStore(v, destPtr)
	c.dynTestAndSetNeg(v)
	c.dynTestAndSetZero(v)
	labelAddr, ok := c.program.Labels[dataLabelName]
	if !ok {
		panic(fmt.Sprintf("label %s not defined", dataLabelName))
	}
	c.cyclesForAbsoluteIndexed(labelAddr, index16, pc)
}

func (c *Compilation) absoluteIndexedStore(valPtr llvm.Value, baseAddr int, indexPtr llvm.Value, pc int) {
	index := c.builder.CreateLoad(indexPtr, "")
	index16 := c.builder.CreateZExt(index, llvm.Int16Type(), "")
	base := llvm.ConstInt(llvm.Int16Type(), uint64(baseAddr), false)
	addr := c.builder.CreateAdd(base, index16, "")
	val := c.builder.CreateLoad(valPtr, "")
	c.dynStore(addr, baseAddr, baseAddr+0xff, val)
	c.cycle(5, pc)
}

func (c *Compilation) dynLoadZpgIndexed(baseAddr int, indexPtr llvm.Value) llvm.Value {
	index := c.builder.CreateLoad(indexPtr, "")
	base := llvm.ConstInt(llvm.Int8Type(), uint64(baseAddr), false)
	addr := c.builder.CreateAdd(base, index, "")
	return c.dynLoad(addr, 0, 0xff)
}

func (c *Compilation) dynLoadIndexed(baseAddr int, indexPtr llvm.Value) llvm.Value {
	index := c.builder.CreateLoad(indexPtr, "")
	index16 := c.builder.CreateZExt(index, llvm.Int16Type(), "")
	base := llvm.ConstInt(llvm.Int16Type(), uint64(baseAddr), false)
	addr := c.builder.CreateAdd(base, index16, "")
	return c.dynLoad(addr, baseAddr, baseAddr+0xff)
}

func (c *Compilation) dynStoreZpgIndexed(baseAddr int, indexPtr llvm.Value, val llvm.Value) {
	index := c.builder.CreateLoad(indexPtr, "")
	base := llvm.ConstInt(llvm.Int8Type(), uint64(baseAddr), false)
	addr8 := c.builder.CreateAdd(base, index, "")
	addr16 := c.builder.CreateZExt(addr8, llvm.Int16Type(), "")
	c.dynStore(addr16, 0, 0xff, val)
}

func (c *Compilation) dynStoreIndexed(baseAddr int, indexPtr llvm.Value, val llvm.Value) {
	index := c.builder.CreateLoad(indexPtr, "")
	index16 := c.builder.CreateZExt(index, llvm.Int16Type(), "")
	base := llvm.ConstInt(llvm.Int16Type(), uint64(baseAddr), false)
	addr := c.builder.CreateAdd(base, index16, "")
	c.dynStore(addr, baseAddr, baseAddr+0xff, val)
}

func (c *Compilation) absoluteIndexedLoad(destPtr llvm.Value, baseAddr int, indexPtr llvm.Value, pc int) {
	index := c.builder.CreateLoad(indexPtr, "")
	index16 := c.builder.CreateZExt(index, llvm.Int16Type(), "")
	base := llvm.ConstInt(llvm.Int16Type(), uint64(baseAddr), false)
	addr := c.builder.CreateAdd(base, index16, "")
	v := c.dynLoad(addr, baseAddr, baseAddr+0xff)
	c.builder.CreateStore(v, destPtr)
	c.dynTestAndSetZero(v)
	c.dynTestAndSetNeg(v)
	c.cyclesForAbsoluteIndexed(baseAddr, index16, pc)
}

func (c *Compilation) cyclesForIndirectY(baseAddr, addr llvm.Value, pc int) {
	// if address & 0xff00 != (address + y) & 0xff00
	xff00 := llvm.ConstInt(llvm.Int16Type(), uint64(0xff00), false)
	baseAddrMasked := c.builder.CreateAnd(baseAddr, xff00, "")
	addrMasked := c.builder.CreateAnd(addr, xff00, "")
	eq := c.builder.CreateICmp(llvm.IntEQ, baseAddrMasked, addrMasked, "")
	loadDoneBlock := c.createBlock("LoadDone")
	pageBoundaryCrossedBlock := c.createIf(eq)
	// executed if page boundary is not crossed
	c.cycle(5, pc)
	c.builder.CreateBr(loadDoneBlock)
	// executed if page boundary crossed
	c.selectBlock(pageBoundaryCrossedBlock)
	c.cycle(6, pc)
	c.builder.CreateBr(loadDoneBlock)
	// done
	c.selectBlock(loadDoneBlock)
}

func (c *Compilation) cyclesForAbsoluteIndexedPtr(baseAddr int, indexPtr llvm.Value, pc int) {
	index := c.builder.CreateLoad(indexPtr, "")
	index16 := c.builder.CreateZExt(index, llvm.Int16Type(), "")
	c.cyclesForAbsoluteIndexed(baseAddr, index16, pc)
}

func (c *Compilation) cyclesForAbsoluteIndexed(baseAddr int, index16 llvm.Value, pc int) {
	// if address & 0xff00 != (address + x) & 0xff00
	baseAddrMasked := baseAddr & 0xff00
	baseAddrMaskedValue := llvm.ConstInt(llvm.Int16Type(), uint64(baseAddrMasked), false)

	baseAddrValue := llvm.ConstInt(llvm.Int16Type(), uint64(baseAddr), false)
	addrPlusX := c.builder.CreateAdd(baseAddrValue, index16, "")
	xff00 := llvm.ConstInt(llvm.Int16Type(), uint64(0xff00), false)
	maskedAddrPlusX := c.builder.CreateAnd(addrPlusX, xff00, "")

	eq := c.builder.CreateICmp(llvm.IntEQ, baseAddrMaskedValue, maskedAddrPlusX, "")
	loadDoneBlock := c.createBlock("LoadDone")
	pageBoundaryCrossedBlock := c.createIf(eq)
	// executed if page boundary is not crossed
	c.cycle(4, pc)
	c.builder.CreateBr(loadDoneBlock)
	// executed if page boundary crossed
	c.selectBlock(pageBoundaryCrossedBlock)
	c.cycle(5, pc)
	c.builder.CreateBr(loadDoneBlock)
	// done
	c.selectBlock(loadDoneBlock)
}

func (c *Compilation) dynTestAndSetCarryAddition(a llvm.Value, v llvm.Value, carry llvm.Value) {
	a32 := c.builder.CreateZExt(a, llvm.Int32Type(), "")
	carry32 := c.builder.CreateZExt(carry, llvm.Int32Type(), "")
	v32 := c.builder.CreateZExt(v, llvm.Int32Type(), "")
	aPlusV32 := c.builder.CreateAdd(a32, v32, "")
	newA32 := c.builder.CreateAdd(aPlusV32, carry32, "")
	isCarry := c.builder.CreateICmp(llvm.IntUGE, newA32, llvm.ConstInt(llvm.Int32Type(), 0x100, false), "")
	c.builder.CreateStore(isCarry, c.rSCarry)
}

func (c *Compilation) dynTestAndSetOverflowAddition(a llvm.Value, b llvm.Value, r llvm.Value) {
	x80 := llvm.ConstInt(llvm.Int8Type(), 0x80, false)
	x0 := llvm.ConstInt(llvm.Int8Type(), 0x0, false)
	aXorB := c.builder.CreateXor(a, b, "")
	aXorBMasked := c.builder.CreateAnd(aXorB, x80, "")
	aXorR := c.builder.CreateXor(a, r, "")
	aXorRMasked := c.builder.CreateAnd(aXorR, x80, "")
	isOverA := c.builder.CreateICmp(llvm.IntEQ, aXorBMasked, x0, "")
	isOverR := c.builder.CreateICmp(llvm.IntEQ, aXorRMasked, x80, "")
	isOver := c.builder.CreateAnd(isOverA, isOverR, "")
	c.builder.CreateStore(isOver, c.rSOver)
}

func (c *Compilation) dynTestAndSetOverflowSubtraction(a llvm.Value, b llvm.Value, carry llvm.Value) {
	c0 := llvm.ConstInt(llvm.Int8Type(), 0, false)
	c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
	x80 := llvm.ConstInt(llvm.Int8Type(), 0x80, false)

	aMinusB := c.builder.CreateSub(a, b, "")
	invertedCarry := c.builder.CreateSub(c1, carry, "")
	val := c.builder.CreateSub(aMinusB, invertedCarry, "")

	aXorVal := c.builder.CreateXor(a, val, "")
	aXorValMasked := c.builder.CreateAnd(aXorVal, x80, "")
	aXorB := c.builder.CreateXor(a, b, "")
	aXorBMasked := c.builder.CreateAnd(aXorB, x80, "")
	isOverVal := c.builder.CreateICmp(llvm.IntNE, aXorValMasked, c0, "")
	isOverB := c.builder.CreateICmp(llvm.IntNE, aXorBMasked, c0, "")

	isOver := c.builder.CreateAnd(isOverVal, isOverB, "")
	c.builder.CreateStore(isOver, c.rSOver)
}

func (c *Compilation) performCmp(lval llvm.Value, rval llvm.Value) {
	diff := c.builder.CreateSub(lval, rval, "")
	c.dynTestAndSetZero(diff)
	c.dynTestAndSetNeg(diff)
	c.dynTestAndSetCarrySubtraction(lval, rval)
}

func (c *Compilation) labelAsEntryPoint(labelName string) int {
	id, ok := c.labelIds[labelName]
	if ok {
		return id
	}
	c.entryLabelCount += 1
	c.labelIds[labelName] = c.entryLabelCount
	return c.entryLabelCount
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

func (c *Compilation) declareReadFn(name string) llvm.Value {
	readByteType := llvm.FunctionType(llvm.Int8Type(), []llvm.Type{}, false)
	fn := llvm.AddFunction(c.mod, name, readByteType)
	fn.SetLinkage(llvm.ExternalLinkage)
	return fn
}

func (c *Compilation) declareWriteFn(name string) llvm.Value {
	writeByteType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int8Type()}, false)
	fn := llvm.AddFunction(c.mod, name, writeByteType)
	fn.SetLinkage(llvm.ExternalLinkage)
	return fn
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

	// PPU
	c.ppuReadStatusFn = c.declareReadFn("rom_ppu_read_status")
	c.ppuReadOamDataFn = c.declareReadFn("rom_ppu_read_oamdata")
	c.ppuReadDataFn = c.declareReadFn("rom_ppu_read_data")
	c.ppuCtrlFn = c.declareWriteFn("rom_ppu_write_control")
	c.ppuMaskFn = c.declareWriteFn("rom_ppu_write_mask")
	c.ppuAddrFn = c.declareWriteFn("rom_ppu_write_address")
	c.setPpuDataFn = c.declareWriteFn("rom_ppu_write_data")
	c.oamAddrFn = c.declareWriteFn("rom_ppu_write_oamaddress")
	c.setOamDataFn = c.declareWriteFn("rom_ppu_write_oamdata")
	c.setPpuScrollFn = c.declareWriteFn("rom_ppu_write_scroll")
	c.ppuWriteDma = c.declareWriteFn("rom_ppu_write_dma")

	// APU
	c.apuWriteSquare1CtrlFn = c.declareWriteFn("rom_apu_write_square1control")
	c.apuWriteSquare1SweepsFn = c.declareWriteFn("rom_apu_write_square1sweeps")
	c.apuWriteSquare1LowFn = c.declareWriteFn("rom_apu_write_square1low")
	c.apuWriteSquare1HighFn = c.declareWriteFn("rom_apu_write_square1high")
	c.apuWriteSquare2CtrlFn = c.declareWriteFn("rom_apu_write_square2control")
	c.apuWriteSquare2SweepsFn = c.declareWriteFn("rom_apu_write_square2sweeps")
	c.apuWriteSquare2LowFn = c.declareWriteFn("rom_apu_write_square2low")
	c.apuWriteSquare2HighFn = c.declareWriteFn("rom_apu_write_square2high")
	c.apuWriteTriangleCtrlFn = c.declareWriteFn("rom_apu_write_trianglecontrol")
	c.apuWriteTriangleLowFn = c.declareWriteFn("rom_apu_write_trianglelow")
	c.apuWriteTriangleHighFn = c.declareWriteFn("rom_apu_write_trianglehigh")
	c.apuWriteNoiseBaseFn = c.declareWriteFn("rom_apu_write_noisebase")
	c.apuWriteNoisePeriodFn = c.declareWriteFn("rom_apu_write_noiseperiod")
	c.apuWriteNoiseLengthFn = c.declareWriteFn("rom_apu_write_noiselength")
	c.apuWriteDmcFlagsFn = c.declareWriteFn("rom_apu_write_dmcflags")
	c.apuWriteDmcDirectLoadFn = c.declareWriteFn("rom_apu_write_dmcdirectload")
	c.apuWriteDmcSampleAddressFn = c.declareWriteFn("rom_apu_write_dmcsampleaddress")
	c.apuWriteDmcSampleLengthFn = c.declareWriteFn("rom_apu_write_dmcsamplelength")
	c.apuWriteCtrlFlags1Fn = c.declareWriteFn("rom_apu_write_controlflags1")
	c.apuWriteCtrlFlags2Fn = c.declareWriteFn("rom_apu_write_controlflags2")

	// pads
	c.padWrite1Fn = c.declareWriteFn("rom_pad_write1")
	c.padWrite2Fn = c.declareWriteFn("rom_pad_write2")
	c.padRead1Fn = c.declareReadFn("rom_pad_read1")
	c.padRead2Fn = c.declareReadFn("rom_pad_read2")
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
	// * push PC low onto stack
	c.pushWordToStack(c.builder.CreateLoad(c.rPC, ""))
	// * push processor status onto stack
	c.pushStatusReg()
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
	c.labelIds = map[string]int{}
	c.entryLabelCount = 3 // irq, reset, nmi

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
	text := llvm.ConstString("panic: attempted to write to invalid memory address\n", true)
	c.runtimePanicMsg = llvm.AddGlobal(c.mod, text.Type(), "panicMsg")
	c.runtimePanicMsg.SetLinkage(llvm.PrivateLinkage)
	c.runtimePanicMsg.SetInitializer(text)

	// first pass to generate data declarations
	c.mode = dataStmtMode
	p.Ast.Ast(c)

	c.createFunctionDeclares()
	c.createRegisters()

	// main function / entry point
	mainType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int32Type()}, false)
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
		c.Warnings = append(c.Warnings, "missing irq entry point; inserting dummy.")
		tmp := llvm.AddBasicBlock(c.mainFn, "IRQ_Routine")
		c.irqBlock = &tmp
		c.builder.SetInsertPointAtEnd(*c.irqBlock)
		c.builder.CreateUnreachable()
	}

	// entry jump table
	c.selectBlock(entry)
	c.builder.SetInsertPointAtEnd(entry)
	badInterruptBlock := c.createBlock("BadInterrupt")
	sw := c.builder.CreateSwitch(c.mainFn.Param(0), badInterruptBlock, c.entryLabelCount)
	c.selectBlock(badInterruptBlock)
	c.createPanic()
	sw.AddCase(llvm.ConstInt(llvm.Int32Type(), 1, false), *c.nmiBlock)
	sw.AddCase(llvm.ConstInt(llvm.Int32Type(), 2, false), *c.resetBlock)
	sw.AddCase(llvm.ConstInt(llvm.Int32Type(), 3, false), *c.irqBlock)
	for labelName, labelId := range c.labelIds {
		sw.AddCase(llvm.ConstInt(llvm.Int32Type(), uint64(labelId), false), c.labeledBlocks[labelName])
	}

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
