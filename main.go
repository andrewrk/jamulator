package main

import (
	"./asm6502"
	"./nes"
	"fmt"
	"os"
	"flag"
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

func main() {
	flag.Parse()
	if flag.NArg() != 1 && flag.NArg() != 2 { usageAndQuit() }
	filename := flag.Arg(0)
	if astFlag || assembleFlag || compileFlag {
		programAst, err := asm6502.ParseFile(filename)
		if err != nil { panic(err) }
		if astFlag { programAst.Print() }
		if !assembleFlag && !compileFlag { return }
		program := programAst.ToProgram()
		if len(program.Errors) > 0 {
			for _, err := range(program.Errors) {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		if compileFlag {
			outfile := filename + ".bc"
			if flag.NArg() == 2 {
				outfile = flag.Arg(1)
			}
			err := program.Compile(outfile)
			if err != nil { panic(err) }
		}
		outfile := filename + ".bin"
		if flag.NArg() == 2 {
			outfile = flag.Arg(1)
		}
		err = program.AssembleToFile(outfile)
		if err != nil { panic(err) }
		return
	} else if unRomFlag {
		rom, err := nes.DisassembleFile(filename)
		if err != nil { panic(err) }
		fmt.Println(rom.String())
		return
	} else if disassembleFlag {
		panic("disasseble not yet supported")
	} else if romFlag {
		panic("rom assembly not yet supported")
	}
	usageAndQuit()
}
