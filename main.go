package main

import (
	"./asm6502"
	"fmt"
	"os"
	"flag"
)

var astFlag bool
func init() {
	flag.BoolVar(&astFlag, "ast", false, "Print the abstract syntax tree and quit")
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Printf("Usage: %s [options] code.asm\n", os.Args[0])
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
	program.Compile(filename + ".bc")
}
