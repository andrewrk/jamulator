package asm6502

import (
	"io"
	"os"
	"fmt"
	"bufio"
	"errors"
	"strings"
	"container/list"
	//"encoding/binary"
)

type SourceWriter struct {
	program *Program
	writer *bufio.Writer
	Errors []string
}

type Renderer interface {
	Render(SourceWriter) error
}

//func (p *Program) disassembleAsInstruction(index int) {
//	n := p.Ast.statements[index].(*DataStatement)
//	if len(n.dataList) != 0 { panic("expected DataStatement of size 1") }
//	intDataItem := n.dataList[0].(*IntegerDataItem)
//	opCode := byte(*intDataItem)
//	opCodeInfo := opCodeDataMap[opCode]
//	switch opCodeInfo.addrMode {
//	case nilAddr:
//		panic("can't disassemble as instruction: bad op code")
//	case absAddr:
//		i := new(DirectInstruction)
//		i.OpName = opCodeInfo.opName
//		i.Payload = []byte{opCode, 0, 0}
//		i.Value = 
//		valuePart := i.Payload[1:]
//		nn, err := io.ReadAtLeast(r, valuePart, 2)
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			if nn > 0 { p.appendDataByte(valuePart[0]) }
//			if nn > 1 { p.appendDataByte(valuePart[1]) }
//			break
//		}
//		if err != nil { return nil, err }
//		i.Value = int(binary.LittleEndian.Uint16(valuePart))
//		_, hasZeroPageVersion := zeroPageOpCode[i.OpName]
//		if hasZeroPageVersion && i.Value <= 0xff {
//			// this would be recompiled as a zero-page instruction - must be data
//			p.appendDataByte(opCode)
//			p.appendDataByte(valuePart[0])
//			p.appendDataByte(valuePart[1])
//			break
//		}
//		p.Ast.statements = append(p.Ast.statements, i)
//	case absXAddr:
//		i := new(DirectIndexedInstruction)
//		i.OpName = opCodeInfo.opName
//		i.Payload = []byte{opCode, 0, 0}
//		valuePart := i.Payload[1:]
//		nn, err := io.ReadAtLeast(r, valuePart, 2)
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			if nn > 0 { p.appendDataByte(valuePart[0]) }
//			if nn > 1 { p.appendDataByte(valuePart[1]) }
//			break
//		}
//		if err != nil { return nil, err }
//		i.Value = int(binary.LittleEndian.Uint16(valuePart))
//		_, hasZeroPageVersion := zeroPageXOpcode[i.OpName]
//		if hasZeroPageVersion && i.Value <= 0xff {
//			// this would be recompiled as a zero-page instruction - must be data
//			p.appendDataByte(opCode)
//			p.appendDataByte(valuePart[0])
//			p.appendDataByte(valuePart[1])
//			break
//		}
//		i.RegisterName = "X"
//		p.Ast.statements = append(p.Ast.statements, i)
//	case absYAddr:
//		i := new(DirectIndexedInstruction)
//		i.OpName = opCodeInfo.opName
//		i.Payload = []byte{opCode, 0, 0}
//		valuePart := i.Payload[1:]
//		nn, err := io.ReadAtLeast(r, valuePart, 2)
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			if nn > 0 { p.appendDataByte(valuePart[0]) }
//			if nn > 1 { p.appendDataByte(valuePart[1]) }
//			break
//		}
//		if err != nil { return nil, err }
//		i.Value = int(binary.LittleEndian.Uint16(valuePart))
//		_, hasZeroPageVersion := zeroPageYOpCode[i.OpName]
//		if hasZeroPageVersion && i.Value <= 0xff {
//			// this would be recompiled as a zero-page instruction - must be data
//			p.appendDataByte(opCode)
//			p.appendDataByte(valuePart[0])
//			p.appendDataByte(valuePart[1])
//			break
//		}
//		i.RegisterName = "Y"
//		p.Ast.statements = append(p.Ast.statements, i)
//	case immedAddr:
//		i := new(ImmediateInstruction)
//		i.OpName = opCodeInfo.opName
//		v, err := r.ReadByte()
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			break
//		}
//		if err != nil { return nil, err }
//		i.Value = int(v)
//		p.Ast.statements = append(p.Ast.statements, i)
//	case impliedAddr:
//		i := new(ImpliedInstruction)
//		i.OpName = opCodeInfo.opName
//		p.Ast.statements = append(p.Ast.statements, i)
//	case indirectAddr:
//		// note: only JMP uses this
//		i := new(IndirectInstruction)
//		i.OpName = opCodeInfo.opName
//		i.Payload = []byte{opCode, 0, 0}
//		valuePart := i.Payload[1:]
//		nn, err := io.ReadAtLeast(r, valuePart, 2)
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			if nn > 0 { p.appendDataByte(valuePart[0]) }
//			if nn > 1 { p.appendDataByte(valuePart[1]) }
//			continue
//		}
//		if err != nil { return nil, err }
//		i.Value = int(binary.LittleEndian.Uint16(valuePart))
//		p.Ast.statements = append(p.Ast.statements, i)
//	case xIndexIndirectAddr:
//		i := new(IndirectXInstruction)
//		i.OpName = opCodeInfo.opName
//		v, err := r.ReadByte()
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			break
//		}
//		if err != nil { return nil, err }
//		i.Payload = []byte{opCode, v}
//		i.Value = int(v)
//		p.Ast.statements = append(p.Ast.statements, i)
//	case indirectYIndexAddr:
//		i := new(IndirectYInstruction)
//		i.OpName = opCodeInfo.opName
//		v, err := r.ReadByte()
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			break
//		}
//		if err != nil { return nil, err }
//		i.Payload = []byte{opCode, v}
//		i.Value = int(v)
//		p.Ast.statements = append(p.Ast.statements, i)
//	case relativeAddr:
//		i := new(DirectInstruction)
//		i.OpName = opCodeInfo.opName
//		v, err := r.ReadByte()
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			break
//		}
//		if err != nil { return nil, err }
//		i.Payload = []byte{opCode, v}
//		i.Value = int(v)
//		p.Ast.statements = append(p.Ast.statements, i)
//	case zeroPageAddr:
//		i := new(DirectInstruction)
//		i.OpName = opCodeInfo.opName
//		v, err := r.ReadByte()
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			break
//		}
//		if err != nil { return nil, err }
//		i.Payload = []byte{opCode, v}
//		i.Value = int(v)
//		p.Ast.statements = append(p.Ast.statements, i)
//	case zeroXIndexAddr:
//		i := new(DirectIndexedInstruction)
//		i.OpName = opCodeInfo.opName
//		v, err := r.ReadByte()
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			break
//		}
//		if err != nil { return nil, err }
//		i.Payload = []byte{opCode, v}
//		i.Value = int(v)
//		i.RegisterName = "X"
//		p.Ast.statements = append(p.Ast.statements, i)
//	case zeroYIndexAddr:
//		i := new(DirectIndexedInstruction)
//		i.OpName = opCodeInfo.opName
//		v, err := r.ReadByte()
//		if err == io.EOF {
//			// oops, this must be data.
//			p.appendDataByte(opCode)
//			break
//		}
//		if err != nil { return nil, err }
//		i.Payload = []byte{opCode, v}
//		i.Value = int(v)
//		i.RegisterName = "Y"
//		p.Ast.statements = append(p.Ast.statements, i)
//	}
//}

