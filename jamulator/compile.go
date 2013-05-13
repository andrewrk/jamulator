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
	prgRom          llvm.Value // 32KB PRG ROM
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

	// controllers. see http://wiki.nesdev.com/w/index.php/Standard_controller
	btnReportIndex  llvm.Value // index of the button to report next
	padsActual      llvm.Value // actual contoller state
	padsReport      llvm.Value // reported controller state
	strobeOn        llvm.Value // strobe bit status

	labeledBlocks map[string]llvm.BasicBlock
	labeledData   map[string]bool
	stringTable   map[string]llvm.Value
	// used for the entry jump table so we can do JSR
	labelIds        map[string]int
	entryLabelCount int

	currentValue *bytes.Buffer
	currentLabel string
	currentExpecting int
	mode         int
	currentBlock *llvm.BasicBlock
	currentInstr Compiler
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
	memcpyFn  llvm.Value
	exitFn    llvm.Value
	cycleFn   llvm.Value
	// PPU
	ppuReadStatusFn  llvm.Value
	ppuReadOamDataFn llvm.Value
	ppuReadDataFn    llvm.Value
	ppuCtrlFn        llvm.Value
	ppuMaskFn        llvm.Value
	ppuAddrFn        llvm.Value
	setPpuDataFn     llvm.Value
	oamAddrFn        llvm.Value
	setOamDataFn     llvm.Value
	setPpuScrollFn   llvm.Value
	ppuWriteDma      llvm.Value
	// APU
	apuReadStatusFn            llvm.Value
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
	padWriteFn llvm.Value
	padReadFn  llvm.Value
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

const (
	cfExpectNone = iota
	cfExpectData
	cfExpectInstr
)

func (c *Compilation) Visit(n Node) {
	switch c.mode {
	case dataStmtMode:
		c.visitForControlFlow(n)
	case basicBlocksMode:
		c.visitForBasicBlocks(n)
	case compileMode:
		c.visitForCompile(n)
	}
}

func (c *Compilation) ctrlFlowGotData(n Node) {
	switch c.currentExpecting {
	case cfExpectInstr:
		// TODO: we need some of the logic from disassembly here
		//c.Errors = append(c.Errors, fmt.Sprintf("expected instruction, got data: %s", n.(Renderer).Render()))
		c.currentExpecting = cfExpectData
	case cfExpectNone:
		c.labeledData[c.currentLabel] = true
		c.currentExpecting = cfExpectData
	}
}

