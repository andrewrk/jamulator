package jamulator

import (
	"bufio"
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type SourceWriter struct {
	program *Program
	writer  *bufio.Writer
	Errors  []string
}

type Renderer interface {
	Render() string
}

type Disassembly struct {
	Errors []string

	rom  *Rom
	list *list.List
	// maps memory offset to node
	offsets map[int]*list.Element
	offset  int
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
	if elem == nil {
		return 0, errors.New("not enough bytes for word")
	}
	next := elem.Next()
	if next == nil {
		return 0, errors.New("not enough bytes for word")
	}

	b1, err := d.elemAsByte(elem)
	if err != nil {
		return 0, err
	}
	b2, err := d.elemAsByte(next)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint16([]byte{b1, b2}), nil
}

func (d *Disassembly) getLabelAt(addr int) string {
	elem := d.elemAtAddr(addr)
	if elem == nil {
		// cannot get/make label; there is already code there
		return ""
	}
	stmt, ok := elem.Value.(*LabeledStatement)
	if ok {
		return stmt.LabelName
	}
	prev := elem.Prev()
	if prev != nil {
		stmt, ok = prev.Value.(*LabeledStatement)
		if ok {
			return stmt.LabelName
		}
	}
	// put one there
	i := new(LabeledStatement)
	i.LabelName = fmt.Sprintf("Label_%04x", addr)
	d.list.InsertBefore(i, elem)
	return i.LabelName
}

func (d *Disassembly) removeElemAt(addr int) {
	elem := d.elemAtAddr(addr)
	d.list.Remove(elem)
	delete(d.offsets, addr)
}

func (d *Disassembly) markAsInstruction(addr int) error {
	if addr < 0x8000 {
		// non-ROM address. nothing we can do
		return nil
	}
	elem := d.elemAtAddr(addr)
	opCode, err := d.elemAsByte(elem)
	if err != nil {
		return err
	}
	opCodeInfo := opCodeDataMap[opCode]
	switch opCodeInfo.addrMode {
	case nilAddr:
		return errors.New("cannot disassemble as instruction: bad op code")
	case absAddr:
		// convert data statements into instruction statement
		w, err := d.elemAsWord(elem.Next())
		if err != nil {
			return err
		}
		targetAddr := int(w)
		if targetAddr >= 0x8000 {
			// destination is in PRG ROM
			i := new(DirectWithLabelInstruction)
			i.OpName = opCodeInfo.opName
			i.Offset = addr
			i.Size = 3
			i.OpCode = opCode
			i.LabelName = d.getLabelAt(targetAddr)
			elem.Value = i
		} else {
			i := new(DirectInstruction)
			i.OpName = opCodeInfo.opName
			i.Offset = addr
			i.Payload = []byte{opCode, 0, 0}
			i.Value = targetAddr
			binary.LittleEndian.PutUint16(i.Payload[1:], w)
			elem.Value = i
		}

		d.removeElemAt(addr + 1)
		d.removeElemAt(addr + 2)

		switch opCode {
		case 0x4c: // jmp
			d.markAsInstruction(targetAddr)
		case 0x20: // jsr
			d.markAsInstruction(targetAddr)
			d.markAsInstruction(addr + 3)
		default:
			d.markAsInstruction(addr + 3)
		}
	case absXAddr, absYAddr:
		w, err := d.elemAsWord(elem.Next())
		if err != nil {
			return err
		}
		regName := "X"
		if opCodeInfo.addrMode == absYAddr {
			regName = "Y"
		}
		targetAddr := int(w)
		inPrgROM := targetAddr >= 0x8000
		var labelName string
		if inPrgROM {
			labelName = d.getLabelAt(targetAddr)
			// if labelName is blank string, we were unable to get a label
			// at that address, and so we should fall back to direct indexed.
		}
		if inPrgROM && len(labelName) > 0 {
			i := new(DirectWithLabelIndexedInstruction)
			i.OpName = opCodeInfo.opName
			i.Offset = addr
			i.LabelName = d.getLabelAt(targetAddr)
			i.RegisterName = regName
			i.Size = 3
			i.OpCode = opCode
			elem.Value = i
		} else {
			i := new(DirectIndexedInstruction)
			i.OpName = opCodeInfo.opName
			i.Offset = addr
			i.Payload = []byte{opCode, 0, 0}
			i.Value = targetAddr
			i.RegisterName = regName
			binary.LittleEndian.PutUint16(i.Payload[1:], w)
			elem.Value = i
		}

		d.removeElemAt(addr + 1)
		d.removeElemAt(addr + 2)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 3)
	case immedAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i := new(ImmediateInstruction)
		i.OpName = opCodeInfo.opName
		i.Offset = addr
		i.OpCode = opCode
		i.Value = int(v)
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	case impliedAddr:
		i := new(ImpliedInstruction)
		i.OpName = opCodeInfo.opName
		i.Offset = addr
		i.OpCode = opCode
		elem.Value = i

		if opCode == 0x40 {
			// RTI
		} else if opCode == 0x60 {
			// RTS
		} else if opCode == 0x00 {
			// BRK
		} else {
			// next thing is definitely an instruction
			d.markAsInstruction(addr + 1)
		}
	case indirectAddr:
		// note: only JMP uses this
		w, err := d.elemAsWord(elem.Next())
		if err != nil {
			return err
		}
		i := new(IndirectInstruction)
		i.OpName = opCodeInfo.opName
		i.Offset = addr
		i.Payload = []byte{opCode, 0, 0}
		i.Value = int(w)
		binary.LittleEndian.PutUint16(i.Payload[1:], w)
		elem.Value = i

		d.removeElemAt(addr + 1)
		d.removeElemAt(addr + 2)

		if opCode == 0x6c {
			// JMP
		} else {
			// next thing is definitely an instruction
			d.markAsInstruction(addr + 3)
		}
	case xIndexIndirectAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i := new(IndirectXInstruction)
		i.OpName = opCodeInfo.opName
		i.Offset = addr
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	case indirectYIndexAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i := new(IndirectYInstruction)
		i.OpName = opCodeInfo.opName
		i.Offset = addr
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	case relativeAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i := new(DirectWithLabelInstruction)
		i.OpName = opCodeInfo.opName
		i.Offset = addr
		i.Size = 2
		i.OpCode = opCode
		targetAddr := addr + 2 + int(int8(v))
		i.LabelName = d.getLabelAt(targetAddr)
		elem.Value = i

		d.removeElemAt(addr + 1)

		// mark both targets of the branch as instructions
		d.markAsInstruction(addr + 2)
		d.markAsInstruction(targetAddr)
	case zeroPageAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i := new(DirectInstruction)
		i.OpName = opCodeInfo.opName
		i.Offset = addr
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	case zeroXIndexAddr, zeroYIndexAddr:
		regName := "X"
		if opCodeInfo.addrMode == zeroYIndexAddr {
			regName = "Y"
		}
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i := new(DirectIndexedInstruction)
		i.OpName = opCodeInfo.opName
		i.Offset = addr
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		i.RegisterName = regName
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	}
	return nil
}

