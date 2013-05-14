package jamulator

import (
	"bufio"
	"encoding/binary"
	"container/list"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"bytes"
)

type Program struct {
	List      *list.List
	Labels    map[string]int
	Errors    []string
	ChrRom    [][]byte
	PrgRom    [][]byte
	Mirroring Mirroring
	// maps memory offset to element in Ast
	Offsets    map[int]*list.Element
	Variables map[string]int
}

type Assembler interface {
	Resolve() error
	Assemble(symbolGetter) error
	GetPayload() []byte
	GetLine() int
	SetOffset(int)
	GetOffset() int
}

type symbolGetter interface {
	getSymbol(string, int) (int, bool)
}

func (i *Instruction) GetPayload() []byte {
	return i.Payload
}

func (i *Instruction) GetLine() int {
	return i.Line
}

func (i *Instruction) GetOffset() int {
	return i.Offset
}

func (i *Instruction) SetOffset(offset int) {
	i.Offset = offset
}

func (s *DataStatement) GetPayload() []byte {
	return s.Payload
}

func (s *DataStatement) GetLine() int {
	return s.Line
}

func (s *DataStatement) GetOffset() int {
	return s.Offset
}

func (s *DataStatement) SetOffset(offset int) {
	s.Offset = offset
}

func (p *Program) getSymbol(name string, offset int) (int, bool) {
	if name == "." {
		return offset, true
	}
	// look up as variable
	value, ok := p.Variables[name]
	if !ok {
		// look up as label
		value, ok = p.Labels[name]
	}
	return value, ok
}