func (c *Compilation) visitForControlFlow(n Node) {
	switch t := n.(type) {
	case *LabeledStatement:
		c.currentLabel = t.LabelName
		c.currentExpecting = cfExpectNone
	case *DataStatement:
		c.ctrlFlowGotData(n)
	case *DataWordStatement:
		c.ctrlFlowGotData(n)
	case Compiler:
		switch c.currentExpecting {
		case cfExpectData:
			// TODO: we need some of the logic from disassembly here
			//c.Errors = append(c.Errors, fmt.Sprintf("expected data, got instruction: %s", n.(Renderer).Render()))
			c.currentExpecting = cfExpectInstr
			return
		case cfExpectNone:
			c.currentExpecting = cfExpectInstr
		}
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
		c.currentInstr = t
		t.Compile(c)
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

func (c *Compilation) performCmp(lval llvm.Value, rval llvm.Value) {
	diff := c.builder.CreateSub(lval, rval, "")
	c.dynTestAndSetZero(diff)
	c.dynTestAndSetNeg(diff)
	c.dynTestAndSetCarrySubtraction(lval, rval)
}

func (c *Compilation) performRor(val llvm.Value) llvm.Value {
	c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
	c7 := llvm.ConstInt(llvm.Int8Type(), 7, false)
	shifted := c.builder.CreateLShr(val, c1, "")
	carryBit := c.builder.CreateLoad(c.rSCarry, "")
	carry := c.builder.CreateZExt(carryBit, llvm.Int8Type(), "")
	carryShifted := c.builder.CreateShl(carry, c7, "")
	newValue := c.builder.CreateOr(shifted, carryShifted, "")
	c.dynTestAndSetZero(newValue)
	c.dynTestAndSetNeg(newValue)
	c.dynTestAndSetCarryLShr(val)
	return newValue
}

func (c *Compilation) performRol(val llvm.Value) llvm.Value {
	c1 := llvm.ConstInt(val.Type(), 1, false)
	shifted := c.builder.CreateShl(val, c1, "")
	carryBit := c.builder.CreateLoad(c.rSCarry, "")
	carry := c.builder.CreateZExt(carryBit, val.Type(), "")
	newValue := c.builder.CreateOr(shifted, carry, "")

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

func (c *Compilation) performAnd(v llvm.Value) {
	a := c.builder.CreateLoad(c.rA, "")
	newA := c.builder.CreateAnd(a, v, "")
	c.builder.CreateStore(newA, c.rA)
	c.dynTestAndSetZero(newA)
	c.dynTestAndSetNeg(newA)
}

func (c *Compilation) performEor(v llvm.Value) {
	a := c.builder.CreateLoad(c.rA, "")
	newA := c.builder.CreateXor(a, v, "")
	c.builder.CreateStore(newA, c.rA)
	c.dynTestAndSetZero(newA)
	c.dynTestAndSetNeg(newA)
}

func (c *Compilation) dynStore(addr llvm.Value, minAddr int, maxAddr int, val llvm.Value) {
	c.debugPrintf("store $%02x in $%04x\n", []llvm.Value{val, addr})
	if maxAddr < 0x800 {
		// wram. we don't even have to mask it
		indexes := []llvm.Value{
			llvm.ConstInt(addr.Type(), 0, false),
			addr,
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		c.builder.CreateStore(val, ptr)
		return
	}

	if minAddr != 0 || maxAddr != 0xffff {
		c.Warnings = append(c.Warnings, fmt.Sprintf("TODO: dynStore is unoptimized for min $%04x max $%04x", minAddr, maxAddr))
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
	c.createPanic("invalid store address: $%04x\n", []llvm.Value{addr})

	ppuCtrlBlock := c.createBlock("ppuctrl")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0, false), ppuCtrlBlock)
	c.selectBlock(ppuCtrlBlock)
	c.debugPrint("ppu_write_control\n")
	c.builder.CreateCall(c.ppuCtrlFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuMaskBlock := c.createBlock("ppumask")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 1, false), ppuMaskBlock)
	c.selectBlock(ppuMaskBlock)
	c.debugPrint("ppu_write_mask\n")
	c.builder.CreateCall(c.ppuMaskFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	oamAddrBlock := c.createBlock("oamaddr")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 3, false), oamAddrBlock)
	c.selectBlock(oamAddrBlock)
	c.debugPrint("ppu_write_oamaddr\n")
	c.builder.CreateCall(c.oamAddrFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	oamDataBlock := c.createBlock("oamdata")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 4, false), oamDataBlock)
	c.selectBlock(oamDataBlock)
	c.debugPrint("ppu_write_oamdata\n")
	c.builder.CreateCall(c.setOamDataFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuScrollBlock := c.createBlock("ppuscroll")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 5, false), ppuScrollBlock)
	c.selectBlock(ppuScrollBlock)
	c.debugPrint("ppu_write_scroll\n")
	c.builder.CreateCall(c.setPpuScrollFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuAddrBlock := c.createBlock("ppuaddr")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 6, false), ppuAddrBlock)
	c.selectBlock(ppuAddrBlock)
	c.debugPrint("ppu_write_address\n")
	c.builder.CreateCall(c.ppuAddrFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuDataBlock := c.createBlock("ppudata")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 7, false), ppuDataBlock)
	c.selectBlock(ppuDataBlock)
	c.debugPrint("ppu_write_data\n")
	c.builder.CreateCall(c.setPpuDataFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	// this generated code runs if the write is >= 0x4000
	c.selectBlock(notInPpuRamBlock)
	x4017 := llvm.ConstInt(llvm.Int16Type(), 0x4017, false)
	inApuRam := c.builder.CreateICmp(llvm.IntULE, addr, x4017, "")
	notInApuRamBlock := c.createIf(inApuRam)
	// if the write is in the APU RAM range
	badApuRamBlock := c.createBlock("BadAPUAddr")
	sw = c.builder.CreateSwitch(addr, badApuRamBlock, 22)
	c.selectBlock(badApuRamBlock)
	c.createPanic("invalid store address: $%04x\n", []llvm.Value{addr})

	apuSqr1CtrlBlock := c.createBlock("rom_apu_write_square1control")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4000, false), apuSqr1CtrlBlock)
	c.selectBlock(apuSqr1CtrlBlock)
	c.debugPrint("rom_apu_write_square1control\n")
	c.builder.CreateCall(c.apuWriteSquare1CtrlFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuSqr1SweepsBlock := c.createBlock("rom_apu_write_square1sweeps")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4001, false), apuSqr1SweepsBlock)
	c.selectBlock(apuSqr1SweepsBlock)
	c.debugPrint("rom_apu_write_square1sweeps\n")
	c.builder.CreateCall(c.apuWriteSquare1SweepsFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuSqr1LowBlock := c.createBlock("rom_apu_write_square1low")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4002, false), apuSqr1LowBlock)
	c.selectBlock(apuSqr1LowBlock)
	c.debugPrint("rom_apu_write_square1low\n")
	c.builder.CreateCall(c.apuWriteSquare1LowFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuSqr1HighBlock := c.createBlock("rom_apu_write_square1high")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4003, false), apuSqr1HighBlock)
	c.selectBlock(apuSqr1HighBlock)
	c.debugPrint("rom_apu_write_square1high\n")
	c.builder.CreateCall(c.apuWriteSquare1HighFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuSqr2CtrlBlock := c.createBlock("rom_apu_write_square2control")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4004, false), apuSqr2CtrlBlock)
	c.selectBlock(apuSqr2CtrlBlock)
	c.debugPrint("rom_apu_write_square2control\n")
	c.builder.CreateCall(c.apuWriteSquare2CtrlFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuSqr2SweepsBlock := c.createBlock("rom_apu_write_square2sweeps")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4005, false), apuSqr2SweepsBlock)
	c.selectBlock(apuSqr2SweepsBlock)
	c.debugPrint("rom_apu_write_square2sweeps\n")
	c.builder.CreateCall(c.apuWriteSquare2SweepsFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuSqr2LowBlock := c.createBlock("rom_apu_write_square2low")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4006, false), apuSqr2LowBlock)
	c.selectBlock(apuSqr2LowBlock)
	c.debugPrint("rom_apu_write_square2low\n")
	c.builder.CreateCall(c.apuWriteSquare2LowFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuSqr2HighBlock := c.createBlock("rom_apu_write_square2high")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4007, false), apuSqr2HighBlock)
	c.selectBlock(apuSqr2HighBlock)
	c.debugPrint("rom_apu_write_square2high\n")
	c.builder.CreateCall(c.apuWriteSquare2HighFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuTriCtrlBlock := c.createBlock("rom_apu_write_trianglecontrol")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4008, false), apuTriCtrlBlock)
	c.selectBlock(apuTriCtrlBlock)
	c.debugPrint("rom_apu_write_trianglecontrol\n")
	c.builder.CreateCall(c.apuWriteTriangleCtrlFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuTriLowBlock := c.createBlock("rom_apu_write_trianglelow")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x400a, false), apuTriLowBlock)
	c.selectBlock(apuTriLowBlock)
	c.debugPrint("rom_apu_write_trianglelow\n")
	c.builder.CreateCall(c.apuWriteTriangleLowFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuTriHighBlock := c.createBlock("rom_apu_write_trianglehigh")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x400b, false), apuTriHighBlock)
	c.selectBlock(apuTriHighBlock)
	c.debugPrint("rom_apu_write_trianglehigh\n")
	c.builder.CreateCall(c.apuWriteTriangleHighFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuNoiseBaseBlock := c.createBlock("rom_apu_write_noisebase")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x400c, false), apuNoiseBaseBlock)
	c.selectBlock(apuNoiseBaseBlock)
	c.debugPrint("rom_apu_write_noisebase\n")
	c.builder.CreateCall(c.apuWriteNoiseBaseFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuNoisePeriodBlock := c.createBlock("rom_apu_write_noiseperiod")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x400e, false), apuNoisePeriodBlock)
	c.selectBlock(apuNoisePeriodBlock)
	c.debugPrint("rom_apu_write_noiseperiod\n")
	c.builder.CreateCall(c.apuWriteNoisePeriodFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuNoiseLengthBlock := c.createBlock("rom_apu_write_noiselength")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x400f, false), apuNoiseLengthBlock)
	c.selectBlock(apuNoiseLengthBlock)
	c.debugPrint("rom_apu_write_noiselength\n")
	c.builder.CreateCall(c.apuWriteNoiseLengthFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuDmcFlagsBlock := c.createBlock("rom_apu_write_dmcflags")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4010, false), apuDmcFlagsBlock)
	c.selectBlock(apuDmcFlagsBlock)
	c.debugPrint("rom_apu_write_dmcflags\n")
	c.builder.CreateCall(c.apuWriteDmcFlagsFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuDmcDirectLoadBlock := c.createBlock("rom_apu_write_dmcdirectload")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4011, false), apuDmcDirectLoadBlock)
	c.selectBlock(apuDmcDirectLoadBlock)
	c.debugPrint("rom_apu_write_dmcdirectload\n")
	c.builder.CreateCall(c.apuWriteDmcDirectLoadFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuDmcSampleAddrBlock := c.createBlock("rom_apu_write_dmcsampleaddress")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4012, false), apuDmcSampleAddrBlock)
	c.selectBlock(apuDmcSampleAddrBlock)
	c.debugPrint("rom_apu_write_dmcsampleaddress\n")
	c.builder.CreateCall(c.apuWriteDmcSampleAddressFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuDmcSampleLenBlock := c.createBlock("rom_apu_write_dmcsamplelength")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4013, false), apuDmcSampleLenBlock)
	c.selectBlock(apuDmcSampleLenBlock)
	c.debugPrint("rom_apu_write_dmcsamplelength\n")
	c.builder.CreateCall(c.apuWriteDmcSampleLengthFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	ppuDmaBlock := c.createBlock("rom_ppu_write_dma")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4014, false), ppuDmaBlock)
	c.selectBlock(ppuDmaBlock)
	c.debugPrint("ppu_write_oamdata\n")
	c.builder.CreateCall(c.setOamDataFn, []llvm.Value{val}, "")
	c.debugPrint("rom_ppu_write_dma\n")
	c.builder.CreateCall(c.ppuWriteDma, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuCtrlFlags1Block := c.createBlock("rom_apu_write_controlflags1")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4015, false), apuCtrlFlags1Block)
	c.selectBlock(apuCtrlFlags1Block)
	c.debugPrint("rom_apu_write_controlflags1\n")
	c.builder.CreateCall(c.apuWriteCtrlFlags1Fn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	padWriteBlock := c.createBlock("padWrite")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4016, false), padWriteBlock)
	c.selectBlock(padWriteBlock)
	c.debugPrintf("pad_write $%02x\n", []llvm.Value{val})
	c.builder.CreateCall(c.padWriteFn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	apuCtrlFlags2Block := c.createBlock("rom_apu_write_controlflags2")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 0x4017, false), apuCtrlFlags2Block)
	c.selectBlock(apuCtrlFlags2Block)
	c.debugPrint("rom_apu_write_controlflags2\n")
	c.builder.CreateCall(c.apuWriteCtrlFlags2Fn, []llvm.Value{val}, "")
	c.builder.CreateBr(storeDoneBlock)

	// if not in any known writable range
	c.selectBlock(notInApuRamBlock)
	c.createPanic("invalid store address: $%04x\n", []llvm.Value{addr})

	// done. X_X
	c.selectBlock(storeDoneBlock)
}

