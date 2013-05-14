package jamulator

import (
	"fmt"
	"github.com/axw/gollvm/llvm"
)

func (i *Instruction) Compile(c *Compilation) {
	c.debugPrint(fmt.Sprintf("%s\n", i.Render()))

	var labelAddr int
	var ok bool
	if i.LabelName != "" {
		labelAddr, ok = c.program.Labels[i.LabelName]
		if !ok {
			panic(fmt.Sprintf("label %s addr not defined: %s", i.LabelName, i.Render()))
		}
	}
	var immedValue llvm.Value
	if i.Type == ImmediateInstruction {
		immedValue = llvm.ConstInt(llvm.Int8Type(), uint64(i.Value), false)
	}

	var addrNext = i.Offset+len(i.Payload)

	switch i.OpCode {
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("unrecognized instruction: %s", i.Render()))
	case 0xa2: // ldx immediate
		c.builder.CreateStore(immedValue, c.rX)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, addrNext)
	case 0xa0: // ldy immediate
		c.builder.CreateStore(immedValue, c.rY)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, addrNext)
	case 0xa9: // lda immediate
		c.builder.CreateStore(immedValue, c.rA)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, addrNext)
	case 0x69: // adc immediate
		c.performAdc(immedValue)
		c.cycle(2, addrNext)
	case 0xe9: // sbc immediate
		c.performSbc(immedValue)
		c.cycle(2, addrNext)
	case 0x29: // and immediate
		c.performAnd(immedValue)
		c.cycle(2, addrNext)
	case 0xc9: // cmp immediate
		reg := c.builder.CreateLoad(c.rA, "")
		c.performCmp(reg, immedValue)
		c.cycle(2, addrNext)
	case 0xe0: // cpx immediate
		reg := c.builder.CreateLoad(c.rX, "")
		c.performCmp(reg, immedValue)
		c.cycle(2, addrNext)
	case 0xc0: // cpy immediate
		reg := c.builder.CreateLoad(c.rY, "")
		c.performCmp(reg, immedValue)
		c.cycle(2, addrNext)
	case 0x49: // eor immediate
		c.performEor(immedValue)
		c.cycle(2, addrNext)
	case 0x09: // ora immediate
		a := c.builder.CreateLoad(c.rA, "")
		newA := c.builder.CreateOr(a, immedValue, "")
		c.builder.CreateStore(newA, c.rA)
		c.dynTestAndSetZero(newA)
		c.dynTestAndSetNeg(newA)
		c.cycle(2, addrNext)

	case 0x0a: // asl implied
		a := c.builder.CreateLoad(c.rA, "")
		c.builder.CreateStore(c.performAsl(a), c.rA)
		c.cycle(2, addrNext)
	//case 0x00: // brk implied
	case 0x18: // clc implied
		c.clearCarry()
		c.cycle(2, addrNext)
	case 0x38: // sec implied
		c.setCarry()
		c.cycle(2, addrNext)
	case 0xd8: // cld implied
		c.clearDec()
		c.cycle(2, addrNext)
	case 0x58: // cli implied
		c.clearInt()
		c.cycle(2, addrNext)
	case 0xb8: // clv implied
		c.clearOverflow()
		c.cycle(2, addrNext)
	case 0xca: // dex implied
		c.increment(c.rX, -1)
		c.cycle(2, addrNext)
	case 0x88: // dey implied
		c.increment(c.rY, -1)
		c.cycle(2, addrNext)
	case 0xe8: // inx implied
		c.increment(c.rX, 1)
		c.cycle(2, addrNext)
	case 0xc8: // iny implied
		c.increment(c.rY, 1)
		c.cycle(2, addrNext)
	case 0x4a: // lsr implied
		oldValue := c.builder.CreateLoad(c.rA, "")
		c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
		newValue := c.builder.CreateLShr(oldValue, c1, "")
		c.builder.CreateStore(newValue, c.rA)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetNeg(newValue)
		c.dynTestAndSetCarryLShr(oldValue)
		c.cycle(2, addrNext)
	case 0xea: // nop implied
		c.cycle(2, addrNext)
	case 0x48: // pha implied
		a := c.builder.CreateLoad(c.rA, "")
		c.pushToStack(a)
		c.cycle(3, addrNext)
	case 0x68: // pla implied
		v := c.pullFromStack()
		c.builder.CreateStore(v, c.rA)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		c.cycle(4, addrNext)
	//case 0x08: // php implied
	case 0x28: // plp implied
		c.pullStatusReg()
		c.cycle(4, addrNext)
	case 0x2a: // rol implied
		a := c.builder.CreateLoad(c.rA, "")
		c.builder.CreateStore(c.performRol(a), c.rA)
		c.cycle(2, addrNext)
	case 0x6a: // ror implied
		a := c.builder.CreateLoad(c.rA, "")
		c.builder.CreateStore(c.performRor(a), c.rA)
		c.cycle(2, addrNext)
	case 0x40: // rti implied
		c.pullStatusReg()
		pc := c.pullWordFromStack()
		c.builder.CreateStore(pc, c.rPC)
		c.cycle(6, -1) // -1 because we already stored the PC
		c.builder.CreateRetVoid()
		c.currentBlock = nil
	case 0x60: // rts implied
		pc := c.pullWordFromStack()
		pc = c.builder.CreateAdd(pc, llvm.ConstInt(pc.Type(), 1, false), "")
		c.debugPrintf("rts: new pc $%04x\n", []llvm.Value{pc})
		c.builder.CreateStore(pc, c.rPC)
		c.cycle(6, -1)
		c.builder.CreateRetVoid()
		c.currentBlock = nil
	case 0xf8: // sed implied
		c.setDec()
		c.cycle(2, addrNext)
	case 0x78: // sei implied
		c.setInt()
		c.cycle(2, addrNext)
	case 0xaa: // tax implied
		c.transfer(c.rA, c.rX)
		c.cycle(2, addrNext)
	case 0xa8: // tay implied
		c.transfer(c.rA, c.rY)
		c.cycle(2, addrNext)
	case 0xba: // tsx implied
		c.transfer(c.rSP, c.rX)
		c.cycle(2, addrNext)
	case 0x8a: // txa implied
		c.transfer(c.rX, c.rA)
		c.cycle(2, addrNext)
	case 0x9a: // txs implied
		// TXS does not set flags
		v := c.builder.CreateLoad(c.rX, "")
		c.builder.CreateStore(v, c.rSP)
		c.cycle(2, addrNext)
	case 0x98: // tya implied
		c.transfer(c.rY, c.rA)
		c.cycle(2, addrNext)

	case 0x79: // adc abs y
		v := c.dynLoadIndexed(i.Value, c.rY)
		c.performAdc(v)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rY, addrNext)
	case 0xf9: // sbc abs y
		v := c.dynLoadIndexed(i.Value, c.rY)
		c.performSbc(v)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rY, addrNext)
	case 0xd9: // cmp abs y
		reg := c.builder.CreateLoad(c.rA, "")
		mem := c.dynLoadIndexed(i.Value, c.rY)
		c.performCmp(reg, mem)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rX, addrNext)
	case 0xdd: // cmp abs x
		reg := c.builder.CreateLoad(c.rA, "")
		mem := c.dynLoadIndexed(i.Value, c.rX)
		c.performCmp(reg, mem)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rX, addrNext)
	case 0xd5: // cmp zpg x
		reg := c.builder.CreateLoad(c.rA, "")
		mem := c.dynLoadZpgIndexed(i.Value, c.rX)
		c.performCmp(reg, mem)
		c.cycle(4, addrNext)
	case 0xb9: // lda abs y
		c.absoluteIndexedLoad(c.rA, i.Value, c.rY, addrNext)
	case 0xbe: // ldx abs y
		c.absoluteIndexedLoad(c.rX, i.Value, c.rY, addrNext)
	case 0xbd: // lda abs x
		c.absoluteIndexedLoad(c.rA, i.Value, c.rX, addrNext)
	case 0xbc: // ldy abs x
		c.absoluteIndexedLoad(c.rY, i.Value, c.rX, addrNext)
	case 0x99: // sta abs y
		c.absoluteIndexedStore(c.rA, i.Value, c.rY, addrNext)
	case 0x9d: // sta abs x
		c.absoluteIndexedStore(c.rA, i.Value, c.rX, addrNext)
	case 0x96: // stx zpg y
		v := c.builder.CreateLoad(c.rX, "")
		c.dynStoreZpgIndexed(i.Value, c.rY, v)
		c.cycle(4, addrNext)
	case 0x95: // sta zpg x
		v := c.builder.CreateLoad(c.rA, "")
		c.dynStoreZpgIndexed(i.Value, c.rX, v)
		c.cycle(4, addrNext)
	case 0x94: // sty zpg x
		v := c.builder.CreateLoad(c.rY, "")
		c.dynStoreZpgIndexed(i.Value, c.rX, v)
		c.cycle(4, addrNext)
	case 0xb6: // ldx zpg y
		v := c.dynLoadZpgIndexed(i.Value, c.rY)
		c.builder.CreateStore(v, c.rX)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		c.cycle(4, addrNext)
	case 0xb4: // ldy zpg x
		v := c.dynLoadZpgIndexed(i.Value, c.rX)
		c.builder.CreateStore(v, c.rY)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		c.cycle(4, addrNext)
	case 0xb5: // lda zpg x
		v := c.dynLoadZpgIndexed(i.Value, c.rX)
		c.builder.CreateStore(v, c.rA)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		c.cycle(4, addrNext)
	case 0x7d: // adc abs x
		v := c.dynLoadIndexed(i.Value, c.rX)
		c.performAdc(v)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rX, addrNext)
	case 0xfd: // sbc abs x
		v := c.dynLoadIndexed(i.Value, c.rX)
		c.performSbc(v)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rX, addrNext)
	case 0x75: // adc zpg x
		v := c.dynLoadZpgIndexed(i.Value, c.rX)
		c.performAdc(v)
		c.cycle(4, addrNext)
	case 0xf5: // sbc zpg x
		v := c.dynLoadZpgIndexed(i.Value, c.rX)
		c.performSbc(v)
		c.cycle(4, addrNext)
	case 0x1e: // asl abs x
		oldValue := c.dynLoadIndexed(i.Value, c.rX)
		newValue := c.performAsl(oldValue)
		c.dynStoreIndexed(i.Value, c.rX, newValue)
		c.cycle(7, addrNext)
	case 0x16: // asl zpg x
		oldValue := c.dynLoadZpgIndexed(i.Value, c.rX)
		newValue := c.performAsl(oldValue)
		c.dynStoreZpgIndexed(i.Value, c.rX, newValue)
		c.cycle(6, addrNext)
	case 0xde: // dec abs x
		oldValue := c.dynLoadIndexed(i.Value, c.rX)
		newValue := c.incrementVal(oldValue, -1)
		c.dynStoreIndexed(i.Value, c.rX, newValue)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetNeg(newValue)
		c.cycle(7, addrNext)
	case 0xfe: // inc abs x
		oldValue := c.dynLoadIndexed(i.Value, c.rX)
		newValue := c.incrementVal(oldValue, 1)
		c.dynStoreIndexed(i.Value, c.rX, newValue)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetNeg(newValue)
		c.cycle(7, addrNext)
	case 0xd6: // dec zpg x
		oldValue := c.dynLoadZpgIndexed(i.Value, c.rX)
		newValue := c.incrementVal(oldValue, -1)
		c.dynStoreZpgIndexed(i.Value, c.rX, newValue)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetNeg(newValue)
		c.cycle(6, addrNext)
	case 0xf6: // inc zpg x
		oldValue := c.dynLoadZpgIndexed(i.Value, c.rX)
		newValue := c.incrementVal(oldValue, 1)
		c.dynStoreZpgIndexed(i.Value, c.rX, newValue)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetNeg(newValue)
		c.cycle(6, addrNext)
	case 0x3e: // rol abs x
		oldValue := c.dynLoadIndexed(i.Value, c.rX)
		newValue := c.performRol(oldValue)
		c.dynStoreIndexed(i.Value, c.rX, newValue)
		c.cycle(7, addrNext)
	case 0x7e: // ror abs x
		oldValue := c.dynLoadIndexed(i.Value, c.rX)
		newValue := c.performRor(oldValue)
		c.dynStoreIndexed(i.Value, c.rX, newValue)
		c.cycle(7, addrNext)
	case 0x39: // and abs y
		v := c.dynLoadIndexed(i.Value, c.rY)
		c.performAnd(v)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rY, addrNext)
	case 0x3d: // and abs x
		v := c.dynLoadIndexed(i.Value, c.rX)
		c.performAnd(v)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rX, addrNext)
	case 0x35: // and zpg x
		v := c.dynLoadZpgIndexed(i.Value, c.rX)
		c.performAnd(v)
		c.cycle(4, addrNext)
	case 0x5d: // eor abs x
		v := c.dynLoadIndexed(i.Value, c.rX)
		c.performEor(v)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rX, addrNext)
	case 0x55: // eor zpg x
		v := c.dynLoadZpgIndexed(i.Value, c.rX)
		c.performEor(v)
		c.cycle(4, addrNext)
	case 0x59: // eor abs y
		v := c.dynLoadIndexed(i.Value, c.rY)
		c.performEor(v)
		c.cyclesForAbsoluteIndexedPtr(i.Value, c.rY, addrNext)
	//case 0x19: // ora abs y
	//case 0x5e: // lsr abs x
	//case 0x1d: // ora abs x
	//case 0x56: // lsr zpg x
	//case 0x15: // ora zpg x
	//case 0x36: // rol zpg x
	//case 0x76: // ror zpg x

	//case 0x6c: // jmp indirect

	case 0x4c: // jmp
		// branch instruction - cycle before execution
		c.cycle(3, labelAddr)
		destBlock, ok := c.labeledBlocks[i.LabelName]
		if !ok {
			panic(fmt.Sprintf("label %s block not defined: %s", i.LabelName, i.Render()))
		}
		c.builder.CreateBr(destBlock)
		c.currentBlock = nil
	case 0x20: // jsr
		pc := llvm.ConstInt(llvm.Int16Type(), uint64(i.Offset+2), false)

		c.debugPrintf("jsr: saving $%04x\n", []llvm.Value{pc})

		c.pushWordToStack(pc)
		c.cycle(6, labelAddr)
		id := c.labelAsEntryPoint(i.LabelName)
		c.builder.CreateCall(c.mainFn, []llvm.Value{llvm.ConstInt(llvm.Int32Type(), uint64(id), false)}, "")

	case 0xf0: // beq
		isZero := c.builder.CreateLoad(c.rSZero, "")
		c.createBranch(isZero, i.LabelName, i.Offset)
	case 0x90: // bcc
		isCarry := c.builder.CreateLoad(c.rSCarry, "")
		notCarry := c.builder.CreateNot(isCarry, "")
		c.createBranch(notCarry, i.LabelName, i.Offset)
	case 0xb0: // bcs
		isCarry := c.builder.CreateLoad(c.rSCarry, "")
		c.createBranch(isCarry, i.LabelName, i.Offset)
	case 0x30: // bmi
		isNeg := c.builder.CreateLoad(c.rSNeg, "")
		c.createBranch(isNeg, i.LabelName, i.Offset)
	case 0xd0: // bne
		isZero := c.builder.CreateLoad(c.rSZero, "")
		notZero := c.builder.CreateNot(isZero, "")
		c.createBranch(notZero, i.LabelName, i.Offset)
	case 0x10: // bpl
		isNeg := c.builder.CreateLoad(c.rSNeg, "")
		notNeg := c.builder.CreateNot(isNeg, "")
		c.createBranch(notNeg, i.LabelName, i.Offset)
	//case 0x50: // bvc
	//case 0x70: // bvs


	case 0xa5, 0xad: // lda (zpg, abs)
		v := c.load(i.Value)
		c.builder.CreateStore(v, c.rA)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		if i.OpCode == 0xa5 {
			c.cycle(3, addrNext)
		} else {
			c.cycle(4, addrNext)
		}
	case 0xa4, 0xac: // ldy (zpg, abs)
		v := c.load(i.Value)
		c.builder.CreateStore(v, c.rY)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		if i.OpCode == 0xa4 {
			c.cycle(3, addrNext)
		} else {
			c.cycle(4, addrNext)
		}
	case 0xa6, 0xae: // ldx (zpg, abs)
		v := c.load(i.Value)
		c.builder.CreateStore(v, c.rX)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		if i.OpCode == 0xa6 {
			c.cycle(3, addrNext)
		} else {
			c.cycle(4, addrNext)
		}
	case 0xc6: // dec zpg
		c.incrementMem(i.Value, -1)
		c.cycle(5, addrNext)
	case 0xce: // dec abs
		c.incrementMem(i.Value, -1)
		c.cycle(6, addrNext)
	case 0xe6: // inc zpg
		c.incrementMem(i.Value, 1)
		c.cycle(5, addrNext)
	case 0xee: // inc abs
		c.incrementMem(i.Value, 1)
		c.cycle(6, addrNext)
	case 0x46, 0x4e: // lsr (zpg, abs)
		oldValue := c.load(i.Value)
		c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
		newValue := c.builder.CreateLShr(oldValue, c1, "")
		c.store(i.Value, newValue)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetCarryLShr(oldValue)
		if i.OpCode == 0x46 {
			c.cycle(5, addrNext)
		} else {
			c.cycle(6, addrNext)
		}
	case 0x45: // eor zpg
		c.performEor(c.load(i.Value))
		c.cycle(3, addrNext)
	case 0x4d: // eor abs
		c.performEor(c.load(i.Value))
		c.cycle(4, addrNext)
	case 0xc5: // cmp zpg
		reg := c.builder.CreateLoad(c.rA, "")
		c.performCmp(reg, c.load(i.Value))
		c.cycle(3, addrNext)
	case 0xcd: // cmp abs
		reg := c.builder.CreateLoad(c.rA, "")
		c.performCmp(reg, c.load(i.Value))
		c.cycle(4, addrNext)
	case 0x65: // adc zpg
		c.performAdc(c.load(i.Value))
		c.cycle(3, addrNext)
	case 0x6d: // adc abs
		c.performAdc(c.load(i.Value))
		c.cycle(4, addrNext)
	case 0xe5: // sbc zpg
		c.performSbc(c.load(i.Value))
		c.cycle(3, addrNext)
	case 0xed: // sbc abs
		c.performSbc(c.load(i.Value))
		c.cycle(4, addrNext)
	case 0x05, 0x0d: // ora (zpg, abs)
		a := c.builder.CreateLoad(c.rA, "")
		mem := c.load(i.Value)
		newA := c.builder.CreateOr(a, mem, "")
		c.builder.CreateStore(newA, c.rA)
		c.dynTestAndSetZero(newA)
		c.dynTestAndSetNeg(newA)
		if i.OpCode == 0x05 {
			c.cycle(3, addrNext)
		} else {
			c.cycle(4, addrNext)
		}
	case 0x25: // and zpg
		c.performAnd(c.load(i.Value))
		c.cycle(3, addrNext)
	case 0x2d: // and abs
		c.performAnd(c.load(i.Value))
		c.cycle(4, addrNext)
	case 0x24: // bit zpg
		c.performBit(c.load(i.Value))
		c.cycle(3, addrNext)
	case 0x2c: // bit abs
		c.performBit(c.load(i.Value))
		c.cycle(4, addrNext)
	case 0x06: // asl zpg
		oldValue := c.load(i.Value)
		newValue := c.performAsl(oldValue)
		c.store(i.Value, newValue)
		c.cycle(5, addrNext)
	case 0x0e: // asl abs
		oldValue := c.load(i.Value)
		newValue := c.performAsl(oldValue)
		c.store(i.Value, newValue)
		c.cycle(6, addrNext)
	//case 0x90: // bcc rel
	//case 0xb0: // bcs rel
	//case 0xf0: // beq rel
	//case 0x30: // bmi rel
	//case 0xd0: // bne rel
	//case 0x10: // bpl rel
	//case 0x50: // bvc rel
	//case 0x70: // bvs rel

	//case 0xe4: // cpx zpg
	//case 0xc4: // cpy zpg
	case 0x26: // rol zpg
		oldValue := c.load(i.Value)
		newValue := c.performRol(oldValue)
		c.store(i.Value, newValue)
		c.cycle(5, addrNext)
	case 0x66: // ror zpg
		oldValue := c.load(i.Value)
		newValue := c.performRor(oldValue)
		c.store(i.Value, newValue)
		c.cycle(5, addrNext)
	case 0x2e: // rol abs
		oldValue := c.load(i.Value)
		newValue := c.performRol(oldValue)
		c.store(i.Value, newValue)
		c.cycle(6, addrNext)
	case 0x6e: // ror abs
		oldValue := c.load(i.Value)
		newValue := c.performRor(oldValue)
		c.store(i.Value, newValue)
		c.cycle(6, addrNext)

	//case 0xec: // cpx abs
	//case 0xcc: // cpy abs
	//case 0x4c: // jmp abs
	//case 0x20: // jsr abs
	case 0x85: // sta zpg
		c.store(i.Value, c.builder.CreateLoad(c.rA, ""))
		c.cycle(3, addrNext)
	case 0x8d: // sta abs
		c.store(i.Value, c.builder.CreateLoad(c.rA, ""))
		c.cycle(4, addrNext)
	case 0x86: // stx zpg
		c.store(i.Value, c.builder.CreateLoad(c.rX, ""))
		c.cycle(3, addrNext)
	case 0x8e: // stx abs
		c.store(i.Value, c.builder.CreateLoad(c.rX, ""))
		c.cycle(4, addrNext)
	case 0x84: // sty zpg
		c.store(i.Value, c.builder.CreateLoad(c.rY, ""))
		c.cycle(3, addrNext)
	case 0x8c: // sty abs
		c.store(i.Value, c.builder.CreateLoad(c.rY, ""))
		c.cycle(4, addrNext)

	case 0xa1: // lda indirect x
		index := c.builder.CreateLoad(c.rX, "")
		base := llvm.ConstInt(llvm.Int8Type(), uint64(i.Value), false)
		addr := c.builder.CreateAdd(base, index, "")
		v := c.dynLoad(addr, 0, 0xff)
		c.builder.CreateStore(v, c.rA)
		c.cycle(6, addrNext)
	//case 0x61: // adc indirect x
	//case 0x21: // and indirect x
	//case 0xc1: // cmp indirect x
	//case 0x41: // eor indirect x
	//case 0x01: // ora indirect x
	//case 0xe1: // sbc indirect x
	//case 0x81: // sta indirect x


	//case 0x71: // adc indirect y
	//case 0x31: // and indirect y
	//case 0xd1: // cmp indirect y
	//case 0x51: // eor indirect y
	case 0xb1: // lda indirect y
		baseAddr := c.loadWord(i.Value)
		rY := c.builder.CreateLoad(c.rY, "")
		rYw := c.builder.CreateZExt(rY, llvm.Int16Type(), "")
		addr := c.builder.CreateAdd(baseAddr, rYw, "")
		val := c.dynLoad(addr, 0, 0xffff)
		c.builder.CreateStore(val, c.rA)
		c.dynTestAndSetNeg(val)
		c.dynTestAndSetZero(val)
		c.cyclesForIndirectY(baseAddr, addr, addrNext)
	//case 0x11: // ora indirect y
	//case 0xf1: // sbc indirect y
	case 0x91: // sta indirect y
		baseAddr := c.loadWord(i.Value)
		rY := c.builder.CreateLoad(c.rY, "")
		rYw := c.builder.CreateZExt(rY, llvm.Int16Type(), "")
		addr := c.builder.CreateAdd(baseAddr, rYw, "")
		rA := c.builder.CreateLoad(c.rA, "")
		c.dynStore(addr, 0, 0xffff, rA)
		c.cycle(6, addrNext)
	}
}
