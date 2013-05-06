package jamulator

import (
	"fmt"
	"reflect"
)

type astPrinter struct {
	indentLevel int
}

func (ap *astPrinter) Visit(n Node) {
	for i := 0; i < ap.indentLevel; i++ {
		fmt.Print(" ")
	}
	fmt.Println(reflect.TypeOf(n))
	ap.indentLevel += 2
}

func (ap *astPrinter) VisitEnd(n Node) {
	ap.indentLevel -= 2
}

func (p ProgramAST) Print() {
	p.Ast(&astPrinter{})
}