func (c *Compilation) store(addr int, i8 llvm.Value) {
	c.debugPrintf("store $%02x in $%04x\n", []llvm.Value{i8, llvm.ConstInt(llvm.Int16Type(), uint64(addr), false)})

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
			c.debugPrint("rom_apu_write_square1control\n")
			c.builder.CreateCall(c.apuWriteSquare1CtrlFn, []llvm.Value{i8}, "")
		case 0x4001:
			c.debugPrint("rom_apu_write_square1sweeps\n")
			c.builder.CreateCall(c.apuWriteSquare1SweepsFn, []llvm.Value{i8}, "")
		case 0x4002:
			c.debugPrint("rom_apu_write_square1low\n")
			c.builder.CreateCall(c.apuWriteSquare1LowFn, []llvm.Value{i8}, "")
		case 0x4003:
			c.debugPrint("rom_apu_write_square1high\n")
			c.builder.CreateCall(c.apuWriteSquare1HighFn, []llvm.Value{i8}, "")
		case 0x4004:
			c.debugPrint("rom_apu_write_square2control\n")
			c.builder.CreateCall(c.apuWriteSquare2CtrlFn, []llvm.Value{i8}, "")
		case 0x4005:
			c.debugPrint("rom_apu_write_square2sweeps\n")
			c.builder.CreateCall(c.apuWriteSquare2SweepsFn, []llvm.Value{i8}, "")
		case 0x4006:
			c.debugPrint("rom_apu_write_square2low\n")
			c.builder.CreateCall(c.apuWriteSquare2LowFn, []llvm.Value{i8}, "")
		case 0x4007:
			c.debugPrint("rom_apu_write_square2high\n")
			c.builder.CreateCall(c.apuWriteSquare2HighFn, []llvm.Value{i8}, "")
		case 0x4008:
			c.debugPrint("rom_apu_write_trianglecontrol\n")
			c.builder.CreateCall(c.apuWriteTriangleCtrlFn, []llvm.Value{i8}, "")
		case 0x400a:
			c.debugPrint("rom_apu_write_trianglelow\n")
			c.builder.CreateCall(c.apuWriteTriangleLowFn, []llvm.Value{i8}, "")
		case 0x400b:
			c.debugPrint("rom_apu_write_trianglehigh\n")
			c.builder.CreateCall(c.apuWriteTriangleHighFn, []llvm.Value{i8}, "")
		case 0x400c:
			c.debugPrint("rom_apu_write_noisebase\n")
			c.builder.CreateCall(c.apuWriteNoiseBaseFn, []llvm.Value{i8}, "")
		case 0x400e:
			c.debugPrint("rom_apu_write_noiseperiod\n")
			c.builder.CreateCall(c.apuWriteNoisePeriodFn, []llvm.Value{i8}, "")
		case 0x400f:
			c.debugPrint("rom_apu_write_noiselength\n")
			c.builder.CreateCall(c.apuWriteNoiseLengthFn, []llvm.Value{i8}, "")
		case 0x4010:
			c.debugPrint("rom_apu_write_dmcflags\n")
			c.builder.CreateCall(c.apuWriteDmcFlagsFn, []llvm.Value{i8}, "")
		case 0x4011:
			c.debugPrint("rom_apu_write_dmcdirectload\n")
			c.builder.CreateCall(c.apuWriteDmcDirectLoadFn, []llvm.Value{i8}, "")
		case 0x4012:
			c.debugPrint("rom_apu_write_dmcsampleaddress\n")
			c.builder.CreateCall(c.apuWriteDmcSampleAddressFn, []llvm.Value{i8}, "")
		case 0x4013:
			c.debugPrint("rom_apu_write_dmcsamplelength\n")
			c.builder.CreateCall(c.apuWriteDmcSampleLengthFn, []llvm.Value{i8}, "")
		case 0x4014:
			c.debugPrint("ppu_write_oamdata\n")
			c.builder.CreateCall(c.setOamDataFn, []llvm.Value{i8}, "")
			c.debugPrint("rom_ppu_write_dma\n")
			c.builder.CreateCall(c.ppuWriteDma, []llvm.Value{i8}, "")
		case 0x4015:
			c.debugPrint("rom_apu_write_controlflags1\n")
			c.builder.CreateCall(c.apuWriteCtrlFlags1Fn, []llvm.Value{i8}, "")
		case 0x4016:
			c.debugPrintf("pad_write $%02x\n", []llvm.Value{i8})
			c.builder.CreateCall(c.padWriteFn, []llvm.Value{i8}, "")
		case 0x4017:
			c.debugPrint("rom_apu_write_controlflags2\n")
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
			llvm.ConstInt(addr.Type(), 0, false),
			addr,
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		return c.builder.CreateLoad(ptr, "")
	}
	if maxAddr < 0x2000 {
		// address masking needed, but it's definitely in WRAM
		maskedAddr := c.builder.CreateAnd(addr, llvm.ConstInt(llvm.Int16Type(), 0x800-1, false), "")
		indexes := []llvm.Value{
			llvm.ConstInt(maskedAddr.Type(), 0, false),
			maskedAddr,
		}
		ptr := c.builder.CreateGEP(c.wram, indexes, "")
		return c.builder.CreateLoad(ptr, "")
	}
	x8000 := llvm.ConstInt(addr.Type(), 0x8000, false)
	if minAddr >= 0x8000 && maxAddr <= 0xffff {
		// PRG ROM load
		offsetAddr := c.builder.CreateSub(addr, x8000, "")
		indexes := []llvm.Value{
			llvm.ConstInt(offsetAddr.Type(), 0, false),
			offsetAddr,
		}
		ptr := c.builder.CreateGEP(c.prgRom, indexes, "")
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
	c.createPanic("invalid load address: $%04x\n", []llvm.Value{addr})

	ppuReadStatusBlock := c.createBlock("ppu_read_status")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 2, false), ppuReadStatusBlock)
	c.selectBlock(ppuReadStatusBlock)
	c.debugPrint("ppu_read_status\n")
	v = c.builder.CreateCall(c.ppuReadStatusFn, []llvm.Value{}, "")
	c.builder.CreateStore(v, result)
	c.builder.CreateBr(loadDoneBlock)

	ppuReadOamDataBlock := c.createBlock("ppu_read_oamdata")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 4, false), ppuReadOamDataBlock)
	c.selectBlock(ppuReadOamDataBlock)
	c.debugPrint("ppu_read_oamdata\n")
	v = c.builder.CreateCall(c.ppuReadOamDataFn, []llvm.Value{}, "")
	c.builder.CreateStore(v, result)
	c.builder.CreateBr(loadDoneBlock)

	ppuReadDataBlock := c.createBlock("ppu_read_data")
	sw.AddCase(llvm.ConstInt(llvm.Int16Type(), 7, false), ppuReadDataBlock)
	c.selectBlock(ppuReadDataBlock)
	c.debugPrint("ppu_read_data\n")
	v = c.builder.CreateCall(c.ppuReadDataFn, []llvm.Value{}, "")
	c.builder.CreateStore(v, result)
	c.builder.CreateBr(loadDoneBlock)

	// this generated code runs if the write is > PPU RAM range
	c.selectBlock(notInPpuRamBlock)
	inPrgRom := c.builder.CreateICmp(llvm.IntUGE, addr, x8000, "")
	notInPrgRomBlock := c.createIf(inPrgRom)
	// this generated code runs if the write is in the PRG ROM range
	offsetAddr := c.builder.CreateSub(addr, x8000, "")
	indexes = []llvm.Value{
		llvm.ConstInt(offsetAddr.Type(), 0, false),
		offsetAddr,
	}
	ptr = c.builder.CreateGEP(c.prgRom, indexes, "")
	v = c.builder.CreateLoad(ptr, "")
	c.builder.CreateStore(v, result)
	c.builder.CreateBr(loadDoneBlock)
	// this generated code runs if the write is not in the PRG ROM range
	c.selectBlock(notInPrgRomBlock)
	c.createPanic("invalid load address: $%04x\n", []llvm.Value{addr})

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
		llvm.ConstInt(llvm.Int16Type(), 0, false),
		llvm.ConstInt(llvm.Int16Type(), uint64(maskedAddr), false),
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
	case addr == 0x4015:
		c.debugPrint("rom_apu_read_status\n")
		return c.builder.CreateCall(c.apuReadStatusFn, []llvm.Value{}, "")
	case addr == 0x4016:
		c0 := llvm.ConstInt(llvm.Int8Type(), 0, false)
		v := c.builder.CreateCall(c.padReadFn, []llvm.Value{c0}, "")
		c.debugPrintf("pad_read1 $%02x\n", []llvm.Value{v})
		return v
	case addr == 0x4017:
		c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
		c.debugPrint("pad_read2\n")
		return c.builder.CreateCall(c.padReadFn, []llvm.Value{c1}, "")
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

