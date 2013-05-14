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
)

type Renderer interface {
	Render() string
}

type Disassembly struct {
	prog       *Program
	offset     int
	dynJumps   []int
	jumpTables map[int]bool
}

func (d *Disassembly) elemAsByte(elem *list.Element) (byte, error) {
	if elem == nil {
		return 0, errors.New("not enough bytes for byte")
	}
	stmt, ok := elem.Value.(*DataStatement)
	if !ok {
		return 0, errors.New("already marked as instruction")
	}
	if stmt.dataList.Len() != 1 {
		return 0, errors.New("expected DataStatement of size 1")
	}
	intDataItem, ok := stmt.dataList.Front().Value.(*IntegerDataItem)
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

func (d *Disassembly) getLabelAt(addr int, name string) (string, error) {
	elem := d.elemAtAddr(addr)
	if elem == nil {
		// cannot get/make label; there is already code there
		return "", errors.New("cannot insert a label mid-instruction")
	}
	stmt := d.elemLabelStmt(elem)
	if stmt != nil {
		return stmt.LabelName, nil
	}
	// put one there
	i := new(LabelStatement)
	i.LabelName = name
	if len(i.LabelName) == 0 {
		i.LabelName = fmt.Sprintf("Label_%04x", addr)
	}
	d.prog.List.InsertBefore(i, elem)

	// save in the label map
	d.prog.Labels[i.LabelName] = addr

	return i.LabelName, nil
}

func (d *Disassembly) removeElemAt(addr int) {
	elem := d.elemAtAddr(addr)
	d.prog.List.Remove(elem)
	delete(d.prog.Offsets, addr)
}

func (d *Disassembly) isJumpTable(addr int) bool {
	isJmpTable, ok := d.jumpTables[addr]
	if ok {
		return isJmpTable
	}
	isJmpTable = d.detectJumpTable(addr)
	d.jumpTables[addr] = isJmpTable
	return isJmpTable
}

func (d *Disassembly) detectJumpTable(addr int) bool {
	const (
		expectAsl = iota
		expectTay
		expectPlaA
		expectStaA
		expectPlaB
		expectStaB
		expectInyC
		expectLdaC
		expectStaC
		expectInYD
		expectLdaD
		expectStaD
		expectJmp
	)
	state := expectAsl
	var memA, memC int
	for elem := d.elemAtAddr(addr); elem != nil; elem = elem.Next() {
		switch state {
		case expectAsl:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0x0a {
				return false
			}
			state = expectTay
		case expectTay:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0xa8 {
				return false
			}
			state = expectPlaA
		case expectPlaA:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0x68 {
				return false
			}
			state = expectStaA
		case expectStaA:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0x85 && i.OpCode != 0x8d {
				return false
			}
			memA = i.Value
			state = expectPlaB
		case expectPlaB:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0x68 {
				return false
			}
			state = expectStaB
		case expectStaB:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0x85 && i.OpCode != 0x8d {
				return false
			}
			if i.Value != memA+1 {
				return false
			}
			state = expectInyC
		case expectInyC:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0xc8 {
				return false
			}
			state = expectLdaC
		case expectLdaC:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0xb1 {
				return false
			}
			if i.Value != memA {
				return false
			}
			state = expectStaC
		case expectStaC:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0x85 && i.OpCode != 0x8d {
				return false
			}
			memC = i.Value
			state = expectInYD
		case expectInYD:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0xc8 {
				return false
			}
			state = expectLdaD
		case expectLdaD:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0xb1 {
				return false
			}
			if i.Value != memA {
				return false
			}
			state = expectStaD
		case expectStaD:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0x85 && i.OpCode != 0x8d {
				return false
			}
			if i.Value != memC+1 {
				return false
			}
			state = expectJmp
		case expectJmp:
			i, ok := elem.Value.(*Instruction)
			if !ok {
				return false
			}
			if i.OpCode != 0x6c {
				return false
			}
			if i.Value != memC {
				return false
			}
			return true
		}
	}
	return false
}

func (d *Disassembly) markAsInstruction(addr int) error {
	if addr < 0x8000 {
		// non-ROM address. nothing we can do
		return nil
	}
	elem := d.elemAtAddr(addr)
	opCode, err := d.elemAsByte(elem)
	if err != nil {
		// already decoded as instruction
		return nil
	}
	i := new(Instruction)
	opCodeInfo := opCodeDataMap[opCode]
	i.OpName = opCodeInfo.opName
	i.OpCode = opCode
	i.Offset = addr
	switch opCodeInfo.addrMode {
	case nilAddr:
		return errors.New("cannot disassemble as instruction: bad op code")
	case absAddr:
		// convert data statements into instruction statement
		w, err := d.elemAsWord(elem.Next())
		if err != nil {
			return err
		}
		i.Value = int(w)
		i.Payload = []byte{opCode, 0, 0}
		binary.LittleEndian.PutUint16(i.Payload[1:], w)
		i.LabelName, err = d.getLabelAt(i.Value, "")
		if err == nil {
			i.Type = DirectWithLabelInstruction
		} else {
			i.Type = DirectInstruction
		}
		elem.Value = i

		d.removeElemAt(addr + 1)
		d.removeElemAt(addr + 2)

		switch opCode {
		case 0x4c: // jmp
			d.markAsInstruction(i.Value)
		case 0x20: // jsr
			d.markAsInstruction(i.Value)
			if d.isJumpTable(i.Value) {
				// mark this and remember to come back later
				d.dynJumps = append(d.dynJumps, addr+3)
			} else {
				d.markAsInstruction(addr + 3)
			}
		default:
			d.markAsInstruction(addr + 3)
		}
	case absXAddr, absYAddr:
		w, err := d.elemAsWord(elem.Next())
		if err != nil {
			return err
		}
		if opCodeInfo.addrMode == absYAddr {
			i.RegisterName = "Y"
		} else {
			i.RegisterName = "X"
		}
		i.Value = int(w)
		i.Payload = []byte{opCode, 0, 0}
		binary.LittleEndian.PutUint16(i.Payload[1:], w)
		i.LabelName, err = d.getLabelAt(i.Value, "")
		if err == nil {
			i.Type = DirectWithLabelIndexedInstruction
		} else {
			i.Type = DirectIndexedInstruction
		}
		elem.Value = i

		d.removeElemAt(addr + 1)
		d.removeElemAt(addr + 2)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 3)
	case immedAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i.Type = ImmediateInstruction
		i.Value = int(v)
		i.Payload = []byte{opCode, v}
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	case impliedAddr:
		i.Type = ImpliedInstruction
		i.Payload = []byte{opCode}
		elem.Value = i

		switch opCode {
		case 0x40: // RTI
		case 0x60: // RTS
		case 0x00: // BRK
		default:
			// next thing is definitely an instruction
			d.markAsInstruction(addr + 1)
		}
	case indirectAddr:
		// note: only JMP uses this
		w, err := d.elemAsWord(elem.Next())
		if err != nil {
			return err
		}
		i.Type = IndirectInstruction
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
		i.Type = IndirectXInstruction
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
		i.Type = IndirectYInstruction
		i.Value = int(v)
		i.Payload = []byte{opCode, v}
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	case relativeAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i.Type = DirectWithLabelInstruction
		i.Value = addr + 2 + int(int8(v))
		i.Payload = []byte{opCode, v}
		i.LabelName, err = d.getLabelAt(i.Value, "")
		if err != nil {
			panic(err)
		}
		elem.Value = i

		d.removeElemAt(addr + 1)

		// mark both targets of the branch as instructions
		d.markAsInstruction(addr + 2)
		d.markAsInstruction(i.Value)
	case zeroPageAddr:
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i.Type = DirectInstruction
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	case zeroXIndexAddr, zeroYIndexAddr:
		if opCodeInfo.addrMode == zeroYIndexAddr {
			i.RegisterName = "Y"
		} else {
			i.RegisterName = "X"
		}
		v, err := d.elemAsByte(elem.Next())
		if err != nil {
			return err
		}
		i.Type = DirectIndexedInstruction
		i.Payload = []byte{opCode, v}
		i.Value = int(v)
		i.RegisterName = i.RegisterName
		elem.Value = i

		d.removeElemAt(addr + 1)

		// next thing is definitely an instruction
		d.markAsInstruction(addr + 2)
	}
	return nil
}

