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

type StatementList struct {
	This Node
	Next * StatementList
}

func (sl StatementList) Ast(v Visitor) {
	v.Visit(sl)
	sl.This.Ast(v)
	if sl.Next != nil {
		sl.Next.Ast(v)
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

var program *Program
%}

%union {
	integer int
	identifier string
	statementList StatementList
	statement Node
	instructionStatement Node
	assignStatement AssignStatement
	program Program
}

%type <statementList> statementList
%type <assignStatement> assignStatement
%type <statement> statement
%type <instructionStatement> instructionStatement
%type <program> program

%token <identifier> tokIdentifier
%token <integer> tokInteger
%token tokEqual
%token tokPound

%%

program : statementList {
	program = &Program{&$1}
}

statementList : statement statementList {
	$$ = StatementList{$1, &$2}
} | statement {
	$$ = StatementList{$1, nil}
}

statement : assignStatement {
	$$ = $1
} | instructionStatement {
	$$ = $1
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
}

%%

