package asm6502

import (
	"io"
	"os"
	"bufio"
)

func Disassemble(reader io.Reader) (*Program, error) {
	p := new(Program)

	// TODO: actually disassemble
	//r := bufio.NewReader(reader)

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

func (p *Program) WriteSource(writer io.Writer) error {
	w := bufio.NewWriter(writer)

	// TODO: actually print source
	_, err := w.WriteString("source\n")
	if err != nil { return err }

	w.Flush()
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
