package jamulator

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

func (rom *Rom) RecompileToBinary(filename string, flags CompileFlags) error {
	if len(rom.PrgRom) != 1 && len(rom.PrgRom) != 2 {
		return errors.New("only roms with 1-2 prg rom banks are supported")
	}
	fmt.Fprintf(os.Stderr, "Disassembling...\n")
	program, err := rom.Disassemble()
	if err != nil {
		return err
	}
	if len(program.Errors) > 0 {
		return errors.New(strings.Join(program.Errors, "\n"))
	}

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
	out, err := exec.Command("llc", "-o", tmpPrgObject, "-filetype=obj", "-relocation-model=pic", tmpPrgBitcode).CombinedOutput()
	fmt.Fprint(os.Stderr, string(out))
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Linking...\n")
	out, err = exec.Command("gcc", tmpPrgObject, runtimeArchive, "-lGLEW", "-lGL", "-lSDL", "-lSDL_gfx", "-o", filename).CombinedOutput()
	fmt.Fprint(os.Stderr, string(out))
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Done: %s\n", filename)

	return nil
}