type Disassembly struct {
	reader *bufio.Reader
	list *list.List
	// maps memory offset to node
	offsets map[int]*list.Element
}

func (d *Disassembly) toProgram() *Program {
	p := new(Program)
	p.Ast = new(ProgramAST)
	p.Ast.statements = make(StatementList, 0, d.list.Len())
	for e := d.list.Front(); e != nil; e = e.Next() {
		p.Ast.statements = append(p.Ast.statements, e.Value.(Node))
	}
	return p
}

func (d *Disassembly) readAllAsData() error {
	offset := 0xc000
	for {
		b, err := d.reader.ReadByte()
		if err == io.EOF { break }
		if err != nil { return err }
		i := new(DataStatement)
		i.dataList = make(DataList, 1)
		item := IntegerDataItem(b)
		i.dataList[0] = &item
		i.Offset = len(d.offsets)
		i.Size = 1
		d.offsets[offset] = d.list.PushBack(i)
		offset += 1
	}
	if offset != 0x10000 {
		return errors.New("Program is not the correct size.")
	}
	return nil
}

func (d *Disassembly) markAsDataWordLabel(elem *list.Element, name string) {
	// TODO: actually do something
}

func (d *Disassembly) collapseDataStatements() {
	if d.list.Len() < 2 { return }
	const MAX_DATA_LIST_LEN = 16
	for e := d.list.Front().Next(); e != nil; e = e.Next() {
		dataStmt, ok := e.Value.(*DataStatement)
		if !ok { continue }
		prev, ok := e.Prev().Value.(*DataStatement)
		if !ok { continue }
		if len(prev.dataList) + len(dataStmt.dataList) > MAX_DATA_LIST_LEN { continue }
		for _, v := range(dataStmt.dataList) {
			prev.dataList = append(prev.dataList, v)
		}
		elToDel := e
		e = e.Prev()
		d.list.Remove(elToDel)

	}
}


func allAscii(dl DataList) bool {
	for _, v := range(dl) {
		switch t := v.(type) {
		case *IntegerDataItem:
			if *t < 32 || *t > 126 {
				return false
			}
		case *StringDataItem:
			// nothing to do
		default:
			panic("unrecognized data list item")
		}
	}
	return true
}

func dataListToStr(dl DataList) string {
	str := ""
	for _, v := range(dl) {
		switch t := v.(type) {
		case *IntegerDataItem: str += string(*t)
		case *StringDataItem: str += string(*t)
		default: panic("unknown data item type")
		}
	}
	return str
}

