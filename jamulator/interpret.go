package jamulator

import (
	"github.com/axw/gollvm/llvm"
	"fmt"
)

var interpretOps = [256]func(*Compilation) {
	// 0x00
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	func (c *Compilation) {
		// 0x0a asl implied
		c.debugPrintf("asl\n", []llvm.Value{})
		a := c.builder.CreateLoad(c.rA, "")
		c.builder.CreateStore(c.performAsl(a), c.rA)
		c.cycle(2, -1)
	},
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0x10
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0x20
	func (c *Compilation) {
		// 0x20 jsr abs
		newPc := c.interpAbsAddr()
		c.debugPrintf("jsr $%04x\n", []llvm.Value{newPc})

		pc := c.builder.CreateLoad(c.rPC, "")
		pcMinusOne := c.builder.CreateSub(pc, llvm.ConstInt(pc.Type(), 1, false), "")

		c.debugPrintf("jsr: saving $%04x\n", []llvm.Value{pcMinusOne})

		c.pushWordToStack(pcMinusOne)
		c.builder.CreateStore(newPc, c.rPC)
		c.cycle(6, -1)
	},
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	func (c *Compilation) {
		// 0x2c bit abs
		addr := c.interpAbsAddr()
		c.debugPrintf("bit $%04x\n", []llvm.Value{addr})
		c.performBit(c.dynLoad(addr, 0, 0xffff))
		c.cycle(4, -1)
	},
	nil,
	nil,
	nil,
	// 0x30
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0x40
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	func (c *Compilation) {
		// 0x48 pha implied
		c.debugPrintf("pha\n", []llvm.Value{})
		a := c.builder.CreateLoad(c.rA, "")
		c.pushToStack(a)
		c.cycle(3, -1)
	},
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0x50
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0x60
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0x70
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0x80
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	func (c *Compilation) {
		// 0x8d sta abs
		v := c.interpAbsAddr()
		c.debugPrintf("sta $%04x\n", []llvm.Value{v})
		c.dynStore(v, 0, 0xffff, c.builder.CreateLoad(c.rA, ""))
		c.cycle(4, -1)
	},
	nil,
	nil,
	// 0x90
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0xa0
	nil,
	nil,
	func (c *Compilation) {
		// 0xa2 ldx immediate
		v := c.interpImmedAddr()
		c.debugPrintf("ldx #$%02x\n", []llvm.Value{v})
		c.performLdx(v)
		c.cycle(2, -1)
	},
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	func (c *Compilation) {
		// 0xa9 lda immediate
		v := c.interpImmedAddr()
		c.debugPrintf("lda #$%02x\n", []llvm.Value{v})
		c.performLda(v)
		c.cycle(2, -1)
	},
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0xb0
	nil,
	nil,
	nil,
	nil,
	func (c *Compilation) {
		// 0xb4 ldy zpg x
		addr := c.interpZpgIndexAddr(c.rX)
		c.debugPrintf("ldy $%02x, X\n", []llvm.Value{addr})
		v := c.dynLoad(addr, 0, 0xff)
		c.performLdy(v)
		c.cycle(4, -1)
	},
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0xc0
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0xd0
	func (c *Compilation) {
		// 0xd0 bne relative
		// TODO: optimize by not loading destAddr if we're not in debug mode
		destAddr := c.interpRelAddr()
		c.debugPrintf("bne $%04x\n", []llvm.Value{destAddr})

		pc := c.builder.CreateLoad(c.rPC, "")
		c1 := llvm.ConstInt(pc.Type(), 1, false)
		xff00 := llvm.ConstInt(llvm.Int16Type(), uint64(0xff00), false)

		doneBlock := c.createBlock("done")
		isZero := c.builder.CreateLoad(c.rSZero, "")
		notZero := c.builder.CreateNot(isZero, "")
		// if ! zero
		notBranchingBlock := c.createIf(notZero)
		c.builder.CreateStore(destAddr, c.rPC)
		// if instrAddr&0xff00 == addr&0xff00 {
		instrAddr := c.builder.CreateSub(pc, c1, "")
		maskedInstrAddr := c.builder.CreateAnd(instrAddr, xff00, "")
		maskedDestAddr := c.builder.CreateAnd(destAddr, xff00, "")
		eq := c.builder.CreateICmp(llvm.IntEQ, maskedInstrAddr, maskedDestAddr, "")
		// if same page page
		crossedPageBlock := c.createIf(eq)
		c.cycle(3, -1)
		c.builder.CreateBr(doneBlock)
		// else if crossed page
		c.selectBlock(crossedPageBlock)
		c.cycle(4, -1)
		c.builder.CreateBr(doneBlock)
		// else if not branching
		c.selectBlock(notBranchingBlock)
		// pc++
		newPc := c.builder.CreateAdd(pc, c1, "")
		c.builder.CreateStore(newPc, c.rPC)
		c.cycle(2, -1)
		c.builder.CreateBr(doneBlock)
		// done
		c.selectBlock(doneBlock)
	},
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0xe0
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	// 0xf0
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
	nil,
}