func (d *Disassembly) ToProgram() *Program {
	// add the org statement
	orgStatement := new(OrgPseudoOp)
	orgStatement.Fill = 0xff // this is the default; causes it to be left off when rendered
	orgStatement.Value = d.offset
	d.prog.List.PushFront(orgStatement)

	return d.prog
}

func (d *Disassembly) readAllAsData() {
	d.offset = 0x10000 - 0x4000*len(d.prog.PrgRom)
	offset := d.offset
	for _, bank := range d.prog.PrgRom {
		for _, b := range bank {
			stmt := new(DataStatement)
			stmt.Type = ByteDataStmt
			stmt.dataList = list.New()
			item := IntegerDataItem(b)
			stmt.dataList.PushBack(&item)
			stmt.Offset = offset
			stmt.Payload = []byte{b}
			d.prog.Offsets[stmt.Offset] = d.prog.List.PushBack(stmt)
			offset += 1
		}
	}
}

func (d *Disassembly) elemAtAddr(addr int) *list.Element {
	elem, ok := d.prog.Offsets[addr]
	if ok {
		return elem
	}
	// if there is only 1 prg rom bank, it is at 0x8000 and mirrored at 0xc000
	if len(d.prog.PrgRom) == 1 && addr < 0xc000 {
		return d.prog.Offsets[addr+0x4000]
	}
	return nil
}