func (d *Disassembly) Error() string {
	return strings.Join(d.Errors, "\n")
}

func (d *Disassembly) ToProgram() *Program {
	p := new(Program)
	p.Ast = new(ProgramAST)
	p.Ast.statements = make(StatementList, 0, d.list.Len())
	p.offsets = map[int]Node{}

	orgStatement := new(OrgPseudoOp)
	orgStatement.Fill = 0xff // this is the default; causes it to be left off when rendered
	orgStatement.Value = d.offset
	p.Ast.statements = append(p.Ast.statements, orgStatement)

	for e := d.list.Front(); e != nil; e = e.Next() {
		p.Ast.statements = append(p.Ast.statements, e.Value.(Node))
	}
	for k, v := range d.offsets {
		n := v.Value.(Node)
		p.offsets[k] = n
		// if 1 bank, it's mirrored at 0x8000 and 0xc000
		if len(d.rom.PrgRom) == 1 {
			p.offsets[k-0x4000] = n
		}
	}
	return p
}

func (d *Disassembly) readAllAsData() {
	d.offset = 0x10000 - 0x4000*len(d.rom.PrgRom)
	offset := d.offset
	for _, bank := range d.rom.PrgRom {
		for _, b := range bank {
			stmt := new(DataStatement)
			stmt.dataList = make(DataList, 1)
			item := IntegerDataItem(b)
			stmt.dataList[0] = &item
			stmt.Offset = offset
			stmt.Size = 1
			d.offsets[stmt.Offset] = d.list.PushBack(stmt)
			offset += 1
		}
	}
}