func (c *Compilation) createPanic(msg string, args []llvm.Value) {
	renderer := c.currentInstr.(Renderer)
	c.printf(msg, args)
	c.printf(fmt.Sprintf("current instruction: %s\n", renderer.Render()), []llvm.Value{})
	c.printf("A: $%02x  X: $%02x  Y: $%02x  SP: $%02x  PC: $%04x\n", []llvm.Value{
		c.builder.CreateLoad(c.rA, ""),
		c.builder.CreateLoad(c.rX, ""),
		c.builder.CreateLoad(c.rY, ""),
		c.builder.CreateLoad(c.rSP, ""),
		c.builder.CreateLoad(c.rPC, ""),
	})
	c.printf("N: %d  V: %d  -  B: %d  D: %d  I: %d  Z: %d  C: %d\n", []llvm.Value{
		c.builder.CreateZExt(c.builder.CreateLoad(c.rSNeg, ""), llvm.Int8Type(), ""),
		c.builder.CreateZExt(c.builder.CreateLoad(c.rSOver, ""), llvm.Int8Type(), ""),
		c.builder.CreateZExt(c.builder.CreateLoad(c.rSBrk, ""), llvm.Int8Type(), ""),
		c.builder.CreateZExt(c.builder.CreateLoad(c.rSDec, ""), llvm.Int8Type(), ""),
		c.builder.CreateZExt(c.builder.CreateLoad(c.rSInt, ""), llvm.Int8Type(), ""),
		c.builder.CreateZExt(c.builder.CreateLoad(c.rSZero, ""), llvm.Int8Type(), ""),
		c.builder.CreateZExt(c.builder.CreateLoad(c.rSCarry, ""), llvm.Int8Type(), ""),
	})
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
	spZExt := c.builder.CreateZExt(spPlusOne, llvm.Int16Type(), "")
	addr := c.builder.CreateAdd(spZExt, llvm.ConstInt(llvm.Int16Type(), 0x100, false), "")
	return c.dynLoad(addr, 0x100, 0x1ff)
}