func (d *Disassembly) markAsDataWordLabel(addr int, suggestedName string) error {
	elem1 := d.elemAtAddr(addr)
	elem2 := elem1.Next()
	s1 := elem1.Value.(*DataStatement)
	s2 := elem2.Value.(*DataStatement)
	if s1.dataList.Len() != 1 {
		return errors.New("expected DataList len 1")
	}
	if s2.dataList.Len() != 1 {
		return errors.New("expected DataList len 1")
	}
	n1 := s1.dataList.Front().Value.(*IntegerDataItem)
	n2 := s2.dataList.Front().Value.(*IntegerDataItem)

	newStmt := &DataStatement{
		Type: WordDataStmt,
		Offset: addr,
		Payload: []byte{byte(*n1), byte(*n2)},
		dataList: list.New(),
	}
	targetAddr := int(binary.LittleEndian.Uint16(newStmt.Payload))


	elem1.Value = newStmt
	d.removeElemAt(addr + 1)

	if targetAddr < 0x8000 {
		// target not in PRG ROM
		tmp := IntegerDataItem(targetAddr)
		newStmt.dataList.PushBack(&tmp)
		return nil
	}

	// target in PRG ROM

	err := d.markAsInstruction(targetAddr)
	if err != nil {
		tmp := IntegerDataItem(targetAddr)
		newStmt.dataList.PushBack(&tmp)
		return nil
	}

	labelName, err := d.getLabelAt(targetAddr, suggestedName)
	if err != nil {
		tmp := IntegerDataItem(targetAddr)
		newStmt.dataList.PushBack(&tmp)
		return nil
	}
	newStmt.dataList.PushBack(&LabelCall{labelName})

	return nil
}

func (d *Disassembly) collapseDataStatements() {
	if d.prog.List.Len() < 2 {
		return
	}
	const MAX_DATA_LIST_LEN = 8
	for e := d.prog.List.Front().Next(); e != nil; e = e.Next() {
		dataStmt, ok := e.Value.(*DataStatement)
		if !ok {
			continue
		}
		prev, ok := e.Prev().Value.(*DataStatement)
		if !ok {
			continue
		}
		if prev.dataList.Len()+dataStmt.dataList.Len() > MAX_DATA_LIST_LEN {
			continue
		}
		for de := dataStmt.dataList.Front(); de != nil; de = de.Next() {
			prev.dataList.PushBack(de.Value)
		}
		elToDel := e
		e = e.Prev()
		d.prog.List.Remove(elToDel)

	}
}

func allAscii(dl *list.List) bool {
	for e := dl.Front(); e != nil; e = e.Next() {
		switch t := e.Value.(type) {
		case *IntegerDataItem:
			if *t < 32 || *t > 126 || *t == '"' {
				return false
			}
		case *StringDataItem:
			// nothing to do
		default:
			panic(fmt.Sprintf("unrecognized data list item: %T", e.Value))
		}
	}
	return true
}