var interpretOpCount = 0

func init() {
	// count the non-nil ones
	for _, fn := range interpretOps {
		if fn != nil {
			interpretOpCount += 1
		}
	}

}

func (c *Compilation) interpImmedAddr() llvm.Value {
	oldPc := c.builder.CreateLoad(c.rPC, "")
	newPc := c.builder.CreateAdd(oldPc, llvm.ConstInt(oldPc.Type(), 1, false), "")
	c.builder.CreateStore(newPc, c.rPC)
	return c.dynLoad(oldPc, 0, 0xffff)
}

func (c *Compilation) interpAbsAddr() llvm.Value {
	oldPc := c.builder.CreateLoad(c.rPC, "")
	addr := c.dynLoadWord(oldPc)
	newPc := c.builder.CreateAdd(oldPc, llvm.ConstInt(oldPc.Type(), 2, false), "")
	c.builder.CreateStore(newPc, c.rPC)
	return addr
}

func (c *Compilation) interpRelAddr() llvm.Value {
	pc := c.builder.CreateLoad(c.rPC, "")
	offset8 := c.dynLoad(pc, 0, 0xffff)
	offset16 := c.builder.CreateSExt(offset8, llvm.Int16Type(), "")
	addr := c.builder.CreateAdd(pc, offset16, "")
	c1 := llvm.ConstInt(llvm.Int16Type(), 1, false)
	return c.builder.CreateAdd(addr, c1, "")
}

func (c *Compilation) interpZpgIndexAddr(indexPtr llvm.Value) llvm.Value {
	pc := c.builder.CreateLoad(c.rPC, "")
	base := c.dynLoad(pc, 0, 0xffff)
	index := c.builder.CreateLoad(indexPtr, "")
	return c.builder.CreateAdd(base, index, "")
}

func (c *Compilation) addInterpretBlock() {
	// here we create a basic block that we jump to when we need to
	// fall back on interpreting
	c.selectBlock(c.interpretBlock)
	// load the pc
	pc := c.builder.CreateLoad(c.rPC, "")
	// get the opcode at pc
	opCode := c.dynLoad(pc, 0, 0xffff)
	// increment pc
	pc = c.builder.CreateAdd(pc, llvm.ConstInt(pc.Type(), 1, false), "")
	c.builder.CreateStore(pc, c.rPC)
	// switch on the opcode
	badOpCodeBlock := c.createBlock("BadOpCode")
	sw := c.builder.CreateSwitch(opCode, badOpCodeBlock, interpretOpCount)
	c.selectBlock(badOpCodeBlock)
	c.createPanic("invalid op code: $%02x\n", []llvm.Value{opCode})

	i8Type := llvm.Int8Type()
	for op, fn := range interpretOps {
		if fn == nil {
			continue
		}
		bb := c.createBlock(fmt.Sprintf("x%02x", op))
		sw.AddCase(llvm.ConstInt(i8Type, uint64(op), false), bb)
		c.selectBlock(bb)
		fn(c)
		// jump back to dynJumpBlock. maybe we're back in
		// statically compiled happy land.
		c.builder.CreateBr(c.dynJumpBlock)
	}
}
