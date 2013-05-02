package main

import (
	"./asm6502"
	"./nes"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

var astFlag bool
var assembleFlag bool
var disassembleFlag bool
var unRomFlag bool
var compileFlag bool
var romFlag bool
var disableOptFlag, dumpFlag, dumpPreFlag bool

func init() {
	flag.BoolVar(&astFlag, "ast", false, "Print the abstract syntax tree and quit")
	flag.BoolVar(&assembleFlag, "asm", false, "Assemble into 6502 machine code")
	flag.BoolVar(&disassembleFlag, "dis", false, "Disassemble 6502 machine code")
	flag.BoolVar(&romFlag, "rom", false, "Assemble a jam package into an NES ROM")
	flag.BoolVar(&unRomFlag, "unrom", false, "Disassemble an NES ROM into a jam package")
	flag.BoolVar(&compileFlag, "c", false, "Compile into a native executable")
	flag.BoolVar(&disableOptFlag, "O0", false, "Disable optimizations")
	flag.BoolVar(&dumpFlag, "d", false, "Dump LLVM IR code for generated code")
	flag.BoolVar(&dumpPreFlag, "dd", false, "Dump LLVM IR code for generated code before verifying module")
}

func usageAndQuit() {
	fmt.Printf("Usage: %s [options] inputfile [outputfile]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func removeExtension(filename string) string {
	return filename[0 : len(filename)-len(path.Ext(filename))]
}

func compile(filename string, program *asm6502.Program) {
	outfile := removeExtension(filename) + ".bc"
	if flag.NArg() == 2 {
		outfile = flag.Arg(1)
	}
	fmt.Printf("Compiling to %s\n", outfile)
	var flags asm6502.CompileFlags
	if disableOptFlag {
		flags |= asm6502.DisableOptFlag
	}
	if dumpFlag {
		flags |= asm6502.DumpModuleFlag
	}
	if dumpPreFlag {
		flags |= asm6502.DumpModulePreFlag
	}
	c := program.Compile(outfile, flags)
	if len(c.Errors) != 0 {
		fmt.Fprintf(os.Stderr, "Errors:\n%s\n", strings.Join(c.Errors, "\n"))
		return
	}
	if len(c.Warnings) != 0 {
		fmt.Fprintf(os.Stderr, "Warnings:\n%s\n", strings.Join(c.Warnings, "\n"))
	}
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 && flag.NArg() != 2 {
		usageAndQuit()
	}
	filename := flag.Arg(0)
	if astFlag || assembleFlag {
		fmt.Printf("Parsing %s\n", filename)
		programAst, err := asm6502.ParseFile(filename)
		if err != nil {
			panic(err)
		}
		if astFlag {
			programAst.Print()
		}
		if !assembleFlag && !compileFlag {
			return
		}
		fmt.Printf("Assembling %s\n", filename)
		program := programAst.ToProgram()
		if len(program.Errors) > 0 {
			for _, err := range program.Errors {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		if compileFlag {
			compile(filename, program)
			return
		}
		if assembleFlag {
			outfile := removeExtension(filename) + ".bin"
			if flag.NArg() == 2 {
				outfile = flag.Arg(1)
			}
			fmt.Printf("Writing to %s\n", outfile)
			err = program.AssembleToFile(outfile)
			if err != nil {
				panic(err)
			}
		}
		return
	} else if unRomFlag {
		fmt.Printf("loading %s\n", filename)
		rom, err := nes.LoadFile(filename)
		if err != nil {
			panic(err)
		}
		outdir := removeExtension(filename)
		if flag.NArg() == 2 {
			outdir = flag.Arg(1)
		}
		fmt.Printf("disassembling to %s\n", outdir)
		err = rom.DisassembleToDir(outdir)
		if err != nil {
			panic(err)
		}
		return
	} else if disassembleFlag {
		fmt.Printf("disassembling %s\n", filename)
		p, err := asm6502.DisassembleFile(filename)
		if err != nil {
			panic(err)
		}
		if compileFlag {
			compile(filename, p)
			return
		}
		if disassembleFlag {
			outfile := removeExtension(filename) + ".asm"
			if flag.NArg() == 2 {
				outfile = flag.Arg(1)
			}
			fmt.Printf("writing source %s\n", outfile)
			err = p.WriteSourceFile(outfile)
			if err != nil {
				panic(err)
			}
		}
		return
	} else if romFlag {
		fmt.Printf("building rom from %s\n", filename)
		r, err := nes.AssembleFile(filename)
		if err != nil {
			panic(err)
		}
		fmt.Printf("saving rom %s\n", r.Filename)
		err = r.SaveFile(path.Dir(filename))
		if err != nil {
			panic(err)
		}
		return
	}
	usageAndQuit()
}
