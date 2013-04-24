package main

import (
	"./asm6502"
	"fmt"
	"os"
	"flag"
)

var astFlag bool
var assembleFlag bool
func init() {
	flag.BoolVar(&astFlag, "ast", false, "Print the abstract syntax tree and quit")
	flag.BoolVar(&assembleFlag, "asm", false, "Assemble the input into machine code")
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 && flag.NArg() != 2 {
		fmt.Printf("Usage: %s [options] inputfile [outputfile]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	filename := flag.Arg(0)
	programAst, err := asm6502.ParseFile(filename)
	if err != nil { panic(err) }
	if astFlag {
		programAst.Print()
		os.Exit(0)
	}
	program := programAst.ToProgram()
	if len(program.Errors) > 0 {
		for _, err := range(program.Errors) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
	if assembleFlag {
		outfile := filename + ".bin"
		if flag.NArg() == 2 {
			outfile = flag.Arg(1)
		}
		program.Assemble(outfile)
	} else {
		outfile := filename + ".bc"
		if flag.NArg() == 2 {
			outfile = flag.Arg(1)
		}
		program.Compile(outfile)
	}
}
