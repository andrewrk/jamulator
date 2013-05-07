package main

import (
	"./jamulator"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

var (
	astFlag         bool
	assembleFlag    bool
	disassembleFlag bool
	unRomFlag       bool
	compileFlag     bool
	romFlag         bool
	disableOptFlag  bool
	dumpFlag        bool
	dumpPreFlag     bool
	debugFlag       bool
	recompileFlag   bool
)

// TODO: change this to use commands
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
	flag.BoolVar(&debugFlag, "g", false, "Include debug print statements in generated code")
	flag.BoolVar(&recompileFlag, "recompile", false, "Recompile an NES ROM into a native binary")
}

func usageAndQuit() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] inputfile [outputfile]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func removeExtension(filename string) string {
	return filename[0 : len(filename)-len(path.Ext(filename))]
}

func compileFlags() (flags jamulator.CompileFlags) {
	if disableOptFlag {
		flags |= jamulator.DisableOptFlag
	}
	if dumpFlag {
		flags |= jamulator.DumpModuleFlag
	}
	if dumpPreFlag {
		flags |= jamulator.DumpModulePreFlag
	}
	if debugFlag {
		flags |= jamulator.IncludeDebugFlag
	}
	return
}

func compile(filename string, program *jamulator.Program) {
	outfile := removeExtension(filename) + ".bc"
	if flag.NArg() == 2 {
		outfile = flag.Arg(1)
	}
	fmt.Fprintf(os.Stderr, "Compiling to %s\n", outfile)
	c, err := program.CompileToFilename(outfile, compileFlags())
	if err != nil {
		panic(err)
	}
	if len(c.Errors) != 0 {
		fmt.Fprintf(os.Stderr, "Errors:\n%s\n", strings.Join(c.Errors, "\n"))
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Parsing %s\n", filename)
		programAst, err := jamulator.ParseFile(filename)
		if err != nil {
			panic(err)
		}
		if astFlag {
			programAst.Print()
		}
		if !assembleFlag && !compileFlag {
			return
		}
		fmt.Fprintf(os.Stderr, "Assembling %s\n", filename)
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
			fmt.Fprintf(os.Stderr, "Writing to %s\n", outfile)
			err = program.AssembleToFile(outfile)
			if err != nil {
				panic(err)
			}
		}
		return
	} else if unRomFlag || recompileFlag {
		fmt.Fprintf(os.Stderr, "loading %s\n", filename)
		rom, err := jamulator.LoadFile(filename)
		if err != nil {
			panic(err)
		}
		if unRomFlag {
			outdir := removeExtension(filename)
			if flag.NArg() == 2 {
				outdir = flag.Arg(1)
			}
			fmt.Fprintf(os.Stderr, "disassembling to %s\n", outdir)
			err = rom.DisassembleToDir(outdir)
			if err != nil {
				panic(err)
			}
			return
		}
		// recompile to native binary
		outfile := removeExtension(filename)
		if flag.NArg() == 2 {
			outfile = flag.Arg(1)
		}
		err = rom.RecompileToBinary(outfile, compileFlags())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}
		return
	} else if disassembleFlag {
		fmt.Fprintf(os.Stderr, "disassembling %s\n", filename)
		p, err := jamulator.DisassembleFile(filename)
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
			fmt.Fprintf(os.Stderr, "writing source %s\n", outfile)
			err = p.WriteSourceFile(outfile)
			if err != nil {
				panic(err)
			}
		}
		return
	} else if romFlag {
		fmt.Fprintf(os.Stderr, "building rom from %s\n", filename)
		r, err := jamulator.AssembleRomFile(filename)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "saving rom %s\n", r.Filename)
		err = r.SaveFile(path.Dir(filename))
		if err != nil {
			panic(err)
		}
		return
	}
	usageAndQuit()
}
