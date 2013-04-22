package main

import (
	"./asm6502"
	"fmt"
	"reflect"
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
	program, err := asm6502.ParseFile("test.6502.asm")
	if err != nil { panic(err) }
	program.Ast(&astPrinter{})
}