func (c *Compilation) pullWordFromStack() llvm.Value {
	low := c.pullFromStack()
	high := c.pullFromStack()
	low16 := c.builder.CreateZExt(low, llvm.Int16Type(), "")
	high16 := c.builder.CreateZExt(high, llvm.Int16Type(), "")
	word := c.builder.CreateShl(high16, llvm.ConstInt(high16.Type(), 8, false), "")
	return c.builder.CreateOr(word, low16, "")
}

func (c *Compilation) pushToStack(v llvm.Value) {
	// write the value to the address at current stack pointer
	sp := c.builder.CreateLoad(c.rSP, "")
	spZExt := c.builder.CreateZExt(sp, llvm.Int16Type(), "")
	addr := c.builder.CreateAdd(spZExt, llvm.ConstInt(llvm.Int16Type(), 0x100, false), "")
	c.dynStore(addr, 0x100, 0x1ff, v)
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

func (c *Compilation) getStatusByte() llvm.Value {
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
	return s0z
}

func (c *Compilation) cycle(count int, pc int) {
	// pc -1 means don't mess with the pc
	if pc >= 0 {
		c.builder.CreateStore(llvm.ConstInt(llvm.Int16Type(), uint64(pc), false), c.rPC)
	}
	c.debugPrint(fmt.Sprintf("cycles %d\n", count))
	c.debugPrintStatus()

	v := llvm.ConstInt(llvm.Int8Type(), uint64(count), false)
	c.builder.CreateCall(c.cycleFn, []llvm.Value{v}, "")
}

func (c *Compilation) debugPrint(str string) {
	c.debugPrintf(str, []llvm.Value{})
}

func (c *Compilation) getMemoizedStrGlob(str string) llvm.Value {
	glob, ok := c.stringTable[str]
	if !ok {
		text := llvm.ConstString(str, true)
		glob = llvm.AddGlobal(c.mod, text.Type(), "debugPrintStr")
		glob.SetLinkage(llvm.PrivateLinkage)
		glob.SetInitializer(text)
		glob.SetGlobalConstant(true)
		c.stringTable[str] = glob
	}
	return glob
}

func (c *Compilation) printf(str string, values []llvm.Value) {
	glob := c.getMemoizedStrGlob(str)
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	ptr := c.builder.CreatePointerCast(glob, bytePointerType, "")
	args := []llvm.Value{ptr}
	for _, v := range values {
		args = append(args, v)
	}
	c.builder.CreateCall(c.printfFn, args, "")
}

func (c *Compilation) debugPrintf(str string, values []llvm.Value) {
	if c.Flags&IncludeDebugFlag == 0 {
		return
	}
	c.printf(str, values)
}

func (c *Compilation) debugPrintStatus() {
	if c.Flags&IncludeDebugFlag != 0 {
		c.printf("A $%02x  X $%02x  Y $%02x  P $%02x  PC $%04x  SP $%02x\n", []llvm.Value{
			c.builder.CreateLoad(c.rA, ""),
			c.builder.CreateLoad(c.rX, ""),
			c.builder.CreateLoad(c.rY, ""),
			c.getStatusByte(),
			c.builder.CreateLoad(c.rPC, ""),
			c.builder.CreateLoad(c.rSP, ""),
		})
	}

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
	bb, ok := c.labeledBlocks[s.LabelName]
	if !ok {
		// we're not doing codegen for this block. skip.
		return
	}
	if c.currentBlock != nil {
		c.builder.CreateBr(bb)
	}
	c.currentBlock = &bb
	c.builder.SetInsertPointAtEnd(bb)
}

func (s *LabeledStatement) CompileLabels(c *Compilation) {
	// if it's a "data block" ignore it
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

func (c *Compilation) createPrgRomGlobal(prgRom [][]byte) {
	if len(prgRom) > 2 {
		panic("only 1-2 prg rom banks are supported")
	}
	dataLen := 0x8000
	prgDataValues := make([]llvm.Value, 0, dataLen)
	int8type := llvm.Int8Type()
	for len(prgDataValues) < dataLen {
		for _, bank := range prgRom {
			for _, b := range bank {
				lb := llvm.ConstInt(int8type, uint64(b), false)
				prgDataValues = append(prgDataValues, lb)
			}
		}
	}
	prgDataConst := llvm.ConstArray(llvm.ArrayType(int8type, len(prgDataValues)), prgDataValues)
	c.prgRom = llvm.AddGlobal(c.mod, prgDataConst.Type(), "rom_prg_data")
	c.prgRom.SetLinkage(llvm.PrivateLinkage)
	c.prgRom.SetInitializer(prgDataConst)
	c.prgRom.SetGlobalConstant(true)
}

func (c *Compilation) createReadChrFn(chrRom [][]byte) {
	//uint8_t rom_chr_bank_count;
	bankCountConst := llvm.ConstInt(llvm.Int8Type(), uint64(len(chrRom)), false)
	bankCountGlobal := llvm.AddGlobal(c.mod, bankCountConst.Type(), "rom_chr_bank_count")
	bankCountGlobal.SetLinkage(llvm.ExternalLinkage)
	bankCountGlobal.SetInitializer(bankCountConst)
	bankCountGlobal.SetGlobalConstant(true)

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
	chrDataGlobal.SetGlobalConstant(true)
	// void rom_read_chr(uint8_t* dest)
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	readChrType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{bytePointerType}, false)
	readChrFn := llvm.AddFunction(c.mod, "rom_read_chr", readChrType)
	readChrFn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(readChrFn, "Entry")
	c.builder.SetInsertPointAtEnd(entry)
	if dataLen > 0 {
		x2000 := llvm.ConstInt(llvm.Int32Type(), uint64(dataLen), false)
		source := c.builder.CreatePointerCast(chrDataGlobal, bytePointerType, "")
		c.builder.CreateCall(c.memcpyFn, []llvm.Value{readChrFn.Param(0), source, x2000}, "")
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
	// declare void @memcpy(void* dest, void* source, i32 size)
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	memcpyType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{bytePointerType, bytePointerType, llvm.Int32Type()}, false)
	c.memcpyFn = llvm.AddFunction(c.mod, "memcpy", memcpyType)
	c.memcpyFn.SetLinkage(llvm.ExternalLinkage)

	// declare i32 @putchar(i32)
	putCharType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{llvm.Int32Type()}, false)
	c.putCharFn = llvm.AddFunction(c.mod, "putchar", putCharType)
	c.putCharFn.SetLinkage(llvm.ExternalLinkage)

	// declare i32 @printf(i8*, ...)
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
	c.apuReadStatusFn = c.declareReadFn("rom_apu_read_status")
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

func (c *Compilation) addNmiInterruptCode() {
	c.builder.SetInsertPointBefore(c.nmiBlock.FirstInstruction())
	// * push PC high onto stack
	// * push PC low onto stack
	c.pushWordToStack(c.builder.CreateLoad(c.rPC, ""))
	// * push processor status onto stack
	c.pushToStack(c.getStatusByte())
}

func (c *Compilation) addResetInterruptCode() {
	c.builder.SetInsertPointBefore(c.resetBlock.FirstInstruction())
	// set registers
	c0 := llvm.ConstInt(llvm.Int8Type(), 0, false)
	xfd := llvm.ConstInt(llvm.Int8Type(), 0xfd, false)
	bit0 := llvm.ConstInt(llvm.Int1Type(), 0, false)
	bit1 := llvm.ConstInt(llvm.Int1Type(), 1, false)
	c.builder.CreateStore(c0, c.rX)
	c.builder.CreateStore(c0, c.rY)
	c.builder.CreateStore(c0, c.rA)
	c.builder.CreateStore(xfd, c.rSP)

	c.builder.CreateStore(bit0, c.rSNeg)
	c.builder.CreateStore(bit0, c.rSOver)
	c.builder.CreateStore(bit1, c.rSBrk)
	c.builder.CreateStore(bit0, c.rSDec)
	c.builder.CreateStore(bit1, c.rSInt)
	c.builder.CreateStore(bit0, c.rSZero)
	c.builder.CreateStore(bit0, c.rSCarry)
}

func (c *Compilation) setupControllerFramework() {
	// ROM_PAD_STATE_OFF = 0x40,
	x40 := llvm.ConstInt(llvm.Int8Type(), 0x40, false)
	initBtnArray := llvm.ConstArray(x40.Type(), []llvm.Value{
		x40, x40, x40, x40, x40, x40, x40, x40 })
	initPadArray := llvm.ConstArray(initBtnArray.Type(), []llvm.Value{
		initBtnArray,
		initBtnArray,
	})
	// btnReportIndex [2]int
	btnReportIndexType := llvm.ArrayType(llvm.Int8Type(), 2)
	c.btnReportIndex = llvm.AddGlobal(c.mod, btnReportIndexType, "ButtonReportIndex")
	c.btnReportIndex.SetLinkage(llvm.PrivateLinkage)
	c.btnReportIndex.SetInitializer(llvm.ConstNull(btnReportIndexType))
	// padsActual [2][8]byte
	c.padsActual = llvm.AddGlobal(c.mod, initPadArray.Type(), "PadsActual")
	c.padsActual.SetLinkage(llvm.PrivateLinkage)
	c.padsActual.SetInitializer(initPadArray)
	// padsReport [2][8]byte
	c.padsReport = llvm.AddGlobal(c.mod, initPadArray.Type(), "PadsReport")
	c.padsReport.SetLinkage(llvm.PrivateLinkage)
	c.padsReport.SetInitializer(initPadArray)
	// strobeOn bool
	c0 := llvm.ConstInt(llvm.Int1Type(), 0, false)
	c.strobeOn = llvm.AddGlobal(c.mod, c0.Type(), "StrobeOn")
	c.strobeOn.SetLinkage(llvm.PrivateLinkage)
	c.strobeOn.SetInitializer(c0)
	// void rom_set_button_state(uint8_t padIndex, uint8_t buttonIndex, uint8_t value);
	i8Type := llvm.Int8Type()
	setBtnStateType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{i8Type, i8Type, i8Type}, false)
	setBtnStateFn := llvm.AddFunction(c.mod, "rom_set_button_state", setBtnStateType)
	setBtnStateFn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(setBtnStateFn, "Entry")
	c.selectBlock(entry)
	// padsActual[padIndex][buttonIndex] = state
	c.builder.SetInsertPointAtEnd(entry)
	indexes := []llvm.Value{
		llvm.ConstInt(i8Type, 0, false),
		setBtnStateFn.Param(0),
		setBtnStateFn.Param(1),
	}
	ptr := c.builder.CreateGEP(c.padsActual, indexes, "")
	c.builder.CreateStore(setBtnStateFn.Param(2), ptr)
	// if strobeOn then padsReport[padIndex][buttonIndex] = state
	strobeOn := c.builder.CreateLoad(c.strobeOn, "")
	elseBlock := c.createIf(strobeOn)
	ptr = c.builder.CreateGEP(c.padsReport, indexes, "")
	c.builder.CreateStore(setBtnStateFn.Param(2), ptr)
	c.builder.CreateRetVoid()
	c.selectBlock(elseBlock)
	c.builder.CreateRetVoid()

	// these functions are private to this module
	c.createPadWriteFn()
	c.createPadReadFn()

}

