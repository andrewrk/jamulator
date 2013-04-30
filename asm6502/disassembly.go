package asm6502

import (
	"io"
	"os"
	"fmt"
	"bufio"
	"errors"
	"strings"
	"container/list"
	"encoding/binary"
)

type SourceWriter struct {
	program *Program
	writer *bufio.Writer
	Errors []string
}

type Renderer interface {
	Render(SourceWriter) error
}

func (d *Disassembly) elemAsByte(elem *list.Element) (byte, error) {
	if elem == nil {
		return 0, errors.New("not enough bytes for byte")
	}
	stmt, ok := elem.Value.(*DataStatement)
	if !ok {
		return 0, errors.New("already marked as instruction")
	}
	if len(stmt.dataList) != 1 {
		return 0, errors.New("expected DataStatement of size 1")
	}
	intDataItem, ok := stmt.dataList[0].(*IntegerDataItem)
	if !ok {
		return 0, errors.New("expected integer data item")
	}
	b := byte(*intDataItem)
	return b, nil
}

func (d *Disassembly) elemAsWord(elem *list.Element) (uint16, error) {
	if elem == nil { return 0, errors.New("not enough bytes for word") }
	next := elem.Next()
	if next == nil { return 0, errors.New("not enough bytes for word") }

	b1, err := d.elemAsByte(elem)
	if err != nil { return 0, err }
	b2, err := d.elemAsByte(next)
	if err != nil { return 0, err }

	return binary.LittleEndian.Uint16([]byte{b1, b2}), nil
}

