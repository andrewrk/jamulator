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
	nil,
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
	nil,
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