func (c *Compilation) createPadWriteFn() {
	i8Type := llvm.Int8Type()
	// void padWrite(uint8_t value)
	padWriteType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{i8Type}, false)
	c.padWriteFn = llvm.AddFunction(c.mod, "padWrite", padWriteType)
	c.padWriteFn.SetLinkage(llvm.PrivateLinkage)
	entry := llvm.AddBasicBlock(c.padWriteFn, "Entry")
	c.selectBlock(entry)
	// StrobeOn = value&0x1 == 1
	c0 := llvm.ConstInt(i8Type, 0, false)
	c1 := llvm.ConstInt(i8Type, 1, false)
	masked := c.builder.CreateAnd(c.padWriteFn.Param(0), c1, "")
	isOn := c.builder.CreateICmp(llvm.IntNE, masked, c0, "")
	c.builder.CreateStore(isOn, c.strobeOn)
	// if c.StrobeOn {
	elseBlock := c.createIf(isOn)
	//     memcpy(padsReport, padsActual, 16)
	c16 := llvm.ConstInt(llvm.Int32Type(), 16, false)
	bytePointerType := llvm.PointerType(i8Type, 0)
	dest := c.builder.CreatePointerCast(c.padsReport, bytePointerType, "")
	source := c.builder.CreatePointerCast(c.padsActual, bytePointerType, "")
	c.builder.CreateCall(c.memcpyFn, []llvm.Value{dest, source, c16}, "")
	//     btnReportIndex[0] = 0
	indexes := []llvm.Value{
		c0,
		c0,
	}
	ptr := c.builder.CreateGEP(c.btnReportIndex, indexes, "")
	c.builder.CreateStore(c0, ptr)
	//     btnReportIndex[1] = 0
	indexes = []llvm.Value{
		c0,
		c1,
	}
	ptr = c.builder.CreateGEP(c.btnReportIndex, indexes, "")
	c.builder.CreateStore(c0, ptr)
	// }
	c.builder.CreateRetVoid()
	c.selectBlock(elseBlock)
	c.builder.CreateRetVoid()
}

