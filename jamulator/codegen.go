package jamulator

import (
	"fmt"
	"github.com/axw/gollvm/llvm"
)

func (i *ImmediateInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	v := llvm.ConstInt(llvm.Int8Type(), uint64(i.Value), false)
	switch i.OpCode {
	case 0xa2: // ldx
		c.builder.CreateStore(v, c.rX)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, i.Offset+i.Size)
	case 0xa0: // ldy
		c.builder.CreateStore(v, c.rY)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, i.Offset+i.Size)
	case 0xa9: // lda
		c.builder.CreateStore(v, c.rA)
		c.testAndSetZero(i.Value)
		c.testAndSetNeg(i.Value)
		c.cycle(2, i.Offset+i.Size)
	case 0x69: // adc
		a := c.builder.CreateLoad(c.rA, "")
		aPlusV := c.builder.CreateAdd(a, v, "")
		carryBit := c.builder.CreateLoad(c.rSCarry, "")
		carry := c.builder.CreateZExt(carryBit, llvm.Int8Type(), "")
		newA := c.builder.CreateAdd(aPlusV, carry, "")
		c.dynTestAndSetNeg(newA)
		c.dynTestAndSetZero(newA)
		c.dynTestAndSetOverflowAddition(a, v, newA)
		c.dynTestAndSetCarryAddition(a, v, carry)

		c.cycle(2, i.Offset+i.Size)
	case 0x29: // and
		a := c.builder.CreateLoad(c.rA, "")
		newA := c.builder.CreateAnd(a, v, "")
		c.builder.CreateStore(newA, c.rA)
		c.dynTestAndSetZero(newA)
		c.dynTestAndSetNeg(newA)
		c.cycle(2, i.Offset+i.Size)
	case 0xc9: // cmp
		c.createCompare(c.rA, v)
		c.cycle(2, i.Offset+i.Size)
	case 0xe0: // cpx
		c.createCompare(c.rX, v)
		c.cycle(2, i.Offset+i.Size)
	case 0xc0: // cpy
		c.createCompare(c.rY, v)
		c.cycle(2, i.Offset+i.Size)
	case 0x49: // eor
		a := c.builder.CreateLoad(c.rA, "")
		newA := c.builder.CreateXor(a, v, "")
		c.builder.CreateStore(newA, c.rA)
		c.dynTestAndSetZero(newA)
		c.dynTestAndSetNeg(newA)
		c.cycle(2, i.Offset+i.Size)
	case 0x09: // ora
		a := c.builder.CreateLoad(c.rA, "")
		newA := c.builder.CreateOr(a, v, "")
		c.builder.CreateStore(newA, c.rA)
		c.dynTestAndSetZero(newA)
		c.dynTestAndSetNeg(newA)
		c.cycle(2, i.Offset+i.Size)
	//case 0xe9: // sbc
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s immediate lacks Compile() implementation", i.OpName))
	}
}

