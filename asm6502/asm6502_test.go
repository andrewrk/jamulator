package asm6502

import (
	"testing"
	"io/ioutil"
	"bytes"
)

func TestSuite6502Asm(t *testing.T) {
	expected, err := ioutil.ReadFile("test/suite6502.bin.ref")
	if err != nil { t.Error(err) }
	programAst, err := ParseFile("test/suite6502.asm")
	if err != nil { t.Error(err) }
	program := programAst.ToProgram()
	if len(program.Errors) > 0 { t.Error("unexpected errors") }
	buf := new(bytes.Buffer)
	program.Assemble(buf)
	if bytes.Compare(buf.Bytes(), expected) != 0 {
		t.Error("does not match expected output")
	}
}