func dataListToStr(dl *list.List) string {
	str := ""
	for e := dl.Front(); e != nil; e = e.Next() {
		switch t := e.Value.(type) {
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
			oi.dis.prog.List.Remove(delItem)
		}
		orgStmt := new(OrgPseudoOp)
		orgStmt.Value = firstOffset + oi.repeatCount
		orgStmt.Fill = oi.repeatingByte
		oi.dis.prog.List.InsertBefore(orgStmt, e)
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
	if d.prog.List.Len() < orgMinRepeatAmt {
		return
	}
	orgIdent := new(orgIdentifier)
	orgIdent.dis = d
	for e := d.prog.List.Front().Next(); e != nil; e = e.Next() {
		dataStmt, ok := e.Value.(*DataStatement)
		if !ok || dataStmt.dataList.Len() != 1 {
			orgIdent.stop(e)
			continue
		}
		v, ok := dataStmt.dataList.Front().Value.(*IntegerDataItem)
		if !ok {
			orgIdent.stop(e)
			continue
		}
		orgIdent.gotByte(e, byte(*v))
	}
}

func (d *Disassembly) groupAsciiStrings() {
	const threshold = 5
	if d.prog.List.Len() < threshold {
		return
	}
	e := d.prog.List.Front()
	first := e
	buf := new(bytes.Buffer)
	for e != nil {
		dataStmt, ok := e.Value.(*DataStatement)
		if !ok || dataStmt.Type != ByteDataStmt || !allAscii(dataStmt.dataList) {
			if buf.Len() >= threshold {
				firstStmt := first.Value.(*DataStatement)
				firstStmt.dataList = list.New()
				tmp := StringDataItem(buf.String())
				firstStmt.dataList.PushBack(&tmp)
				for {
					elToDel := first.Next()
					if elToDel == e {
						break
					}
					d.prog.List.Remove(elToDel)
				}
			}
			buf = new(bytes.Buffer)
			e = e.Next()
			first = e
			continue
		}
		buf.WriteString(dataListToStr(dataStmt.dataList))
		e = e.Next()
	}
}

func (d *Disassembly) elemLabelStmt(elem *list.Element) *LabelStatement {
	if elem == nil {
		return nil
	}
	stmt, ok := elem.Value.(*LabelStatement)
	if ok {
		return stmt
	}
	prev := elem.Prev()
	if prev != nil {
		stmt, ok = prev.Value.(*LabelStatement)
		if ok {
			return stmt
		}
	}
	return nil
}

func (d *Disassembly) resolveDynJumpCases() {
	// this function is recursive, since calling markAsDataWordLabel can
	// append more dynJumps
	if len(d.dynJumps) == 0 {
		return
	}
	// use the last item in the dynJumps list, and check a single address
	dynJumpAddr := d.dynJumps[len(d.dynJumps)-1]
	elem := d.elemAtAddr(dynJumpAddr)
	if d.elemLabelStmt(elem) != nil {
		// this dynJump has been exhausted. remove it from the list
		d.dynJumps = d.dynJumps[0 : len(d.dynJumps)-1]
		d.resolveDynJumpCases()
		return
	}
	_, err := d.elemAsWord(elem)
	if err != nil {
		// this dynJump has been exhausted. remove it from the list
		d.dynJumps = d.dynJumps[0 : len(d.dynJumps)-1]
		d.resolveDynJumpCases()
		return
	}
	stmt := elem.Value.(*DataStatement)
	// update the address to the next possible jump point
	d.dynJumps[len(d.dynJumps)-1] = dynJumpAddr + 2
	d.markAsDataWordLabel(stmt.Offset, "")
	d.resolveDynJumpCases()
}

func (r *Rom) Disassemble() (*Program, error) {
	if len(r.PrgRom) != 1 && len(r.PrgRom) != 2 {
		return nil, errors.New("only 1 or 2 prg rom banks supported")
	}

	dis := new(Disassembly)
	dis.jumpTables = make(map[int]bool)
	dis.prog = new(Program)
	dis.prog.List = list.New()
	dis.prog.Offsets = make(map[int]*list.Element)
	dis.prog.Labels = make(map[string]int)
	dis.prog.ChrRom = r.ChrRom
	dis.prog.PrgRom = r.PrgRom

	dis.readAllAsData()

	// use the known entry points to recursively disassemble data statements
	dis.markAsDataWordLabel(0xfffa, "NMI_Routine")
	dis.markAsDataWordLabel(0xfffc, "Reset_Routine")
	dis.markAsDataWordLabel(0xfffe, "IRQ_Routine")

	// go over the dynamic jumps that we found and mark the options as labels
	dis.resolveDynJumpCases()

	dis.identifyOrgs()
	dis.groupAsciiStrings()
	dis.collapseDataStatements()

	p := dis.ToProgram()
	p.ChrRom = r.ChrRom
	p.PrgRom = r.PrgRom
	p.Mirroring = r.Mirroring

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

func (i *Instruction) Render() string {
	switch i.Type {
	case ImmediateInstruction:
		return fmt.Sprintf("%s #$%02x", i.OpName, i.Value)
	case ImpliedInstruction:
		return i.OpName
	case DirectInstruction:
		if opCodeDataMap[i.OpCode].addrMode == zeroPageAddr {
			return fmt.Sprintf("%s $%02x", i.OpName, i.Value)
		}
		return fmt.Sprintf("%s $%04x", i.OpName, i.Value)
	case DirectWithLabelInstruction:
		return fmt.Sprintf("%s %s", i.OpName, i.LabelName)
	case DirectIndexedInstruction:
		addrMode := opCodeDataMap[i.OpCode].addrMode
		if addrMode == zeroXIndexAddr || addrMode == zeroYIndexAddr {
			return fmt.Sprintf("%s $%02x, %s", i.OpName, i.Value, i.RegisterName)
		}
		return fmt.Sprintf("%s $%04x, %s", i.OpName, i.Value, i.RegisterName)
	case DirectWithLabelIndexedInstruction:
		return fmt.Sprintf("%s %s, %s", i.OpName, i.LabelName, i.RegisterName)
	case IndirectInstruction:
		return fmt.Sprintf("%s ($%04x)", i.OpName, i.Value)
	case IndirectXInstruction:
		return fmt.Sprintf("%s ($%02x, X)", i.OpName, i.Value)
	case IndirectYInstruction:
		return fmt.Sprintf("%s ($%02x), Y", i.OpName, i.Value)
	}
	panic("unexpected Instruction Type")
}

func (i *OrgPseudoOp) Render() string {
	if i.Fill == 0xff {
		return fmt.Sprintf(".org $%04x", i.Value)
	}
	return fmt.Sprintf(".org $%04x, $%02x", i.Value, i.Fill)
}

func (s *DataStatement) Render() string {
	buf := new(bytes.Buffer)
	switch s.Type {
	default: panic("unexpected DataStatement Type")
	case ByteDataStmt:
		buf.WriteString(".db ")
	case WordDataStmt:
		buf.WriteString(".dw ")
	}
	for e := s.dataList.Front(); e != nil; e = e.Next() {
		switch t := e.Value.(type) {
		case *LabelCall:
			buf.WriteString(t.LabelName)
		case *StringDataItem:
			buf.WriteString("\"")
			buf.WriteString(string(*t))
			buf.WriteString("\"")
		case *IntegerDataItem:
			switch s.Type {
			default: panic("unexpected DataStatement Type")
			case ByteDataStmt:
				buf.WriteString(fmt.Sprintf("$%02x", int(*t)))
			case WordDataStmt:
				buf.WriteString(fmt.Sprintf("$%04x", int(*t)))
			}
		}
		if e != s.dataList.Back() {
			buf.WriteString(", ")
		}
	}
	return buf.String()
}

func (s *LabelStatement) Render() string {
	return fmt.Sprintf("%s:", s.LabelName)
}

func (p *Program) WriteSource(writer io.Writer) (err error) {
	w := bufio.NewWriter(writer)

	for e := p.List.Front(); e != nil; e = e.Next() {
		switch t := e.Value.(type) {
		default:
			panic(fmt.Sprintf("unrecognized node: %T", e.Value))
		case *Instruction:
			_, err = w.WriteString("    ")
			_, err = w.WriteString(t.Render())
			_, err = w.WriteString("\n")
		case *LabelStatement:
			_, err = w.WriteString(t.Render())
			_, err = w.WriteString("\n")
		case *DataStatement:
			_, err = w.WriteString("    ")
			_, err = w.WriteString(t.Render())
			_, err = w.WriteString("\n")
		case *OrgPseudoOp:
			_, err = w.WriteString(t.Render())
			_, err = w.WriteString("\n")
		}
	}

	w.Flush()
	return
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