func (d *Disassembly) markAsInstruction(addr int) error {
	elem := d.offsets[addr]
	opCode, err := d.elemAsByte(elem)
	if err != nil { return err }
	opCodeInfo := opCodeDataMap[opCode]
	switch opCodeInfo.addrMode {
	case nilAddr:
		panic("can't disassemble as instruction: bad op code")
	case absAddr:
		w, err := d.elemAsWord(elem.Next())
		if err != nil { return err }
		i := new(DirectInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, 0, 0}
		i.Value = int(w)
		binary.LittleEndian.PutUint16(i.Payload[1:], w)
		elem.Value = i
		d.list.Remove(elem.Next())
		d.list.Remove(elem.Next())
	case absXAddr:
		w, err := d.elemAsWord(elem.Next())
		if err != nil { return err }
		i := new(DirectIndexedInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, 0, 0}
		i.Value = int(w)
		i.RegisterName = "X"
		binary.LittleEndian.PutUint16(i.Payload[1:], w)
		elem.Value = i
		d.list.Remove(elem.Next())
		d.list.Remove(elem.Next())
	case absYAddr:
		w, err := d.elemAsWord(elem.Next())
		if err != nil { return err }
		i := new(DirectIndexedInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, 0, 0}
		i.Value = int(w)
		i.RegisterName = "Y"
		binary.LittleEndian.PutUint16(i.Payload[1:], w)
		elem.Value = i
		d.list.Remove(elem.Next())
		d.list.Remove(elem.Next())
	case immedAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil { return err }
		i := new(ImmediateInstruction)
		i.OpName = opCodeInfo.opName
		i.Value = int(v)
		elem.Value = i
		d.list.Remove(elem.Next())
	case impliedAddr:
		i := new(ImpliedInstruction)
		i.OpName = opCodeInfo.opName
		elem.Value = i
	case indirectAddr:
		// note: only JMP uses this
		w, err := d.elemAsWord(elem.Next())
		if err != nil { return err }
		i := new(IndirectInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, 0, 0}
		i.Value = int(w)
		binary.LittleEndian.PutUint16(i.Payload[1:], w)
		elem.Value = i
		d.list.Remove(elem.Next())
		d.list.Remove(elem.Next())
	case xIndexIndirectAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil { return err }
		i := new(IndirectXInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		elem.Value = i
		d.list.Remove(elem.Next())
	case indirectYIndexAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil { return err }
		i := new(IndirectYInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		elem.Value = i
		d.list.Remove(elem.Next())
	case relativeAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil { return err }
		i := new(DirectInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		elem.Value = i
		d.list.Remove(elem.Next())
	case zeroPageAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil { return err }
		i := new(DirectInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		elem.Value = i
		d.list.Remove(elem.Next())
	case zeroXIndexAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil { return err }
		i := new(DirectIndexedInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		i.RegisterName = "X"
		elem.Value = i
		d.list.Remove(elem.Next())
	case zeroYIndexAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil { return err }
		i := new(DirectIndexedInstruction)
		i.OpName = opCodeInfo.opName
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		i.RegisterName = "Y"
		elem.Value = i
		d.list.Remove(elem.Next())
	}
	return nil
}

type Disassembly struct {
	reader *bufio.Reader
	list *list.List
	// maps memory offset to node
	offsets map[int]*list.Element
	Errors []string
}

func (d *Disassembly) Error() string {
	return strings.Join(d.Errors, "\n")
}

func (d *Disassembly) toProgram() *Program {
	p := new(Program)
	p.Ast = new(ProgramAST)
	p.Ast.statements = make(StatementList, 0, d.list.Len())

	orgStatement := new(OrgPseudoOp)
	orgStatement.Value = 0xc000
	p.Ast.statements = append(p.Ast.statements, orgStatement)

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
		i.Offset = offset
		i.Size = 1
		d.offsets[offset] = d.list.PushBack(i)
		offset += 1
	}
	if offset != 0x10000 {
		return errors.New("Program is not the correct size.")
	}
	return nil
}

func (d *Disassembly) insertLabelAt(addr int, name string) {
	elem := d.offsets[addr]
	stmt := new(LabeledStatement)
	stmt.LabelName = name
	d.list.InsertBefore(stmt, elem)
}

func (d *Disassembly) markAsDataWordLabel(addr int, name string) {
	elem1 := d.offsets[addr]
	elem2 := elem1.Next()
	s1 := elem1.Value.(*DataStatement)
	s2 := elem2.Value.(*DataStatement)
	if len(s1.dataList) != 1 { panic("expected DataList len 1") }
	if len(s2.dataList) != 1 { panic("expected DataList len 1") }
	n1 := s1.dataList[0].(*IntegerDataItem)
	n2 := s2.dataList[0].(*IntegerDataItem)

	targetAddr := binary.LittleEndian.Uint16([]byte{byte(*n1), byte(*n2)})
	d.insertLabelAt(int(targetAddr), name)

	newStmt := new(DataWordStatement)
	newStmt.Offset = addr
	newStmt.Size = 2
	newStmt.dataList = WordList{&LabelCall{name}}
	newElem := d.list.InsertBefore(newStmt, elem1)
	d.offsets[addr] = newElem
	d.list.Remove(newElem.Next())
	d.list.Remove(newElem.Next())

	err := d.markAsInstruction(int(targetAddr))
	if err != nil {
		d.Errors = append(d.Errors, err.Error())
	}
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

const orgMinRepeatAmt = 64
type orgIdentifier struct {
	repeatingByte byte
	firstElem *list.Element
	repeatCount int
	dis *Disassembly
}

func (oi *orgIdentifier) stop(e *list.Element) {
	if oi.repeatCount > orgMinRepeatAmt {
		firstOffset := oi.firstElem.Value.(*DataStatement).Offset
		for i := 0; i < oi.repeatCount; i++ {
			delItem := oi.firstElem
			oi.firstElem = oi.firstElem.Next()
			oi.dis.list.Remove(delItem)
		}
		orgStmt := new(OrgPseudoOp)
		orgStmt.Value = firstOffset + oi.repeatCount
		orgStmt.Fill = oi.repeatingByte
		oi.dis.list.InsertBefore(orgStmt, e)
	}
	oi.repeatCount = 0
}

func (oi *orgIdentifier) start(e *list.Element, b byte) {
	oi.firstElem = e
	oi.repeatingByte = b
	oi.repeatCount = 1
}

func (oi *orgIdentifier) gotByte(e *list.Element, b byte) {
	if oi.repeatCount == 0 {
		oi.start(e, b)
	} else if b == oi.repeatingByte {
		oi.repeatCount += 1
	} else {
		oi.stop(e)
		oi.start(e, b)
	}
}

func (d *Disassembly) identifyOrgs() {
	// if a byte repeats enough, use an org statement
	if d.list.Len() < orgMinRepeatAmt { return }
	orgIdent := new(orgIdentifier)
	orgIdent.dis = d
	for e := d.list.Front().Next(); e != nil; e = e.Next() {
		dataStmt, ok := e.Value.(*DataStatement)
		if !ok || len(dataStmt.dataList) != 1 {
			orgIdent.stop(e)
			continue
		}
		v, ok := dataStmt.dataList[0].(*IntegerDataItem)
		if !ok {
			orgIdent.stop(e)
			continue
		}
		orgIdent.gotByte(e, byte(*v))
	}
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
	dis.Errors = make([]string, 0)

	err := dis.readAllAsData()
	if err != nil { return nil, err }

	// use the known entry points to recursively disassemble data statements
	dis.markAsDataWordLabel(0xfffa, "NMI_Routine")
	dis.markAsDataWordLabel(0xfffc, "Reset_Routine")
	dis.markAsDataWordLabel(0xfffe, "IRQ_Routine")

	dis.identifyOrgs()
	dis.groupAsciiStrings()
	dis.collapseDataStatements()

	p := dis.toProgram()

	if len(dis.Errors) > 0 { return p, dis }
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

func (i *OrgPseudoOp) Render(sw SourceWriter) error {
	var err error
	if i.Fill == 0 {
		_, err = sw.writer.WriteString(fmt.Sprintf("org $%04x\n", i.Value))
	} else {
		_, err = sw.writer.WriteString(fmt.Sprintf("org $%04x, %02x\n", i.Value, i.Fill))
	}
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

func (s *DataWordStatement) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString("dc.w ")
	if err != nil { return err }
	for i, node := range(s.dataList) {
		switch t := node.(type) {
		case *LabelCall:
			_, err = sw.writer.WriteString(t.LabelName)
			if err != nil { return err }
		case *IntegerDataItem:
			_, err = sw.writer.WriteString(fmt.Sprintf("#$%04x", int(*t)))
			if err != nil { return err }
		}
		if i < len(s.dataList) - 1 {
			sw.writer.WriteString(", ")
		}
	}
	_, err = sw.writer.WriteString("\n")
	return err
}

func (s *LabeledStatement) Render(sw SourceWriter) error {
	_, err := sw.writer.WriteString(s.LabelName)
	if err != nil { return err }
	_, err = sw.writer.WriteString(":")
	if err != nil { return err }

	if s.Stmt == nil {
		_, err = sw.writer.WriteString("\n")
		return err
	}
	_, err = sw.writer.WriteString(" ")
	if err != nil { return err }
	n := s.Stmt.(Renderer)
	return n.Render(sw)
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
