package asm6502

import (
	"testing"
	"io/ioutil"
	"bytes"
	"fmt"
)

type testAsm struct {
	inFile string
	expectedOutFile string
}

var testAsmList = []testAsm{
	{
		"test/suite6502.asm",
		"test/suite6502.bin.ref",
	},
	{
		"test/zelda.asm",
		"test/zelda.bin.ref",
	},
}

func TestAsm(t *testing.T) {
	for _, ta := range(testAsmList) {
		expected, err := ioutil.ReadFile(ta.expectedOutFile)
		if err != nil { t.Error(err) }
		programAst, err := ParseFile(ta.inFile)
		if err != nil { t.Error(err) }
		program := programAst.ToProgram()
		if len(program.Errors) > 0 { t.Error(fmt.Sprintf("%s: unexpected errors", ta.inFile)) }
		buf := new(bytes.Buffer)
		program.Assemble(buf)
		if bytes.Compare(buf.Bytes(), expected) != 0 {
			t.Error(fmt.Sprintf("%s: does not match expected output", ta.inFile))
		}
	}
}