// computes OpCode, Payload, and Size
func (i *Instruction) Resolve() error {
	var ok bool
	lowerOpName := strings.ToLower(i.OpName)
	switch i.Type {
	default: panic("unexpected instruction type")
	case ImmediateInstruction:
		i.OpCode, ok = opNameToOpCode[immedAddr][lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized immediate instruction: %s", i.Line, i.OpName))
		}
		if i.Value > 0xff {
			return errors.New(fmt.Sprintf("Line %d: Immediate instruction argument must be a 1 byte integer.", i.Line))
		}
		i.Payload = []byte{i.OpCode, byte(i.Value)}
	case ImpliedInstruction:
		i.OpCode, ok = opNameToOpCode[impliedAddr][lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized implied instruction: %s", i.Line, i.OpName))
		}
		i.Payload = []byte{i.OpCode}
	case DirectInstruction:
		// try indirect
		i.OpCode, ok = opNameToOpCode[indirectAddr][lowerOpName]
		if ok {
			if i.Value > 0xff {
				return errors.New(fmt.Sprintf("Line %d: Relative memory address is limited to 1 byte.", i.Line))
			}
			i.Payload = []byte{i.OpCode, byte(i.Value)}
			return nil
		}
		// try zero page
		if i.Value <= 0xff {
			i.OpCode, ok = opNameToOpCode[zeroPageAddr][lowerOpName]
			if ok {
				i.Payload = []byte{i.OpCode, byte(i.Value)}
				return nil
			}
		}
		// must be absolute
		i.OpCode, ok = opNameToOpCode[absAddr][lowerOpName]
		if ok {
			if i.Value > 0xffff {
				return errors.New(fmt.Sprintf("Line %d: Absolute memory address is limited to 2 bytes.", i.Line))
			}
			i.Payload = []byte{i.OpCode, 0, 0}
			binary.LittleEndian.PutUint16(i.Payload[1:], uint16(i.Value))
			return nil
		}
		return errors.New(fmt.Sprintf("Line %d: Unrecognized direct instruction: %s", i.Line, i.OpName))
	case DirectWithLabelInstruction:
		i.OpCode, ok = opNameToOpCode[absAddr][lowerOpName]
		if ok {
			// 0s are placeholder for when we resolve the label
			i.Payload = []byte{i.OpCode, 0, 0}
			return nil
		}
		i.OpCode, ok = opNameToOpCode[relativeAddr][lowerOpName]
		if ok {
			// 0 is placeholder for when we resolve the label
			i.Payload = []byte{i.OpCode, 0}
			return nil
		}
		return errors.New(fmt.Sprintf("Line %d: Unrecognized direct instruction: %s", i.Line, i.OpName))
	case DirectIndexedInstruction:
		lowerRegName := strings.ToLower(i.RegisterName)
		if lowerRegName == "x" {
			if i.Value <= 0xff {
				i.OpCode, ok = opNameToOpCode[zeroXIndexAddr][lowerOpName]
				if ok {
					i.Payload = []byte{i.OpCode, byte(i.Value)}
					return nil
				}
			} else if i.Value > 0xffff {
				return errors.New(fmt.Sprintf("Line %d: Absolute memory address is limited to 2 bytes.", i.Line))
			}
			i.OpCode, ok = opNameToOpCode[absXAddr][lowerOpName]
			if ok {
				i.Payload = []byte{i.OpCode, 0, 0}
				binary.LittleEndian.PutUint16(i.Payload[1:], uint16(i.Value))
				return nil
			}
			return errors.New(fmt.Sprintf("Line %d: Unrecognized absolute, X instruction: %s", i.Line, i.OpName))
		} else if lowerRegName == "y" {
			if i.Value <= 0xff {
				i.OpCode, ok = opNameToOpCode[zeroYIndexAddr][lowerOpName]
				if ok {
					i.Payload = []byte{i.OpCode, byte(i.Value)}
					return nil
				}
			} else if i.Value > 0xffff {
				return errors.New(fmt.Sprintf("Line %d: Absolute memory address is limited to 2 bytes.", i.Line))
			}
			i.OpCode, ok = opNameToOpCode[absYAddr][lowerOpName]
			if !ok {
				i.Payload = []byte{i.OpCode, 0, 0}
				binary.LittleEndian.PutUint16(i.Payload[1:], uint16(i.Value))
				return nil
			}
			return errors.New(fmt.Sprintf("Line %d: Unrecognized absolute, Y instruction: %s", i.Line, i.OpName))
		}
		return errors.New(fmt.Sprintf("Line %d: Register argument must be X or Y", i.Line))
	case DirectWithLabelIndexedInstruction:
		lowerRegName := strings.ToLower(i.RegisterName)
		if lowerRegName == "x" {
			i.OpCode, ok = opNameToOpCode[absXAddr][lowerOpName]
			if ok {
				// 0s are placeholder until we resolve labels
				i.Payload = []byte{i.OpCode, 0, 0}
				return nil
			}
			return errors.New(fmt.Sprintf("Line %d: Unrecognized direct, X instruction: %s", i.Line, i.OpName))
		} else if lowerRegName == "y" {
			i.OpCode, ok = opNameToOpCode[absYAddr][lowerOpName]
			if !ok {
				// 0s are placeholder until we resolve labels
				i.Payload = []byte{i.OpCode, 0, 0}
				return nil
			}
			return errors.New(fmt.Sprintf("Line %d: Unrecognized direct, Y instruction: %s", i.Line, i.OpName))
		}
		return errors.New(fmt.Sprintf("Line %d: Register argument must be X or Y", i.Line))
	case IndirectXInstruction:
		i.OpCode, ok = opNameToOpCode[xIndexIndirectAddr][lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized indirect x indexed instruction: %s", i.Line, i.OpName))
		}
		if i.Value > 0xff {
			return errors.New(fmt.Sprintf("Line %d: Indirect X memory address is limited to 1 byte.", i.Line))
		}
		i.Payload = []byte{i.OpCode, byte(i.Value)}
	case IndirectYInstruction:
		i.OpCode, ok = opNameToOpCode[indirectYIndexAddr][lowerOpName]
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized indirect y indexed instruction: %s", i.Line, i.OpName))
		}
		if i.Value > 0xff {
			return errors.New(fmt.Sprintf("Line %d: Indirect Y memory address is limited to 1 byte.", i.Line))
		}
		i.Payload = []byte{i.OpCode, byte(i.Value)}
	case IndirectInstruction:
		if lowerOpName != "jmp" {
			return errors.New(fmt.Sprintf("Line %d: Unrecognized indirect instruction: %s", i.Line, i.OpName))
		}
		i.Payload = []byte{0x6c, 0, 0}
		if i.Value > 0xffff {
			return errors.New(fmt.Sprintf("Line %d: Memory address is limited to 2 bytes.", i.Line))
		}
		binary.LittleEndian.PutUint16(i.Payload[1:], uint16(i.Value))
	}
	return nil
}