func (d *Disassembly) insertLabelAt(addr int, name string) {
	elem := d.elemAtAddr(addr)
	stmt := new(LabeledStatement)
	stmt.LabelName = name
	d.list.InsertBefore(stmt, elem)
}

func (d *Disassembly) elemAtAddr(addr int) *list.Element {
	elem, ok := d.offsets[addr]
	if ok {
		return elem
	}
	// if there is only 1 prg rom bank, it is at 0x8000 and mirrored at 0xc000
	if len(d.rom.PrgRom) == 1 && addr < 0xc000 {
		return d.offsets[addr+0x4000]
	}
	return nil
}

func (d *Disassembly) markAsDataWordLabel(addr int, name string) {
	elem1 := d.elemAtAddr(addr)
	elem2 := elem1.Next()
	s1 := elem1.Value.(*DataStatement)
	s2 := elem2.Value.(*DataStatement)
	if len(s1.dataList) != 1 {
		panic("expected DataList len 1")
	}
	if len(s2.dataList) != 1 {
		panic("expected DataList len 1")
	}
	n1 := s1.dataList[0].(*IntegerDataItem)
	n2 := s2.dataList[0].(*IntegerDataItem)

	targetAddr := binary.LittleEndian.Uint16([]byte{byte(*n1), byte(*n2)})

	newStmt := new(DataWordStatement)
	newStmt.Offset = addr
	newStmt.Size = 2
	if targetAddr >= 0x8000 {
		// target in PRG ROM
		newStmt.dataList = WordList{&LabelCall{name}}
	} else {
		tmp := IntegerDataItem(targetAddr)
		newStmt.dataList = WordList{&tmp}
	}
	elem1.Value = newStmt
	d.removeElemAt(addr + 1)

	if targetAddr >= 0x8000 {
		// target in PRG ROM
		d.insertLabelAt(int(targetAddr), name)
		d.markAsInstruction(int(targetAddr))
	}
}

func (d *Disassembly) collapseDataStatements() {
	if d.list.Len() < 2 {
		return
	}
	const MAX_DATA_LIST_LEN = 16
	for e := d.list.Front().Next(); e != nil; e = e.Next() {
		dataStmt, ok := e.Value.(*DataStatement)
		if !ok {
			continue
		}
		prev, ok := e.Prev().Value.(*DataStatement)
		if !ok {
			continue
		}
		if len(prev.dataList)+len(dataStmt.dataList) > MAX_DATA_LIST_LEN {
			continue
		}
		for _, v := range dataStmt.dataList {
			prev.dataList = append(prev.dataList, v)
		}
		elToDel := e
		e = e.Prev()
		d.list.Remove(elToDel)

	}
}

