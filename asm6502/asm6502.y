%{
package asm6502

import "fmt"

type Node interface {
	Ast(v Visitor)
}

type Visitor interface {
	Visit(n Node)
	VisitEnd(n Node)
}

type Program struct {
	statements * StatementList
}

func (p Program) Ast(v Visitor) {
	v.Visit(p)
	p.statements.Ast(v)
	v.VisitEnd(p)
}

type StatementList []Node

func (sl StatementList) Ast(v Visitor) {
	v.Visit(sl)
	for i := len(sl) - 1; i >= 0; i-- {
		sl[i].Ast(v)
	}
	v.VisitEnd(sl)
}

type AssignStatement struct {
	VarName string
	Value int
}

func (as AssignStatement) Ast(v Visitor) {
	v.Visit(as)
	v.VisitEnd(as)
}

type ImmediateInstruction struct {
	OpName string
	Value int
}

func (ii ImmediateInstruction) Ast(v Visitor) {
	v.Visit(ii)
	v.VisitEnd(ii)
}

type ImpliedInstruction struct {
	OpName string
}

func (ii ImpliedInstruction) Ast(v Visitor) {
	v.Visit(ii)
	v.VisitEnd(ii)
}

type LabelStatement struct {
	LabelName string
}

func (ls LabelStatement) Ast(v Visitor) {
	v.Visit(ls)
	v.VisitEnd(ls)
}

type AbsoluteWithLabelIndexedInstruction struct {
	OpName string
	LabelName string
	RegisterName string
}

func (n AbsoluteWithLabelIndexedInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

type AbsoluteWithLabelInstruction struct {
	OpName string
	LabelName string
}

func (n AbsoluteWithLabelInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

var program *Program
%}

%union {
	integer int
	identifier string
	statementList StatementList
	statement Node
	instructionStatement Node
	labelStatement LabelStatement
	assignStatement AssignStatement
	program Program
}

%type <statementList> statementList
%type <assignStatement> assignStatement
%type <statement> statement
%type <instructionStatement> instructionStatement
%type <labelStatement> labelStatement
%type <program> program

%token <identifier> tokIdentifier
%token <integer> tokInteger
%token tokEqual
%token tokPound
%token tokColon
%token tokComma
%token tokNewline

%%

program : statementList {
	program = &Program{&$1}
}

statementList : statement statementList {
	if $1 != nil {
		$$ = append($2, $1)
	} else {
		$$ = $2
	}
} | statement {
	$$ = []Node{$1}
}

statement : assignStatement tokNewline {
	$$ = $1
} | instructionStatement tokNewline {
	$$ = $1
} | labelStatement {
	$$ = $1
} | tokNewline {
	// empty statement
	$$ = nil
}

labelStatement : tokIdentifier tokColon {
	$$ = LabelStatement{$1}
}

assignStatement : tokIdentifier tokEqual tokInteger {
	$$ = AssignStatement{$1, $3}
}

instructionStatement : tokIdentifier tokPound tokInteger {
	// immediate address
	$$ = ImmediateInstruction{$1, $3}
} | tokIdentifier {
	// no address
	$$ = ImpliedInstruction{$1}
} | tokIdentifier tokIdentifier tokComma tokIdentifier {
	$$ = AbsoluteWithLabelIndexedInstruction{$1, $2, $4}
} | tokIdentifier tokIdentifier {
	$$ = AbsoluteWithLabelInstruction{$1, $2}
}

%%

