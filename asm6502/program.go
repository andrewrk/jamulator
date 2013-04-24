package asm6502

import (
	"strings"
	"errors"
	"fmt"
)

// Program is a proper program, one that you can compile
// into native code. A ProgramAST can be compiled into a
// Program and 6502 machine code can be read directly into
// a Program
type Program struct {
	Variables map[string] int
	Labels map[string] int
	Errors []error
}

var impliedOpCode = map[string] int {
	"brk": 0x00,
	"clc": 0x18,
	"cld": 0xd8,
	"cli": 0x58,
	"clv": 0xb8,
	"dex": 0xca,
	"dey": 0x88,
	"inx": 0xe8,
	"iny": 0xc8,
	"nop": 0xea,
	"pha": 0x48,
	"php": 0x08,
	"pla": 0x68,
	"plp": 0x28,
	"rti": 0x40,
	"rts": 0x60,
	"sec": 0x38,
	"sed": 0xf8,
	"sei": 0x78,
	"tax": 0xaa,
	"tay": 0xa8,
	"tsx": 0xba,
	"txa": 0x8a,
	"txs": 0x9a,
	"tya": 0x98,
}

var immediateOpCode = map[string] int {
	"adc": 0x69,
	"and": 0x29,
	"cmp": 0xc9,
	"cpx": 0xe0,
	"cpy": 0xc0,
	"eor": 0x49,
	"lda": 0xa9,
	"ldx": 0xa2,
	"ldy": 0xa0,
	"ora": 0x09,
	"sbc": 0xe9,
}

var absIndexedXOpCode = map[string] int {
	"adc": 0x7d,
	"and": 0x3d,
	"asl": 0x1e,
	"cmp": 0xdd,
	"dec": 0xde,
	"eor": 0x5d,
	"inc": 0xfe,
	"lda": 0xbd,
	"ldy": 0xbc,
	"lsr": 0x5e,
	"ora": 0x1d,
	"rol": 0x3e,
	"ror": 0x7e,
	"sbc": 0xfd,
	"sta": 0x9d,
}

var absIndexedYOpCode = map[string] int {
	"adc": 0x79,
	"and": 0x39,
	"cmp": 0xd9,
	"eor": 0x59,
	"lda": 0xb9,
	"ldx": 0xbe,
	"ora": 0x19,
	"sbc": 0xf9,
	"sta": 0x99,
}

var absOpCode = map[string] int {
	"adc": 0x6d,
	"and": 0x2d,
	"asl": 0x0e,
	"bit": 0x2c,
	"cmp": 0xcd,
	"cpx": 0xec,
	"cpy": 0xcc,
	"dec": 0xce,
	"eor": 0x4d,
	"inc": 0xee,
	"jmp": 0x4c,
	"jsr": 0x20,
	"lda": 0xad,
	"ldx": 0xae,
	"ldy": 0xac,
	"lsr": 0x4e,
	"ora": 0x0d,
	"rol": 0x2e,
	"ror": 0x6e,
	"sbc": 0xed,
	"sta": 0x8d,
	"stx": 0x8e,
	"sty": 0x8c,
}

var relOpCode = map[string] int {
	"bcc": 0x90,
	"bcs": 0xb0,
	"beq": 0xf0,
	"bmi": 0x30,
	"bne": 0xd0,
	"bpl": 0x10,
	"bvc": 0x50,
	"bvs": 0x70,
}

type opcodeDef struct {
	opcode int
	size int
}

func (ii ImmediateInstruction) Measure() error {
	lowerOpName := strings.ToLower(ii.OpName)
	opcode, ok := immediateOpCode[lowerOpName]
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized immediate instruction: %s", ii.Line, ii.OpName))
	}
	ii.OpCode = opcode
	ii.Size = 2
	return nil
}

func (ii ImpliedInstruction) Measure() error {
	lowerOpName := strings.ToLower(ii.OpName)
	opcode, ok := impliedOpCode[lowerOpName]
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized implied instruction: %s", ii.Line, ii.OpName))
	}
	ii.OpCode = opcode
	ii.Size = 1
	return nil
}

func (n AbsoluteWithLabelIndexedInstruction) Measure() error {
	lowerOpName := strings.ToLower(n.OpName)
	lowerRegName := strings.ToLower(n.RegisterName)
	if lowerRegName == "x" {
		opcode, ok := absIndexedXOpCode[lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized absolute, X instruction: %s", n.Line, n.OpName))
		}
		n.OpCode = opcode
		n.Size = 3
		return nil
	} else if lowerRegName == "y" {
		opcode, ok := absIndexedYOpCode[lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized absolute, Y instruction: %s", n.Line, n.OpName))
		}
		n.OpCode = opcode
		n.Size = 3
		return nil
	}
	return errors.New(fmt.Sprintf("Line %d: Register argument must be X or Y", n.Line))
}

func (n DirectWithLabelInstruction) Measure() error {
	lowerOpName := strings.ToLower(n.OpName)
	opcode, ok := absOpCode[lowerOpName]
	if ok {
		n.OpCode = opcode
		n.Size = 3
		return nil
	}
	opcode, ok = relOpCode[lowerOpName]
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized direct instruction: %s", n.Line, n.OpName))
	}
	n.OpCode = opcode
	n.Size = 2
	return nil
}

func (n DirectInstruction) Measure() error {
	lowerOpName := strings.ToLower(n.OpName)
	opcode, ok := absOpCode[lowerOpName]
	if ok {
		n.OpCode = opcode
		n.Size = 3
		return nil
	}
	opcode, ok = relOpCode[lowerOpName]
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized direct instruction: %s", n.Line, n.OpName))
	}
	n.OpCode = opcode
	n.Size = 2
	return nil
}

func (n DataStatement) Measure() error {
	n.Size = 0
	for _, dataItem := range(n.dataList) {
		switch t := dataItem.(type) {
		case StringDataItem: n.Size += len(t)
		case IntegerDataItem: n.Size += 1
		default: panic("unknown data item type")
		}
	}
	return nil
}


// collect all variable assignments into a map
func (p *Program) Visit(n Node) {
	switch ss := n.(type) {
	case AssignStatement:
		p.Variables[ss.VarName] = ss.Value
	case Measurer:
		err := ss.Measure()
		if err != nil {
			p.Errors = append(p.Errors, err)
		}
	}
}

func (p *Program) VisitEnd(n Node) {}

func NewProgram() *Program {
	p := Program{
		map[string]int {},
		map[string]int {},
		[]error{},
	}
	return &p
}

func (ast *ProgramAST) ToProgram() (*Program, error) {
	p := NewProgram()
	ast.Ast(p)
	return p, nil
}