func (d *Disassembly) groupAsciiStrings() {
	if d.list.Len() < 3 { return }
	for e := d.list.Front().Next().Next(); e != nil; e = e.Next() {
		dataStmt, ok := e.Value.(*DataStatement)
		if !ok { continue }
		if !allAscii(dataStmt.dataList) {
			e = e.Next()
			if e == nil { break }
			e = e.Next()
			if e == nil { break }
			continue
		}
		prev1, ok := e.Prev().Value.(*DataStatement)
		if !ok { continue }
		if !allAscii(prev1.dataList) {
			e = e.Next()
			if e == nil { break }
			continue
		}
		prev2, ok := e.Prev().Prev().Value.(*DataStatement)
		if !ok { continue }
		if !allAscii(prev2.dataList) {
			continue
		}
		// convert prev2 to string data item
		str := ""
		str += dataListToStr(prev2.dataList)
		str += dataListToStr(prev1.dataList)
		str += dataListToStr(dataStmt.dataList)
		prev2.dataList = make([]Node, 1)
		tmp := StringDataItem(str)
		prev2.dataList[0] = &tmp

		// delete prev1 and e
		e = e.Prev().Prev()
		d.list.Remove(e.Next())
		d.list.Remove(e.Next())
		e = e.Next()
		if e == nil { break }
	}
}

func Disassemble(reader io.Reader) (*Program, error) {
	dis := new(Disassembly)
	dis.reader = bufio.NewReader(reader)
	dis.list = new(list.List)
	dis.offsets = make(map[int]*list.Element)

	err := dis.readAllAsData()
	if err != nil { return nil, err }

	// use the known entry points to recursively disassemble data statements
	dis.markAsDataWordLabel(dis.offsets[0xfffa], "NMI_Routine")
	dis.markAsDataWordLabel(dis.offsets[0xfffc], "Reset_Routine")
	dis.markAsDataWordLabel(dis.offsets[0xfffe], "IRQ_Routine")

	dis.groupAsciiStrings()
	dis.collapseDataStatements()

	p := dis.toProgram()

	return p, nil
}

func DisassembleFile(filename string) (*Program, error) {
	fd, err := os.Open(filename)
	if err != nil { return nil, err}

	p, err := Disassemble(fd)
	err2 := fd.Close()
	if err != nil { return nil, err }
	if err2 != nil { return nil, err2 }

	return p, nil
}

func (sw SourceWriter) Visit(n Node) {
	switch t := n.(type) {
	case Renderer:
		err := t.Render(sw)
		if err != nil {
			sw.Errors = append(sw.Errors, err.Error())
		}
	}
}

func (SourceWriter) VisitEnd(Node) {}

func (sw SourceWriter) Error() string {
	return strings.Join(sw.Errors, "\n")
}

func (i *ImmediateInstruction) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString(fmt.Sprintf("%s #$%02x\n", i.OpName, i.Value))
	return err
}

func (i *ImpliedInstruction) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString(fmt.Sprintf("%s\n", i.OpName))
	return err
}

func (i *DirectInstruction) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString(fmt.Sprintf("%s $%02x\n", i.OpName, i.Value))
	return err
}

func (i *DirectIndexedInstruction) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString(fmt.Sprintf("%s $%02x, %s\n", i.OpName, i.Value, i.RegisterName))
	return err
}

func (i *IndirectInstruction) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString(fmt.Sprintf("%s ($%02x)\n", i.OpName, i.Value))
	return err
}

func (i *IndirectXInstruction) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString(fmt.Sprintf("%s ($%02x, X)\n", i.OpName, i.Value))
	return err
}

func (i *IndirectYInstruction) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString(fmt.Sprintf("%s ($%02x), Y\n", i.OpName, i.Value))
	return err
}

func (s *DataStatement) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString("dc.b ")
	if err != nil { return err }
	for i, node := range(s.dataList) {
		switch t := node.(type) {
		case *StringDataItem:
			_, err = sw.writer.WriteString("\"")
			if err != nil { return err }
			_, err = sw.writer.WriteString(string(*t))
			if err != nil { return err }
			_, err = sw.writer.WriteString("\"")
			if err != nil { return err }
		case *IntegerDataItem:
			_, err = sw.writer.WriteString(fmt.Sprintf("#$%02x", int(*t)))
			if err != nil { return err }
		}
		if i < len(s.dataList) - 1 {
			sw.writer.WriteString(", ")
		}
	}
	_, err = sw.writer.WriteString("\n")
	return err
}

func (p *Program) WriteSource(writer io.Writer) error {
	w := bufio.NewWriter(writer)

	sw := SourceWriter{p, w, make([]string, 0)}
	p.Ast.Ast(sw)
	w.Flush()

	if len(sw.Errors) > 0 {
		return sw
	}
	return nil
}

func (p *Program) WriteSourceFile(filename string) error {
	fd, err := os.Create(filename)
	if err != nil { return err }
	err = p.WriteSource(fd)
	err2 := fd.Close()
	if err != nil { return err }
	if err2 != nil { return err2 }
	return nil
}
