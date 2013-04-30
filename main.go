package main

import (
	"./asm6502"
	"./nes"
	"fmt"
	"os"
	"flag"
	"path"
)

var astFlag bool
var assembleFlag bool
var disassembleFlag bool
var unRomFlag bool
var compileFlag bool
var romFlag bool
func init() {
	flag.BoolVar(&astFlag, "ast", false, "Print the abstract syntax tree and quit")
	flag.BoolVar(&assembleFlag, "asm", false, "Assemble into 6502 machine code")
	flag.BoolVar(&disassembleFlag, "dis", false, "Disassemble 6502 machine code")
	flag.BoolVar(&romFlag, "rom", false, "Assemble a jam package into an NES ROM")
	flag.BoolVar(&unRomFlag, "unrom", false, "Disassemble an NES ROM into a jam package")
	flag.BoolVar(&compileFlag, "c", false, "Compile a jam package into a native executable")
}

func usageAndQuit() {
	fmt.Printf("Usage: %s [options] inputfile [outputfile]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func removeExtension(filename string) string {
	return filename[0:len(filename)-len(path.Ext(filename))]
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 && flag.NArg() != 2 { usageAndQuit() }
	filename := flag.Arg(0)
	if astFlag || assembleFlag || compileFlag {
		fmt.Printf("Parsing %s\n", filename)
		programAst, err := asm6502.ParseFile(filename)
		if err != nil { panic(err) }
		if astFlag { programAst.Print() }
		if !assembleFlag && !compileFlag { return }
		fmt.Printf("Assembling %s\n", filename)
		program := programAst.ToProgram()
		if len(program.Errors) > 0 {
			for _, err := range(program.Errors) {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		if compileFlag {
			outfile := removeExtension(filename) + ".bc"
			if flag.NArg() == 2 {
				outfile = flag.Arg(1)
			}
			fmt.Printf("Compiling to %s\n", outfile)
			err := program.Compile(outfile)
			if err != nil { panic(err) }
		}
		if assembleFlag {
			outfile := removeExtension(filename) + ".bin"
			if flag.NArg() == 2 {
				outfile = flag.Arg(1)
			}
			fmt.Printf("Writing to %s\n", outfile)
			err = program.AssembleToFile(outfile)
			if err != nil { panic(err) }
		}
		return
	} else if unRomFlag {
		fmt.Printf("loading %s\n", filename)
		rom, err := nes.LoadFile(filename)
		if err != nil { panic(err) }
		outdir := removeExtension(filename)
		if flag.NArg() == 2 {
			outdir = flag.Arg(1)
		}
		fmt.Printf("disassembling to %s\n", outdir)
		err = rom.DisassembleToDir(outdir)
		if err != nil { panic(err) }
		return
	} else if disassembleFlag {
		fmt.Printf("disassembling %s\n", filename)
		p, err := asm6502.DisassembleFile(filename)
		if err != nil { panic(err) }
		outfile := removeExtension(filename) + ".asm"
		if flag.NArg() == 2 {
			outfile = flag.Arg(1)
		}
		fmt.Printf("writing source %s\n", outfile)
		err = p.WriteSourceFile(outfile)
		if err != nil { panic(err) }
		return
	} else if romFlag {
		fmt.Printf("building rom from %s\n", filename)
		r, err := nes.AssembleFile(filename)
		if err != nil { panic(err) }
		fmt.Printf("saving rom %s\n", r.Filename)
		err = r.SaveFile(path.Dir(filename))
		if err != nil { panic(err) }
		return
	}
	usageAndQuit()
}
