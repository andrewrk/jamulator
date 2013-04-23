package asm6502

//import "fmt"

type Program struct {
	Variables map[string] int
}


func (p *Program) Visit(n Node) {
	switch as := n.(type) {
	case AssignStatement:
		p.Variables[as.VarName] = as.Value
	}
}

func (p *Program) VisitEnd(n Node) {}

func NewProgram() *Program {
	p := Program{
		map[string]int {},
	}
	return &p
}

func (ast *ProgramAST) ToProgram() (*Program, error) {
	p := NewProgram()
	ast.Ast(p)
	return p, nil
}
