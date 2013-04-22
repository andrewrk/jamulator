package main

import (
	"./asm6502"
	"fmt"
	"reflect"
	"os"
)

type astPrinter struct {
	indentLevel int
}

func (ap *astPrinter) Visit(n asm6502.Node) {
	for i := 0; i < ap.indentLevel; i++ {
		fmt.Print(" ")
	}
	fmt.Println(reflect.TypeOf(n))
	ap.indentLevel += 2
}

func (ap *astPrinter) VisitEnd(n asm6502.Node) {
	ap.indentLevel -= 2
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	filename := os.Args[1]
	program, err := asm6502.ParseFile(filename)
	if err != nil { panic(err) }
	program.Ast(&astPrinter{})
	err = program.Compile("a.out")
	if err != nil { panic(err) }
}

func printUsage() {
	fmt.Println("Usage: jamulator code.asm")
}
