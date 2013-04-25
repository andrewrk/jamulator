package asm6502

import (
	"strings"
	"errors"
	"fmt"
	"os"
	"bufio"
	"io"
	"encoding/binary"
)

// Program is a proper program, one that you can compile
// into native code. A ProgramAST can be compiled into a
// Program and 6502 machine code can be read directly into
// a Program
type Program struct {
	Ast *ProgramAST
	Variables map[string] int
	Labels map[string] uint16
	Errors []error

	offset int
}

type Measurer interface {
	Measure(p *Program) error
	GetSize() int
}

type Assembler interface {
	Assemble(bin *machineCode) error
}

// Maintains the state for assembling a Program into
// machine code
type machineCode struct {
	prog *Program
	writer *bufio.Writer
	Errors []string
}

func (m machineCode) Error() string {
	return strings.Join(m.Errors, "\n")
}

func (bin *machineCode) getLabel(name string, offset uint16) (uint16, bool) {
	if name == "." {
		return offset, true
	}
	value, ok := bin.prog.Labels[name]
	return value, ok
}

var impliedOpCode = map[string] byte {
	"asl": 0x0a,
	"brk": 0x00,
	"clc": 0x18,
	"cld": 0xd8,
	"cli": 0x58,
	"clv": 0xb8,
	"dex": 0xca,
	"dey": 0x88,
	"inx": 0xe8,
	"iny": 0xc8,
	"lsr": 0x4a,
	"nop": 0xea,
	"pha": 0x48,
	"php": 0x08,
	"pla": 0x68,
	"plp": 0x28,
	"rol": 0x2a,
	"ror": 0x6a,
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

var immediateOpCode = map[string] byte {
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

var zeroPageXOpcode = map[string] byte {
	"adc": 0x75,
	"and": 0x35,
	"asl": 0x16,
	"cmp": 0xd5,
	"dec": 0xd6,
	"eor": 0x55,
	"inc": 0xf6,
	"lda": 0xb5,
	"ldy": 0xb4,
	"lsr": 0x56,
	"ora": 0x15,
	"rol": 0x36,
	"ror": 0x76,
	"sbc": 0xf5,
	"sta": 0x95,
	"sty": 0x94,
}

var absIndexedXOpCode = map[string] byte {
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

var zeroPageYOpCode = map[string] byte {
	"ldx": 0xb6,
	"stx": 0x96,
}

var zeroPageOpCode = map[string] byte {
	"adc": 0x65,
	"and": 0x25,
	"asl": 0x06,
	"bit": 0x24,
	"cmp": 0xc5,
	"cpx": 0xe4,
	"cpy": 0xc4,
	"dec": 0xc6,
	"eor": 0x45,
	"inc": 0xe6,
	"lda": 0xa5,
	"ldx": 0xa6,
	"ldy": 0xa4,
	"lsr": 0x46,
	"ora": 0x05,
	"rol": 0x26,
	"ror": 0x66,
	"sbc": 0xe5,
	"sta": 0x85,
	"stx": 0x86,
	"sty": 0x84,
}

var absIndexedYOpCode = map[string] byte {
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

var absOpCode = map[string] byte {
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

var relOpCode = map[string] byte {
	"bcc": 0x90,
	"bcs": 0xb0,
	"beq": 0xf0,
	"bmi": 0x30,
	"bne": 0xd0,
	"bpl": 0x10,
	"bvc": 0x50,
	"bvs": 0x70,
}

var indirectXOpCode = map[string] byte {
	"adc": 0x61,
	"and": 0x21,
	"cmp": 0xc1,
	"eor": 0x41,
	"lda": 0xa1,
	"ora": 0x01,
	"sbc": 0xe1,
	"sta": 0x81,
}

var indirectYOpCode = map[string] byte {
	"adc": 0x71,
	"and": 0x31,
	"cmp": 0xd1,
	"eor": 0x51,
	"lda": 0xb1,
	"ora": 0x11,
	"sbc": 0xf1,
	"sta": 0x91,
}

type opcodeDef struct {
	opcode int
	size int
}

func (ii *ImmediateInstruction) Measure(p *Program) error {
	lowerOpName := strings.ToLower(ii.OpName)
	opcode, ok := immediateOpCode[lowerOpName]
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized immediate instruction: %s", ii.Line, ii.OpName))
	}
	if ii.Value > 0xff {
		return errors.New(fmt.Sprintf("Line %d: Immediate instruction argument must be a 1 byte integer.", ii.Line))
	}
	ii.OpCode = opcode
	ii.Size = 2
	return nil
}

func (n *ImmediateInstruction) Assemble(bin *machineCode) error {
	err := bin.writer.WriteByte(byte(n.OpCode))
	if err != nil { return err }
	return bin.writer.WriteByte(byte(n.Value))
}

func (ii *ImpliedInstruction) Measure(p *Program) error {
	lowerOpName := strings.ToLower(ii.OpName)
	opcode, ok := impliedOpCode[lowerOpName]
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized implied instruction: %s", ii.Line, ii.OpName))
	}
	ii.OpCode = opcode
	ii.Size = 1
	return nil
}

func (n *ImpliedInstruction) Assemble(bin *machineCode) error {
	return bin.writer.WriteByte(byte(n.OpCode))
}

func (n *DirectIndexedInstruction) Measure(p *Program) error {
	lowerOpName := strings.ToLower(n.OpName)
	lowerRegName := strings.ToLower(n.RegisterName)
	if lowerRegName == "x" {
		if n.Value <= 0xff {
			opcode, ok := zeroPageXOpcode[lowerOpName]
			if ok {
				n.Payload = []byte{opcode, byte(n.Value)}
				return nil
			}
		} else if n.Value > 0xffff {
			return errors.New(fmt.Sprintf("Line %d: Absolute memory address is limited to 2 bytes.", n.Line))
		}
		opcode, ok := absIndexedXOpCode[lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized absolute, X instruction: %s", n.Line, n.OpName))
		}
		n.Payload = []byte{opcode, 0, 0}
		binary.LittleEndian.PutUint16(n.Payload[1:], uint16(n.Value))
		return nil
	} else if lowerRegName == "y" {
		if n.Value <= 0xff {
			opcode, ok := zeroPageYOpCode[lowerOpName]
			if ok {
				n.Payload = []byte{opcode, byte(n.Value)}
				return nil
			}
		} else if n.Value > 0xffff {
			return errors.New(fmt.Sprintf("Line %d: Absolute memory address is limited to 2 bytes.", n.Line))
		}
		opcode, ok := absIndexedYOpCode[lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized absolute, Y instruction: %s", n.Line, n.OpName))
		}
		n.Payload = []byte{opcode, 0, 0}
		binary.LittleEndian.PutUint16(n.Payload[1:], uint16(n.Value))
		return nil
	}
	return errors.New(fmt.Sprintf("Line %d: Register argument must be X or Y", n.Line))
}

func (n *DirectIndexedInstruction) Assemble(bin *machineCode) error {
	_, err := bin.writer.Write(n.Payload)
	return err
}


func (n *DirectWithLabelIndexedInstruction) Measure(p *Program) error {
	lowerOpName := strings.ToLower(n.OpName)
	lowerRegName := strings.ToLower(n.RegisterName)
	if lowerRegName == "x" {
		opcode, ok := absIndexedXOpCode[lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized direct, X instruction: %s", n.Line, n.OpName))
		}
		n.OpCode = opcode
		n.Size = 3
		return nil
	} else if lowerRegName == "y" {
		opcode, ok := absIndexedYOpCode[lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized direct, Y instruction: %s", n.Line, n.OpName))
		}
		n.OpCode = opcode
		n.Size = 3
		return nil
	}
	return errors.New(fmt.Sprintf("Line %d: Register argument must be X or Y", n.Line))
}


func (n *DirectWithLabelIndexedInstruction) Assemble(bin *machineCode) error {
	err := bin.writer.WriteByte(n.OpCode)
	if err != nil { return err }
	labelValue, ok := bin.getLabel(n.LabelName, n.Offset + uint16(n.Size))
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Undefined label: %s", n.Line, n.LabelName))
	}
	buf := []byte{0, 0}
	binary.LittleEndian.PutUint16(buf, labelValue)
	_, err = bin.writer.Write(buf)
	return err
}


func (n *DirectWithLabelInstruction) Measure(p *Program) error {
	if p.offset >= 0xffff {
		return errors.New(fmt.Sprintf("Line %d: Instruction is at offset %#x which is greater than 2 bytes.", n.Line, p.offset))
	}
	n.Offset = uint16(p.offset)
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

func (n *DirectWithLabelInstruction) Assemble(bin *machineCode) error {
	err := bin.writer.WriteByte(n.OpCode)
	if err != nil { return err }
	labelValue, ok := bin.getLabel(n.LabelName, n.Offset + uint16(n.Size))
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Undefined label: %s", n.Line, n.LabelName))
	}
	if n.Size == 2 {
		// relative address
		delta := int(n.Offset) - int(labelValue)
		if delta > 127 || delta < -128 {
			return errors.New(fmt.Sprintf("Line %d: Label address must be within 127 bytes of instruction address.", n.Line))
		}
		return bin.writer.WriteByte(byte(delta))
	}
	// absolute address
	buf := []byte{0, 0}
	binary.LittleEndian.PutUint16(buf, labelValue)
	_, err = bin.writer.Write(buf)
	return err
}

func (n *DirectInstruction) Measure(p *Program) error {
	lowerOpName := strings.ToLower(n.OpName)

	// try indirect
	opcode, ok := relOpCode[lowerOpName]
	if ok {
		if n.Value > 0xff {
			return errors.New(fmt.Sprintf("Line %d: Relative memory address is limited to 1 byte.", n.Line))
		}
		n.Payload = []byte{opcode, byte(n.Value)}
		return nil
	}

	// try zero page
	if n.Value <= 0xff {
		opcode, ok := zeroPageOpCode[lowerOpName]
		if ok {
			n.Payload = []byte{opcode, byte(n.Value)}
			return nil
		}
	}

	// must be absolute
	opcode, ok = absOpCode[lowerOpName]
	if ok {
		if n.Value > 0xffff {
			return errors.New(fmt.Sprintf("Line %d: Absolute memory address is limited to 2 bytes.", n.Line))
		}
		n.Payload = []byte{opcode, 0, 0}
		binary.LittleEndian.PutUint16(n.Payload[1:], uint16(n.Value))
		return nil
	}

	return errors.New(fmt.Sprintf("Line %d: Unrecognized direct instruction: %s", n.Line, n.OpName))
}

func (n *DirectInstruction) Assemble(bin *machineCode) error {
	_, err := bin.writer.Write(n.Payload)
	return err
}

func (n *IndirectXInstruction) Measure(p *Program) error {
	lowerOpName := strings.ToLower(n.OpName)
	opcode, ok := indirectXOpCode[lowerOpName]
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized indirect x indexed instruction: %s", n.Line, n.OpName))
	}
	if n.Value > 0xff {
		return errors.New(fmt.Sprintf("Line %d: Indirect X memory address is limited to 1 byte.", n.Line))
	}
	n.Payload = []byte{opcode, byte(n.Value)}
	return nil
}

func (n *IndirectXInstruction) Assemble(bin *machineCode) error {
	_, err := bin.writer.Write(n.Payload)
	return err
}


func (n *IndirectYInstruction) Measure(p *Program) error {
	lowerOpName := strings.ToLower(n.OpName)
	opcode, ok := indirectYOpCode[lowerOpName]
	if !ok {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized indirect y indexed instruction: %s", n.Line, n.OpName))
	}
	if n.Value > 0xff {
		return errors.New(fmt.Sprintf("Line %d: Indirect Y memory address is limited to 1 byte.", n.Line))
	}
	n.Payload = []byte{opcode, byte(n.Value)}
	return nil
}

func (n *IndirectYInstruction) Assemble(bin *machineCode) error {
	_, err := bin.writer.Write(n.Payload)
	return err
}

func (n *DataStatement) Measure(p *Program) error {
	n.Size = 0
	n.Offset = uint16(p.offset)
	for _, dataItem := range(n.dataList) {
		switch t := dataItem.(type) {
		case *StringDataItem: n.Size += len(*t)
		case *IntegerDataItem:
			if *t > 0xff {
				return errors.New(fmt.Sprintf("Line %d: Integer data item limited to 1 byte.", n.Line))
			}
			n.Size += 1
		case *LabelCall: n.Size += 2
		default: panic("unknown data item type")
		}
	}
	return nil
}

func (n *DataStatement) Assemble(bin *machineCode) error {
	for _, dataItem := range(n.dataList) {
		switch t := dataItem.(type) {
			case *StringDataItem:
				_, err := bin.writer.WriteString(string(*t))
				if err != nil { return err }
			case *IntegerDataItem:
				err := bin.writer.WriteByte(byte(*t))
				if err != nil { return err }
			case *LabelCall:
				labelValue, ok := bin.getLabel(t.LabelName, n.Offset + uint16(n.Size))
				if !ok {
					return errors.New(fmt.Sprintf("Line %d: Undefined label: %s", n.Line, t.LabelName))
				}
				int16buf := []byte{0, 0}
				binary.LittleEndian.PutUint16(int16buf, uint16(labelValue))
				_, err := bin.writer.Write(int16buf)
				if err != nil { return err }
			default: panic("unknown data item type")
		}
	}
	return nil
}

func (n *IndirectInstruction) Measure(p *Program) error {
	lowerOpName := strings.ToLower(n.OpName)
	if lowerOpName != "jmp" {
		return errors.New(fmt.Sprintf("Line %d: Unrecognized indirect instruction: %s", n.Line, n.OpName))
	}
	n.Payload = []byte{0x6c, 0, 0}
	if n.Value > 0xffff {
		return errors.New(fmt.Sprintf("Line %d: Memory address is limited to 2 bytes.", n.Line))
	}
	binary.LittleEndian.PutUint16(n.Payload[1:], uint16(n.Value))
	return nil
}

func (n *IndirectInstruction) Assemble(bin *machineCode) error {
	_, err := bin.writer.Write(n.Payload)
	return err
}

// collect all variable assignments into a map
func (p *Program) Visit(n Node) {
	switch ss := n.(type) {
	case *AssignStatement:
		p.Variables[ss.VarName] = ss.Value
	case *OrgPseudoOp:
		p.offset = ss.Value
	case Measurer:
		err := ss.Measure(p)
		if err != nil {
			p.Errors = append(p.Errors, err)
		}
		p.offset += ss.GetSize()
	case *LabeledStatement:
		if p.offset >= 0xffff {
			err := errors.New(fmt.Sprintf("Line %d: Label memory address must fit in 2 bytes.", ss.Line))
			p.Errors = append(p.Errors, err)
		} else if ss.LabelName == "." {
			err := errors.New(fmt.Sprintf("Line %d: Reserved label name: '.'", ss.Line))
			p.Errors = append(p.Errors, err)
		} else {
			_, exists := p.Labels[ss.LabelName]
			if exists {
				err := errors.New(fmt.Sprintf("Line %d: Label %s already defined.", ss.Line, ss.LabelName))
				p.Errors = append(p.Errors, err)
			} else {
				p.Labels[ss.LabelName] = uint16(p.offset)
			}
		}
	}
}

func (p *Program) VisitEnd(n Node) {}

func (bin *machineCode) Visit(n Node) {
	switch nn := n.(type) {
	case Assembler:
		err := nn.Assemble(bin)
		if err != nil {
			bin.Errors = append(bin.Errors, err.Error())
		}
	}
}

func (bin *machineCode) VisitEnd(n Node) {}

func (p *Program) Assemble(w io.Writer) error {
	writer := bufio.NewWriter(w)
	bin := machineCode{
		p,
		writer,
		[]string{},
	}
	p.Ast.Ast(&bin)
	writer.Flush()

	if len(bin.Errors) > 0 {
		return bin
	}

	return nil
}

func (p *Program) AssembleToFile(filename string) error {
	fd, err := os.Create(filename)
	if err != nil { return err}

	err = p.Assemble(fd)
	err2 := fd.Close()
	if err != nil { return err }
	if err2 != nil { return err2 }

	return nil
}

func (ast *ProgramAST) ToProgram() (*Program) {
	p := Program{
		ast,
		map[string]int {},
		map[string]uint16 {},
		[]error{},
		0,
	}
	ast.Ast(&p)
	return &p
}

func (n OrgPseudoOp) Assemble(bin *machineCode) error {
	if n.Value != 0 {
		return errors.New(fmt.Sprintf("Line %d: Only org 0 is currently supported.", n.Line))
	}
	return nil
}
