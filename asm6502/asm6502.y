%{
package asm6502

import "fmt"

type Node interface {
	Ast(v Visitor)
}

// an InstructionStatement is also a Node
type InstructionStatement interface {
	Ast(v Visitor)
	OpName() string
}

type Visitor interface {
	Visit(n Node)
	VisitEnd(n Node)
}

type ProgramAST struct {
	statements StatementList
}

func (p ProgramAST) Ast(v Visitor) {
	v.Visit(p)
	p.statements.Ast(v)
	v.VisitEnd(p)
}

type StatementList []Node

func (sl StatementList) Ast(v Visitor) {
	v.Visit(sl)
	for _, s := range(sl) {
		s.Ast(v)
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
	opName string
	Value int
	Line int
}

func (ii ImmediateInstruction) Ast(v Visitor) {
	v.Visit(ii)
	v.VisitEnd(ii)
}

func (ii ImmediateInstruction) OpName() string {
	return ii.opName
}

type ImpliedInstruction struct {
	opName string
	Line int
}

func (ii ImpliedInstruction) Ast(v Visitor) {
	v.Visit(ii)
	v.VisitEnd(ii)
}

func (ii ImpliedInstruction) OpName() string {
	return ii.opName
}

type LabelStatement struct {
	LabelName string
}

func (ls LabelStatement) Ast(v Visitor) {
	v.Visit(ls)
	v.VisitEnd(ls)
}

type AbsoluteWithLabelIndexedInstruction struct {
	opName string
	LabelName string
	RegisterName string
	Line int
}

func (n AbsoluteWithLabelIndexedInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n AbsoluteWithLabelIndexedInstruction) OpName() string {
	return n.opName
}

type AccumulatorInstruction struct {
	opName string
	Line int
}

func (n AccumulatorInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n AccumulatorInstruction) OpName() string {
	return n.opName
}

type DirectWithLabelInstruction struct {
	opName string
	LabelName string
	Line int
}

func (n DirectWithLabelInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n DirectWithLabelInstruction) OpName() string {
	return n.opName
}

type DataStatement struct {
	dataList DataList
}

func (n DataStatement) Ast(v Visitor) {
	v.Visit(n)
	n.dataList.Ast(v)
	v.VisitEnd(n)
}

type DataList []Node

func (n DataList) Ast(v Visitor) {
	v.Visit(n)
	for _, di := range(n) {
		di.Ast(v)
	}
	v.VisitEnd(n)
}

type StringDataItem string

func (n StringDataItem) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

type IntegerDataItem int

func (n IntegerDataItem) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

var programAst *ProgramAST
%}

%union {
	integer int
	identifier string
	quotedString string
	statementList StatementList
	statement Node
	instructionStatement InstructionStatement
	labelStatement LabelStatement
	assignStatement AssignStatement
	dataStatement DataStatement
	dataList DataList
	dataItem Node
	programAst ProgramAST
}

%type <statementList> statementList
%type <assignStatement> assignStatement
%type <statement> statement
%type <instructionStatement> instructionStatement
%type <labelStatement> labelStatement
%type <dataStatement> dataStatement
%type <dataList> dataList
%type <dataItem> dataItem
%type <programAst> programAst

%token <identifier> tokIdentifier
%token <integer> tokInteger
%token <quotedString> tokQuotedString
%token tokEqual
%token tokPound
%token tokColon
%token tokComma
%token tokNewline
%token tokData

%%

programAst : statementList {
	programAst = &ProgramAST{$1}
}

statementList : statementList statement {
	if $2 == nil {
		$$ = $1
	} else {
		$$ = append($1, $2)
	}
} | statement {
	if $1 == nil {
		$$ = []Node{}
	} else {
		$$ = []Node{$1}
	}
}

statement : assignStatement tokNewline {
	$$ = $1
} | instructionStatement tokNewline {
	$$ = $1
} | labelStatement {
	$$ = $1
} | dataStatement tokNewline {
	$$ = $1
} | tokNewline {
	// empty statement
	$$ = nil
}

dataStatement : tokData dataList {
	$$ = DataStatement{$2}
}

dataList : dataList tokComma dataItem {
	$$ = append($1, $3)
} | dataItem {
	$$ = []Node{$1}
}

dataItem : tokQuotedString {
	$$ = StringDataItem($1)
} | tokInteger {
	$$ = IntegerDataItem($1)
}

labelStatement : tokIdentifier tokColon {
	$$ = LabelStatement{$1}
}

assignStatement : tokIdentifier tokEqual tokInteger {
	$$ = AssignStatement{$1, $3}
}

instructionStatement : tokIdentifier tokPound tokInteger {
	// immediate address
	$$ = ImmediateInstruction{$1, $3, parseLineNumber}
} | tokIdentifier {
	// no address
	$$ = ImpliedInstruction{$1, parseLineNumber}
} | tokIdentifier tokIdentifier tokComma tokIdentifier {
	$$ = AbsoluteWithLabelIndexedInstruction{$1, $2, $4, parseLineNumber}
} | tokIdentifier tokIdentifier {
	if $2 == "a" || $2 == "A" {
		$$ = AccumulatorInstruction{$1, parseLineNumber}
	} else {
		$$ = DirectWithLabelInstruction{$1, $2, parseLineNumber}
	}
}

%%

