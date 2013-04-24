%{
package asm6502

import (
	"fmt"
	"strconv"
	"strings"
)

type Node interface {
	Ast(v Visitor)
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
	OpName string
	Value int
	Line int

	// filled in later
	OpCode byte
	Size int
}

func (ii ImmediateInstruction) Ast(v Visitor) {
	v.Visit(ii)
	v.VisitEnd(ii)
}

func (ii ImmediateInstruction) GetSize() int {
	return ii.Size
}

type ImpliedInstruction struct {
	OpName string
	Line int

	// filled in later
	OpCode byte
	Size int
}

func (ii ImpliedInstruction) Ast(v Visitor) {
	v.Visit(ii)
	v.VisitEnd(ii)
}

func (ii ImpliedInstruction) GetSize() int {
	return ii.Size
}

type LabelStatement struct {
	LabelName string
	Line int
}

func (ls LabelStatement) Ast(v Visitor) {
	v.Visit(ls)
	v.VisitEnd(ls)
}

type DirectWithLabelIndexedInstruction struct {
	OpName string
	LabelName string
	RegisterName string
	Line int

	// filled in later
	OpCode byte
	Size int
	Offset uint16
}

func (n DirectWithLabelIndexedInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n DirectWithLabelIndexedInstruction) GetSize() int {
	return n.Size
}

type DirectIndexedInstruction struct {
	OpName string
	Value int
	RegisterName string
	Line int

	// filled in later
	Payload []byte
}

func (n DirectIndexedInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n DirectIndexedInstruction) GetSize() int {
	return len(n.Payload)
}

type DirectWithLabelInstruction struct {
	OpName string
	LabelName string
	Line int

	OpCode byte
	Size int
	Offset uint16
}

func (n DirectWithLabelInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n DirectWithLabelInstruction) GetSize() int {
	return n.Size
}

type DirectInstruction struct {
	OpName string
	Value int
	Line int

	Payload []byte
}

func (n DirectInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n DirectInstruction) GetSize() int {
	return len(n.Payload)
}

type IndirectXInstruction struct {
	OpName string
	Value int
	Line int

	Payload []byte
}

func (n IndirectXInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n IndirectXInstruction) GetSize() int {
	return len(n.Payload)
}

type IndirectYInstruction struct {
	OpName string
	Value int
	Line int

	OpCode byte
	Size int
}

func (n IndirectYInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n IndirectYInstruction) GetSize() int {
	return n.Size
}

type IndirectInstruction struct {
	OpName string
	Value int
	Line int

	Payload []byte
}

func (n IndirectInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n IndirectInstruction) GetSize() int {
	return len(n.Payload)
}


type OrgPseudoOp struct {
	Value int
	Line int
}

func (n OrgPseudoOp) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

type DataStatement struct {
	dataList DataList

	// filled in later
	Size int
}

func (n DataStatement) Ast(v Visitor) {
	v.Visit(n)
	n.dataList.Ast(v)
	v.VisitEnd(n)
}

func (n DataStatement) GetSize() int {
	return n.Size
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
	instructionStatement Node
	labelStatement LabelStatement
	assignStatement AssignStatement
	dataStatement DataStatement
	dataList DataList
	dataItem Node
	programAst ProgramAST
	processorDecl string
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
%type <processorDecl> processorDecl

%token <identifier> tokIdentifier
%token <integer> tokInteger
%token <quotedString> tokQuotedString
%token tokEqual
%token tokPound
%token tokColon
%token tokComma
%token tokNewline
%token tokData
%token tokProcessor
%token tokLParen
%token tokRParen

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
} | processorDecl tokNewline {
	if $1 != "6502" {
		yylex.Error("Unsupported processor: " + $1 + " - Only 6502 is supported.")
	}
	// empty statement
	$$ = nil
} | tokNewline {
	// empty statement
	$$ = nil
}

processorDecl : tokProcessor tokInteger {
	$$ = strconv.FormatInt(int64($2), 10)
} | tokProcessor tokIdentifier {
	$$ = $2
}

dataStatement : tokData dataList {
	$$ = DataStatement{$2, 0}
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
	$$ = LabelStatement{$1, parseLineNumber}
}

assignStatement : tokIdentifier tokEqual tokInteger {
	$$ = AssignStatement{$1, $3}
}

instructionStatement : tokIdentifier tokPound tokInteger {
	// immediate address
	$$ = ImmediateInstruction{$1, $3, parseLineNumber, 0, 0}
} | tokIdentifier {
	// no address
	$$ = ImpliedInstruction{$1, parseLineNumber, 0, 0}
} | tokIdentifier tokIdentifier tokComma tokIdentifier {
	$$ = DirectWithLabelIndexedInstruction{$1, $2, $4, parseLineNumber, 0, 0, 0}
} | tokIdentifier tokInteger tokComma tokIdentifier {
	$$ = DirectIndexedInstruction{$1, $2, $4, parseLineNumber, []byte{}}
} | tokIdentifier tokIdentifier {
	if $2 == "a" || $2 == "A" {
		$$ = ImpliedInstruction{$1, parseLineNumber, 0, 0}
	} else {
		$$ = DirectWithLabelInstruction{$1, $2, parseLineNumber, 0, 0, 0}
	}
} | tokIdentifier tokInteger {
	switch strings.ToLower($1) {
	case "org":
		$$ = OrgPseudoOp{$2, parseLineNumber}
	default:
		$$ = DirectInstruction{$1, $2, parseLineNumber, []byte{}}
	}
} | tokIdentifier tokLParen tokInteger tokComma tokIdentifier tokRParen {
	if $5 != "x" && $5 != "X" {
		yylex.Error("Register argument must be X.")
	}
	$$ = IndirectXInstruction{$1, $3, parseLineNumber, []byte{}}
} | tokIdentifier tokLParen tokInteger tokRParen tokComma tokIdentifier {
	if $6 != "y" && $6 != "Y" {
		yylex.Error("Register argument must be Y.")
	}
	$$ = IndirectYInstruction{$1, $3, parseLineNumber, 0, 0}
} | tokIdentifier tokLParen tokInteger tokRParen {
	$$ = IndirectInstruction{$1, $3, parseLineNumber, []byte{}}
}

%%