func (i *Instruction) Assemble(sg symbolGetter) error {
	// fill in the rest of the payload
	var ok bool
	switch i.Type {
	default: panic("unexpected instruction type")
	case ImmediateInstruction, ImpliedInstruction, DirectInstruction,
		DirectIndexedInstruction, IndirectXInstruction, IndirectYInstruction,
		IndirectInstruction:
		// nothing to do
	case DirectWithLabelInstruction:
		i.Value, ok = sg.getSymbol(i.LabelName, i.Offset)
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Undefined label: %s", i.Line, i.LabelName))
		}
		if i.Value > 0xffff {
			return errors.New(fmt.Sprintf("Line %d: Symbol must fit into 2 bytes: %s", i.Line, i.LabelName))
		}
		if len(i.Payload) == 2 {
			// relative address
			delta := i.Value - (i.Offset + len(i.Payload))
			if delta > 127 || delta < -128 {
				return errors.New(fmt.Sprintf("Line %d: Label address must be within 127 bytes of instruction address.", i.Line))
			}
			i.Payload[1] = byte(delta)
			return nil
		}
		// absolute address
		binary.LittleEndian.PutUint16(i.Payload[1:], uint16(i.Value))
	case DirectWithLabelIndexedInstruction:
		i.Value, ok = sg.getSymbol(i.LabelName, i.Offset)
		if !ok {
			return errors.New(fmt.Sprintf("Line %d: Undefined symbol: %s", i.Line, i.LabelName))
		}
		if i.Value > 0xffff {
			return errors.New(fmt.Sprintf("Line %d: Symbol must fit into 2 bytes: %s", i.Line, i.LabelName))
		}
		binary.LittleEndian.PutUint16(i.Payload[1:], uint16(i.Value))
	}
	return nil
}

func (s *DataStatement) Resolve() error {
	size := 0
	for e := s.dataList.Front(); e != nil; e = e.Next() {
		switch t := e.Value.(type) {
		case *StringDataItem:
			switch s.Type {
			default: panic("unknown DataStatement Type")
			case ByteDataStmt:
				size += len(*t)
			case WordDataStmt:
				return errors.New(fmt.Sprintf("Line %d: string invalid in data word statement.", s.Line))
			}
		case *IntegerDataItem:
			switch s.Type {
			default: panic("unknown DataStatement Type")
			case ByteDataStmt:
				if *t > 0xff {
					return errors.New(fmt.Sprintf("Line %d: Integer byte data item limited to 1 byte.", s.Line))
				}
				size += 1
			case WordDataStmt:
				if *t > 0xffff {
					return errors.New(fmt.Sprintf("Line %d: Integer word data item limited to 2 bytes.", s.Line))
				}
				size += 2
			}
		case *LabelCall:
			switch s.Type {
			default: panic("unknown DataStatement Type")
			case ByteDataStmt:
				return errors.New(fmt.Sprintf("Line %d: label invalid in byte data statement.", s.Line))
			case WordDataStmt:
				size += 2
			}
		default:
			panic("unknown data item type")
		}
	}
	s.Payload = make([]byte, size)
	return nil
}