func allAscii(dl DataList) bool {
	for _, v := range dl {
		switch t := v.(type) {
		case *IntegerDataItem:
			if *t < 32 || *t > 126 || *t == '"' {
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
	for _, v := range dl {
		switch t := v.(type) {
		case *IntegerDataItem:
			str += string(*t)
		case *StringDataItem:
			str += string(*t)
		default:
			panic("unknown data item type")
		}
	}
	return str
}

const orgMinRepeatAmt = 64

type orgIdentifier struct {
	repeatingByte byte
	firstElem     *list.Element
	repeatCount   int
	dis           *Disassembly
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
	if d.list.Len() < orgMinRepeatAmt {
		return
	}
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
	if d.list.Len() < 3 {
		return
	}
	for e := d.list.Front().Next().Next(); e != nil; e = e.Next() {
		dataStmt, ok := e.Value.(*DataStatement)
		if !ok {
			continue
		}
		if !allAscii(dataStmt.dataList) {
			e = e.Next()
			if e == nil {
				break
			}
			e = e.Next()
			if e == nil {
				break
			}
			continue
		}
		prev1, ok := e.Prev().Value.(*DataStatement)
		if !ok {
			continue
		}
		if !allAscii(prev1.dataList) {
			e = e.Next()
			if e == nil {
				break
			}
			continue
		}
		prev2, ok := e.Prev().Prev().Value.(*DataStatement)
		if !ok {
			continue
		}
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
		if e == nil {
			break
		}
	}
}

func (r *Rom) Disassemble() (*Program, error) {
	if len(r.PrgRom) != 1 && len(r.PrgRom) != 2 {
		return nil, errors.New("only 1 or 2 prg rom banks supported")
	}

	dis := new(Disassembly)
	dis.rom = r
	dis.list = new(list.List)
	dis.offsets = make(map[int]*list.Element)

	dis.readAllAsData()

	// use the known entry points to recursively disassemble data statements
	dis.markAsDataWordLabel(0xfffa, "NMI_Routine")
	dis.markAsDataWordLabel(0xfffc, "Reset_Routine")
	dis.markAsDataWordLabel(0xfffe, "IRQ_Routine")

	dis.identifyOrgs()
	dis.groupAsciiStrings()
	dis.collapseDataStatements()

	p := dis.ToProgram()
	p.ChrRom = r.ChrRom
	p.Mirroring = r.Mirroring

	if len(dis.Errors) > 0 {
		return p, dis
	}
	return p, nil
}

func Disassemble(reader io.Reader) (*Program, error) {
	r := new(Rom)
	bank, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	r.PrgRom = append(r.PrgRom, bank)
	return r.Disassemble()
}

func DisassembleFile(filename string) (*Program, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	p, err := Disassemble(fd)
	err2 := fd.Close()
	if err != nil {
		return nil, err
	}
	if err2 != nil {
		return nil, err2
	}

	return p, nil
}

func (sw SourceWriter) Visit(n Node) {
	switch t := n.(type) {
	case Renderer:
		sw.writer.WriteString(t.Render())
	}
}

func (SourceWriter) VisitEnd(Node) {}

func (sw SourceWriter) Error() string {
	return strings.Join(sw.Errors, "\n")
}

func (i *ImmediateInstruction) Render() string {
	return fmt.Sprintf("%s #$%02x\n", i.OpName, i.Value)
}

func (i *ImpliedInstruction) Render() string {
	return fmt.Sprintf("%s\n", i.OpName)
}

func (i *DirectInstruction) Render() string {
	return fmt.Sprintf("%s $%02x\n", i.OpName, i.Value)
}

func (i *DirectWithLabelInstruction) Render() string {
	return fmt.Sprintf("%s %s\n", i.OpName, i.LabelName)
}

func (i *DirectIndexedInstruction) Render() string {
	return fmt.Sprintf("%s $%02x, %s\n", i.OpName, i.Value, i.RegisterName)
}

func (i *DirectWithLabelIndexedInstruction) Render() string {
	return fmt.Sprintf("%s %s, %s\n", i.OpName, i.LabelName, i.RegisterName)
}

func (i *IndirectInstruction) Render() string {
	return fmt.Sprintf("%s ($%02x)\n", i.OpName, i.Value)
}

func (i *IndirectXInstruction) Render() string {
	return fmt.Sprintf("%s ($%02x, X)\n", i.OpName, i.Value)
}

func (i *IndirectYInstruction) Render() string {
	return fmt.Sprintf("%s ($%02x), Y\n", i.OpName, i.Value)
}

func (i *OrgPseudoOp) Render() string {
	if i.Fill == 0xff {
		return fmt.Sprintf("org $%04x\n", i.Value)
	}
	return fmt.Sprintf("org $%04x, $%02x\n", i.Value, i.Fill)
}

func (s *DataStatement) Render() string {
	buf := new(bytes.Buffer)
	buf.WriteString("dc.b ")
	for i, node := range s.dataList {
		switch t := node.(type) {
		case *StringDataItem:
			buf.WriteString("\"")
			buf.WriteString(string(*t))
			buf.WriteString("\"")
		case *IntegerDataItem:
			buf.WriteString(fmt.Sprintf("#$%02x", int(*t)))
		}
		if i < len(s.dataList)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

func (s *DataWordStatement) Render() string {
	buf := new(bytes.Buffer)
	buf.WriteString("dc.w ")
	for i, node := range s.dataList {
		switch t := node.(type) {
		case *LabelCall:
			buf.WriteString(t.LabelName)
		case *IntegerDataItem:
			buf.WriteString(fmt.Sprintf("#$%04x", int(*t)))
		}
		if i < len(s.dataList)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

func (s *LabeledStatement) Render() string {
	buf := new(bytes.Buffer)
	buf.WriteString(s.LabelName)
	buf.WriteString(":")

	if s.Stmt == nil {
		buf.WriteString("\n")
		return buf.String()
	}
	buf.WriteString(" ")
	n := s.Stmt.(Renderer)
	buf.WriteString(n.Render())
	return buf.String()
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
	if err != nil {
		return err
	}
	err = p.WriteSource(fd)
	err2 := fd.Close()
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	return nil
}
