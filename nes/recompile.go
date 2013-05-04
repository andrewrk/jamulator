package nes

import (
	"../asm6502"
	"bytes"
	"fmt"
	"path"
	"errors"
	"os"
	"os/exec"
	"io/ioutil"
	"strings"
)

func (rom *Rom) RecompileToBinary(filename string, flags asm6502.CompileFlags) error {
	if len(rom.PrgRom) != 1 {
		return errors.New("only roms with 1 prg rom bank are supported")
	}
	fmt.Fprintf(os.Stderr, "Disassembling...\n")
	buf := bytes.NewBuffer(rom.PrgRom[0])
	program, err := asm6502.Disassemble(buf)
	if err != nil {
		return err
	}
	program.ChrRom = rom.ChrRom
	program.Mirroring = rom.Mirroring

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()
	runtimeArchive := "runtime/runtime.a"
	tmpPrgBitcode := path.Join(tmpDir, "prg.bc")
	tmpPrgObject := path.Join(tmpDir, "prg.o")

	fmt.Fprintf(os.Stderr, "Decompiling...\n")
	c, err := program.CompileToFilename(tmpPrgBitcode, flags)
	if err != nil {
		return err
	}
	if len(c.Errors) != 0 {
		return errors.New(strings.Join(c.Errors, "\n"))
	}
	if len(c.Warnings) != 0 {
		fmt.Fprintf(os.Stderr, "Warnings:\n%s\n", strings.Join(c.Warnings, "\n"))
	}
	fmt.Fprintf(os.Stderr, "Compiling...\n")
	out, err := exec.Command("llc", "-o", tmpPrgObject, "-filetype=obj", tmpPrgBitcode).CombinedOutput()
	fmt.Fprint(os.Stderr, string(out))
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Linking...\n")
	out, err = exec.Command("gcc", "-o", filename, tmpPrgObject, runtimeArchive).CombinedOutput()
	fmt.Fprint(os.Stderr, string(out))
	if err != nil {
		return err
	}

	return nil
}