func (i *ImpliedInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	switch i.OpCode {
	//case 0x0a: // asl
	//case 0x00: // brk
	//case 0x18: // clc
	case 0xd8: // cld
		c.clearDec()
		c.cycle(2, i.Offset+i.Size)
	case 0x58: // cli
		c.clearInt()
		c.cycle(2, i.Offset+i.Size)
	//case 0xb8: // clv
	case 0xca: // dex
		c.increment(c.rX, -1)
		c.cycle(2, i.Offset+i.Size)
	case 0x88: // dey
		c.increment(c.rY, -1)
		c.cycle(2, i.Offset+i.Size)
	case 0xe8: // inx
		c.increment(c.rX, 1)
		c.cycle(2, i.Offset+i.Size)
	case 0xc8: // iny
		c.increment(c.rY, 1)
		c.cycle(2, i.Offset+i.Size)
	case 0x4a: // lsr
		oldValue := c.builder.CreateLoad(c.rA, "")
		c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
		newValue := c.builder.CreateLShr(oldValue, c1, "")
		c.builder.CreateStore(newValue, c.rA)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetCarryLShr(oldValue)
		c.cycle(2, i.Offset+i.Size)
	case 0xea: // nop
		c.cycle(2, i.Offset+i.Size)
	//case 0x48: // pha
	//case 0x08: // php
	//case 0x68: // pla
	case 0x28: // plp
		c.pullStatusReg()
		c.cycle(4, i.Offset+i.Size)
	//case 0x2a: // rol
	//case 0x6a: // ror
	case 0x40: // rti
		c.pullStatusReg()
		pc := c.pullWordFromStack()
		c.builder.CreateStore(pc, c.rPC)
		c.cycle(6, -1) // -1 because we already stored the PC
		c.builder.CreateRetVoid()
		c.currentBlock = nil
	case 0x60: // rts
		pc := c.pullWordFromStack()
		pc = c.builder.CreateAdd(pc, llvm.ConstInt(llvm.Int16Type(), 1, false), "")
		c.builder.CreateStore(pc, c.rPC)
		c.cycle(6, -1)
		c.builder.CreateRetVoid()
		c.currentBlock = nil
	//case 0x38: // sec
	case 0xf8: // sed
		c.setDec()
		c.cycle(2, i.Offset+i.Size)
	case 0x78: // sei
		c.setInt()
		c.cycle(2, i.Offset+i.Size)
	case 0xaa: // tax
		c.transfer(c.rA, c.rX)
		c.cycle(2, i.Offset+i.Size)
	case 0xa8: // tay
		c.transfer(c.rA, c.rY)
		c.cycle(2, i.Offset+i.Size)
	case 0xba: // tsx
		c.transfer(c.rSP, c.rX)
		c.cycle(2, i.Offset+i.Size)
	case 0x8a: // txa
		c.transfer(c.rX, c.rA)
		c.cycle(2, i.Offset+i.Size)
	case 0x9a: // txs
		c.transfer(c.rX, c.rSP)
		c.cycle(2, i.Offset+i.Size)
	case 0x98: // tya
		c.transfer(c.rY, c.rA)
		c.cycle(2, i.Offset+i.Size)
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
		c.absoluteIndexedLoadData(c.rA, i.LabelName, c.rX, i.Offset+i.Size)
	case 0xb9: // lda l, Y
		c.absoluteIndexedLoadData(c.rA, i.LabelName, c.rY, i.Offset+i.Size)
	case 0xbe: // ldx l, Y
		c.absoluteIndexedLoadData(c.rX, i.LabelName, c.rY, i.Offset+i.Size)
	case 0xbc: // ldy l, X
		c.absoluteIndexedLoadData(c.rY, i.LabelName, c.rX, i.Offset+i.Size)
	//case 0x7d: // adc l, X
	//case 0x3d: // and l, X
	//case 0x1e: // asl l, X
	//case 0xdd: // cmp l, X
	//case 0xde: // dec l, X
	//case 0x5d: // eor l, X
	//case 0xfe: // inc l, X
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
	//case 0x19: // ora l, Y
	//case 0xf9: // sbc l, Y
	//case 0x99: // sta l, Y
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s <label>, %s lacks Compile() implementation", i.OpName, i.RegisterName))
	}
}