func (s *DataStatement) Assemble(sg symbolGetter) error {
	offset := 0
	for e := s.dataList.Front(); e != nil; e = e.Next() {
		switch t := e.Value.(type) {
		case *StringDataItem:
			if s.Type != ByteDataStmt {
				panic("expected ByteDataStmt")
			}
			str := string(*t)
			bytes.NewBuffer(s.Payload[offset:]).WriteString(str)
			offset += len(str)
		case *IntegerDataItem:
			switch s.Type {
			default: panic("unknown DataStatement Type")
			case ByteDataStmt:
				s.Payload[offset] = byte(*t)
				offset += 1
			case WordDataStmt:
				binary.LittleEndian.PutUint16(s.Payload[offset:], uint16(*t))
				offset += 2
			}
		case *LabelCall:
			if s.Type != WordDataStmt {
				panic("expected WordDataStmt")
			}
			symbolValue, ok := sg.getSymbol(t.LabelName, offset)
			if !ok {
				return errors.New(fmt.Sprintf("Line %d: Undefined symbol: %s", s.Line, t.LabelName))
			}
			if symbolValue > 0xffff {
				return errors.New(fmt.Sprintf("Line %d: Symbol must fit into 2 bytes: %s", s.Line, t.LabelName))
			}
			binary.LittleEndian.PutUint16(s.Payload[offset:], uint16(symbolValue))
			offset += 2
		default:
			panic("unknown data item type")
		}
	}
	return nil
}

func (p *Program) Assemble(w io.Writer) error {
	writer := bufio.NewWriter(w)

	offset := 0
	expectedOffset := 0
	firstOrg := true
	orgFillValue := byte(0)

	for e := p.List.Front(); e != nil; e = e.Next() {
		switch t := e.Value.(type) {
		default: panic("unexpected node")
		case *LabelStatement:
			// nothing to do
		case *OrgPseudoOp:
			offset = t.Value
			orgFillValue = t.Fill
			if firstOrg {
				firstOrg = false
				expectedOffset = offset
			}
		case Assembler:
			for t.GetOffset() > expectedOffset {
				// org fill
				err := writer.WriteByte(orgFillValue)
				if err != nil {
					return err
				}
				expectedOffset += 1
			}
			err := t.Assemble(p)
			if err != nil {
				return err
			}
			_, err = writer.Write(t.GetPayload())
			if err != nil {
				return err
			}
			offset += len(t.GetPayload())
			expectedOffset = offset
		}
	}

	writer.Flush()
	return nil
}

func (p *Program) AssembleToFile(filename string) error {
	fd, err := os.Create(filename)
	if err != nil {
		return err
	}

	err = p.Assemble(fd)
	err2 := fd.Close()
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}

	return nil
}

func (ast ProgramAst) ExpandLabeledStatements() {
	for e := ast.List.Front(); e != nil; e = e.Next() {
		s, ok := e.Value.(*LabeledStatement)
		if ok {
			elemToDel := e
			e = ast.List.InsertAfter(s.Label, e)
			e = ast.List.InsertAfter(s.Stmt, e)
			ast.List.Remove(elemToDel)
		}
	}
}

func (p *Program) Resolve() {
	offset := 0
	for e := p.List.Front(); e != nil; e = e.Next() {
		switch t := e.Value.(type) {
		default: panic("unexpected node")
		case *AssignStatement:
			p.Variables[t.VarName] = t.Value
		case *OrgPseudoOp:
			offset = t.Value
		case *LabelStatement:
			if offset >= 0xffff {
				err := fmt.Sprintf("Line %d: Label memory address must fit in 2 bytes.", t.Line)
				p.Errors = append(p.Errors, err)
				return
			}
			_, exists := p.Labels[t.LabelName]
			if exists {
				err := fmt.Sprintf("Line %d: Label %s already defined.", t.Line, t.LabelName)
				p.Errors = append(p.Errors, err)
				return
			}
			p.Labels[t.LabelName] = offset
		case Assembler:
			if offset >= 0xffff {
				err := fmt.Sprintf("Line %d: Instruction is at offset $%04x which is greater than 2 bytes.", t.GetLine(), offset)
				p.Errors = append(p.Errors, err)
				return
			}
			p.Offsets[offset] = e
			t.SetOffset(offset)
			err := t.Resolve()
			if err != nil {
				p.Errors = append(p.Errors, err.Error())
				return
			}
			offset += len(t.GetPayload())
		}
	}
}

func (ast ProgramAst) ToProgram() (p *Program) {
	ast.ExpandLabeledStatements()
	p = &Program{
		List: ast.List,
		Labels: make(map[string]int),
		Offsets: make(map[int]*list.Element),
		Variables: make(map[string]int),
	}
	p.Resolve()
	return
}