func (c *Compilation) createPadReadFn() {
	i8Type := llvm.Int8Type()
	c0 := llvm.ConstInt(i8Type, 0, false)
	c1 := llvm.ConstInt(i8Type, 1, false)
	c8 := llvm.ConstInt(i8Type, 8, false)
	// uint8_t padRead(uint8_t padIndex)
	padReadType := llvm.FunctionType(i8Type, []llvm.Type{i8Type}, false)
	c.padReadFn = llvm.AddFunction(c.mod, "padRead", padReadType)
	c.padReadFn.SetLinkage(llvm.PrivateLinkage)
	entry := llvm.AddBasicBlock(c.padReadFn, "Entry")
	c.selectBlock(entry)
	// if btnReportIndex[padIndex] >= 8 {
	indexes := []llvm.Value{
		c0,
		c.padReadFn.Param(0),
	}
	btnReportIndexPtr := c.builder.CreateGEP(c.btnReportIndex, indexes, "")
	btnReportIndex := c.builder.CreateLoad(btnReportIndexPtr, "")
	isOb := c.builder.CreateICmp(llvm.IntUGE, btnReportIndex, c8, "")
	notObBlock := c.createIf(isOb)
	//     return 1
	c.builder.CreateRet(c1)
	// }
	c.selectBlock(notObBlock)
	// v := padsReport[padIndex][btnReportIndex[padIndex]]
	indexes = []llvm.Value{
		c0,
		c.padReadFn.Param(0),
		btnReportIndex,
	}
	padsReportPtr := c.builder.CreateGEP(c.padsReport, indexes, "")
	v := c.builder.CreateLoad(padsReportPtr, "")
	// if c.StrobeOn {
	strobeOn := c.builder.CreateLoad(c.strobeOn, "")
	elseBlock := c.createIf(strobeOn)
	//     btnReportIndex[padIndex] = 0
	c.builder.CreateStore(c0, btnReportIndexPtr)
	//     return v
	c.builder.CreateRet(v)
	// } else {
	c.selectBlock(elseBlock)
	//     btnReportIndex[padIndex] += 1
	btnReportIndex = c.builder.CreateAdd(btnReportIndex, c1, "")
	c.builder.CreateStore(btnReportIndex, btnReportIndexPtr)
	//     return v
	c.builder.CreateRet(v)
	// }
}