func (i *DirectIndexedInstruction) Compile(c *Compilation) {
	switch i.Payload[0] {
	//case 0x79: // adc abs y
	//case 0x39: // and abs y
	//case 0xd9: // cmp abs y
	//case 0x59: // eor abs y
	case 0xb9: // lda abs y
		c.absoluteIndexedLoad(c.rA, i.Value, c.rY, i.Offset+i.GetSize())
	case 0xbe: // ldx abs y
		c.absoluteIndexedLoad(c.rX, i.Value, c.rY, i.Offset+i.GetSize())
	case 0xbd: // lda abs x
		c.absoluteIndexedLoad(c.rA, i.Value, c.rX, i.Offset+i.GetSize())
	case 0xbc: // ldy abs x
		c.absoluteIndexedLoad(c.rY, i.Value, c.rX, i.Offset+i.GetSize())
	//case 0x19: // ora abs y
	//case 0xf9: // sbc abs y
	case 0x99: // sta abs y
		c.absoluteIndexedStore(c.rA, i.Value, c.rY, i.Offset+i.GetSize())
	case 0x9d: // sta abs x
		c.absoluteIndexedStore(c.rA, i.Value, c.rX, i.Offset+i.GetSize())
	//case 0xb6: // ldx zpg y
	//case 0x96: // stx zpg y

	//case 0x7d: // adc abs x
	//case 0x3d: // and abs x
	//case 0x1e: // asl abs x
	//case 0xdd: // cmp abs x
	case 0xde: // dec abs x
		oldValue := c.dynLoadIndexed(i.Value, c.rX)
		newValue := c.incrementVal(oldValue, -1)
		c.dynStoreIndexed(i.Value, c.rX, newValue)
		c.cycle(7, i.Offset+i.GetSize())
	//case 0x5d: // eor abs x
	case 0xfe: // inc abs x
		oldValue := c.dynLoadIndexed(i.Value, c.rX)
		newValue := c.incrementVal(oldValue, 1)
		c.dynStoreIndexed(i.Value, c.rX, newValue)
		c.cycle(7, i.Offset+i.GetSize())
	//case 0x5e: // lsr abs x
	//case 0x1d: // ora abs x
	//case 0x3e: // rol abs x
	//case 0x7e: // ror abs x
	//case 0xfd: // sbc abs x

	//case 0x75: // adc zpg x
	//case 0x35: // and zpg x
	//case 0x16: // asl zpg x
	//case 0xd5: // cmp zpg x
	//case 0xd6: // dec zpg x
	//case 0x55: // eor zpg x
	//case 0xf6: // inc zpg x
	//case 0xb5: // lda zpg x
	//case 0xb4: // ldy zpg x
	//case 0x56: // lsr zpg x
	//case 0x15: // ora zpg x
	//case 0x36: // rol zpg x
	//case 0x76: // ror zpg x
	//case 0xf5: // sbc zpg x
	//case 0x95: // sta zpg x
	//case 0x94: // sty zpg x
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
		c.cycle(3, c.program.Labels[i.LabelName])
		destBlock := c.labeledBlocks[i.LabelName]
		c.builder.CreateBr(destBlock)
		c.currentBlock = nil
	case 0x20: // jsr
		pc := llvm.ConstInt(llvm.Int16Type(), uint64(i.Offset+2), false)
		c.pushWordToStack(pc)
		c.cycle(6, c.program.Labels[i.LabelName])
		id := c.labelAsEntryPoint(i.LabelName)
		c.builder.CreateCall(c.mainFn, []llvm.Value{llvm.ConstInt(llvm.Int32Type(), uint64(id), false)}, "")
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
			c.cycle(3, i.Offset+i.GetSize())
		} else {
			c.cycle(4, i.Offset+i.GetSize())
		}
	case 0xa4, 0xac: // ldy (zpg, abs)
		v := c.load(i.Value)
		c.builder.CreateStore(v, c.rY)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		if i.Payload[0] == 0xa4 {
			c.cycle(3, i.Offset+i.GetSize())
		} else {
			c.cycle(4, i.Offset+i.GetSize())
		}
	case 0xa6, 0xae: // ldx (zpg, abs)
		v := c.load(i.Value)
		c.builder.CreateStore(v, c.rX)
		c.dynTestAndSetZero(v)
		c.dynTestAndSetNeg(v)
		if i.Payload[0] == 0xa6 {
			c.cycle(3, i.Offset+i.GetSize())
		} else {
			c.cycle(4, i.Offset+i.GetSize())
		}
	case 0xc6, 0xce: // dec (zpg, abs)
		c.incrementMem(i.Value, -1)
		if i.Payload[0] == 0xc6 {
			c.cycle(5, i.Offset+i.GetSize())
		} else {
			c.cycle(6, i.Offset+i.GetSize())
		}
	case 0xe6, 0xee: // inc (zpg, abs)
		c.incrementMem(i.Value, 1)
		if i.Payload[0] == 0xe6 {
			c.cycle(5, i.Offset+i.GetSize())
		} else {
			c.cycle(6, i.Offset+i.GetSize())
		}
	case 0x46, 0x4e: // lsr (zpg, abs)
		oldValue := c.load(i.Value)
		c1 := llvm.ConstInt(llvm.Int8Type(), 1, false)
		newValue := c.builder.CreateLShr(oldValue, c1, "")
		c.store(i.Value, newValue)
		c.dynTestAndSetZero(newValue)
		c.dynTestAndSetCarryLShr(oldValue)
		if i.Payload[0] == 0x46 {
			c.cycle(5, i.Offset+i.GetSize())
		} else {
			c.cycle(6, i.Offset+i.GetSize())
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
	//case 0x4c: // jmp abs
	//case 0x20: // jsr abs
	//case 0x0d: // ora abs
	//case 0x2e: // rol abs
	//case 0x6e: // ror abs
	//case 0xed: // sbc abs
	case 0x85, 0x8d: // sta (zpg, abs)
		c.store(i.Value, c.builder.CreateLoad(c.rA, ""))
		if i.Payload[0] == 0x85 {
			c.cycle(3, i.Offset+i.GetSize())
		} else {
			c.cycle(4, i.Offset+i.GetSize())
		}
	case 0x86, 0x8e: // stx (zpg, abs)
		c.store(i.Value, c.builder.CreateLoad(c.rX, ""))
		if i.Payload[0] == 0x86 {
			c.cycle(3, i.Offset+i.GetSize())
		} else {
			c.cycle(4, i.Offset+i.GetSize())
		}
	case 0x84, 0x8c: // sty (zpg, abs)
		c.store(i.Value, c.builder.CreateLoad(c.rY, ""))
		if i.Payload[0] == 0x84 {
			c.cycle(3, i.Offset+i.GetSize())
		} else {
			c.cycle(4, i.Offset+i.GetSize())
		}
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s lacks Compile() implementation", i.Render()))
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
		baseAddr := c.loadWord(i.Value)
		rY := c.builder.CreateLoad(c.rY, "")
		rYw := c.builder.CreateZExt(rY, llvm.Int16Type(), "")
		addr := c.builder.CreateAdd(baseAddr, rYw, "")
		rA := c.builder.CreateLoad(c.rA, "")
		c.dynStore(addr, 0, 0xffff, rA)
		c.cycle(6, i.Offset+i.GetSize())
	default:
		c.Errors = append(c.Errors, fmt.Sprintf("%s ($%02x), Y lacks Compile() implementation", i.OpName, i.Value))
	}
}

func (i *IndirectInstruction) Compile(c *Compilation) {
	c.debugPrint(i.Render())
	c.Errors = append(c.Errors, "IndirectInstruction lacks Compile() implementation")
}