func (c *Compilation) createReadMemFn() {
	// uint8_t rom_ram_read(uint16_t addr)
	readMemType := llvm.FunctionType(llvm.Int8Type(), []llvm.Type{llvm.Int16Type()}, false)
	readMemFn := llvm.AddFunction(c.mod, "rom_ram_read", readMemType)
	entry := llvm.AddBasicBlock(readMemFn, "Entry")
	c.selectBlock(entry)
	v := c.dynLoad(readMemFn.Param(0), 0, 0xffff)
	c.builder.CreateRet(v)
}

func (p *Program) CompileToFile(file *os.File, flags CompileFlags) (*Compilation, error) {
	llvm.InitializeNativeTarget()

	c := new(Compilation)
	c.Flags = flags
	c.program = p
	c.mod = llvm.NewModule("asm_module")
	c.builder = llvm.NewBuilder()
	defer c.builder.Dispose()
	c.labeledData = map[string]bool{}
	c.labeledBlocks = map[string]llvm.BasicBlock{}
	c.stringTable = map[string]llvm.Value{}
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

	c.createFunctionDeclares()
	c.createReadChrFn(p.ChrRom)
	c.createPrgRomGlobal(p.PrgRom)

	// first pass to figure out which blocks are "data" and which are "code"
	c.mode = dataStmtMode
	c.currentLabel = ""
	c.currentExpecting = cfExpectNone
	p.Ast.Ast(c)
	if len(c.Errors) > 0 {
		return c, nil
	}

	c.setupControllerFramework()
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
	c.currentBlock = nil
	c.mode = compileMode
	p.Ast.Ast(c)

	c.createReadMemFn()

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
	c.createPanic("invalid interrupt id: %d\n", []llvm.Value{c.mainFn.Param(0)})
	sw.AddCase(llvm.ConstInt(llvm.Int32Type(), 1, false), *c.nmiBlock)
	sw.AddCase(llvm.ConstInt(llvm.Int32Type(), 2, false), *c.resetBlock)
	sw.AddCase(llvm.ConstInt(llvm.Int32Type(), 3, false), *c.irqBlock)
	for labelName, labelId := range c.labelIds {
		sw.AddCase(llvm.ConstInt(llvm.Int32Type(), uint64(labelId), false), c.labeledBlocks[labelName])
	}

	c.addNmiInterruptCode()
	c.addResetInterruptCode()

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
