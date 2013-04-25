%{
package asm6502

import (
	"fmt"
	"strconv"
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

func (p *ProgramAST) Ast(v Visitor) {
	v.Visit(p)
	p.statements.Ast(v)
	v.VisitEnd(p)
}

type StatementList []Node

func (sl *StatementList) Ast(v Visitor) {
	v.Visit(sl)
	for _, s := range(*sl) {
		s.Ast(v)
	}
	v.VisitEnd(sl)
}

type AssignStatement struct {
	VarName string
	Value int
}

func (as *AssignStatement) Ast(v Visitor) {
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

func (ii *ImmediateInstruction) Ast(v Visitor) {
	v.Visit(ii)
	v.VisitEnd(ii)
}

func (ii *ImmediateInstruction) GetSize() int {
	return ii.Size
}

type ImpliedInstruction struct {
	OpName string
	Line int

	// filled in later
	OpCode byte
	Size int
}

func (ii *ImpliedInstruction) Ast(v Visitor) {
	v.Visit(ii)
	v.VisitEnd(ii)
}

func (ii *ImpliedInstruction) GetSize() int {
	return ii.Size
}

type LabeledStatement struct {
	LabelName string
	Stmt Node
	Line int
}

func (ls *LabeledStatement) Ast(v Visitor) {
	v.Visit(ls)
	if ls.Stmt != nil {
		ls.Stmt.Ast(v)
	}
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

func (n *DirectWithLabelIndexedInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n *DirectWithLabelIndexedInstruction) GetSize() int {
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

func (n *DirectIndexedInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n *DirectIndexedInstruction) GetSize() int {
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

func (n *DirectWithLabelInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n *DirectWithLabelInstruction) GetSize() int {
	return n.Size
}

type DirectInstruction struct {
	OpName string
	Value int
	Line int

	Payload []byte
}

func (n *DirectInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n *DirectInstruction) GetSize() int {
	return len(n.Payload)
}

type IndirectXInstruction struct {
	OpName string
	Value int
	Line int

	Payload []byte
}

func (n *IndirectXInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n *IndirectXInstruction) GetSize() int {
	return len(n.Payload)
}

type IndirectYInstruction struct {
	OpName string
	Value int
	Line int

	Payload []byte
}

func (n *IndirectYInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n *IndirectYInstruction) GetSize() int {
	return len(n.Payload)
}

type IndirectInstruction struct {
	OpName string
	Value int
	Line int

	Payload []byte
}

func (n *IndirectInstruction) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

func (n *IndirectInstruction) GetSize() int {
	return len(n.Payload)
}


type OrgPseudoOp struct {
	Value int
	Line int
}

func (n *OrgPseudoOp) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

type SubroutineDecl struct {
	Name string
	Line int
}

func (n *SubroutineDecl) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

type DataStatement struct {
	dataList DataList
	Line int

	// filled in later
	Size int
	Offset uint16
}

func (n *DataStatement) Ast(v Visitor) {
	v.Visit(n)
	n.dataList.Ast(v)
	v.VisitEnd(n)
}

func (n *DataStatement) GetSize() int {
	return n.Size
}

type DataWordStatement struct {
	dataList WordList
	Line int

	// filled in later
	Size int
	Offset uint16
}

func (n *DataWordStatement) Ast(v Visitor) {
	v.Visit(n)
	n.dataList.Ast(v)
	v.VisitEnd(n)
}

func (n *DataWordStatement) GetSize() int {
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

type WordList []Node

func (n WordList) Ast(v Visitor) {
	v.Visit(n)
	for _, di := range(n) {
		di.Ast(v)
	}
	v.VisitEnd(n)
}

type StringDataItem string

func (n *StringDataItem) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

type IntegerDataItem int

func (n *IntegerDataItem) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

type LabelCall struct {
	LabelName string
}

func (n *LabelCall) Ast(v Visitor) {
	v.Visit(n)
	v.VisitEnd(n)
}

var programAst *ProgramAST
%}

%union {
	integer int
	str string
	quotedString string
	statementList StatementList
	assignStatement *AssignStatement
	dataList DataList
	wordList WordList
	programAst ProgramAST
	orgPsuedoOp *OrgPseudoOp
	subroutineDecl *SubroutineDecl
	node Node
}

%type <statementList> statementList
%type <assignStatement> assignStatement
%type <node> statement
%type <node> instructionStatement
%type <node> dataStatement
%type <dataList> dataList
%type <wordList> wordList
%type <node> dataItem
%type <programAst> programAst
%type <str> processorDecl
%type <str> labelName
%type <orgPsuedoOp> orgPsuedoOp
%type <subroutineDecl> subroutineDecl
%type <node> numberExpr

%token <str> tokIdentifier
%token <str> tokRegister
%token <integer> tokInteger
%token <str> tokQuotedString
%token <str> tokInstruction
%token tokEqual
%token tokPound
%token tokDot
%token tokComma
%token tokNewline
%token tokData
%token tokDataWord
%token tokProcessor
%token tokLParen
%token tokRParen
%token tokDot
%token tokOrg
%token tokSubroutine

%%

programAst : statementList {
	programAst = &ProgramAST{$1}
}

statementList : statementList tokNewline statement {
	if $3 == nil {
		$$ = $1
	} else {
		$$ = append($1, $3)
	}
} | statement {
	if $1 == nil {
		$$ = []Node{}
	} else {
		$$ = []Node{$1}
	}
}

statement : tokDot tokIdentifier instructionStatement {
	$$ = &LabeledStatement{"." + $2, $3, parseLineNumber}
} | orgPsuedoOp {
	$$ = $1
} | subroutineDecl {
	$$ = $1
} | instructionStatement {
	$$ = $1
} | tokDot tokIdentifier dataStatement {
	$$ = &LabeledStatement{"." + $2, $3, parseLineNumber}
} | dataStatement {
	$$ = $1
} | assignStatement {
	$$ = $1
} | tokIdentifier {
	$$ = &LabeledStatement{$1, nil, parseLineNumber}
} | processorDecl {
	if $1 != "6502" {
		yylex.Error("Unsupported processor: " + $1 + " - Only 6502 is supported.")
	}
	// empty statement
	$$ = nil
} | {
	// empty statement
	$$ = nil
}

dataStatement : tokData dataList {
	$$ = &DataStatement{$2, parseLineNumber, 0, 0}
} | tokDataWord wordList {
	$$ = &DataWordStatement{$2, parseLineNumber, 0, 0}
}

processorDecl : tokProcessor tokInteger {
	$$ = strconv.FormatInt(int64($2), 10)
} | tokProcessor tokIdentifier {
	$$ = $2
}

wordList : wordList tokComma numberExpr {
	$$ = append($1, $3)
} | numberExpr {
	$$ = []Node{$1}
}

dataList : dataList tokComma dataItem {
	$$ = append($1, $3)
} | dataItem {
	$$ = []Node{$1}
}

numberExpr : tokPound tokInteger {
	tmp := IntegerDataItem($2)
	$$ = &tmp
} | labelName {
	$$ = &LabelCall{$1}
}

dataItem : tokQuotedString {
	tmp := StringDataItem($1)
	$$ = &tmp
} | numberExpr {
	$$ = $1
}

assignStatement : tokIdentifier tokEqual tokInteger {
	$$ = &AssignStatement{$1, $3}
}

orgPsuedoOp : tokOrg tokInteger {
	$$ = &OrgPseudoOp{$2, parseLineNumber}
} | tokOrg tokInteger tokComma tokInteger {
	if $4 > 0xff {
		yylex.Error("ORG directive fill parameter must be a single byte.")
	}
	if $4 != 0 {
		yylex.Error("ORG directive only supports filling with zero.")
	}
	$$ = &OrgPseudoOp{$2, parseLineNumber}
}

subroutineDecl : tokIdentifier tokSubroutine {
	$$ = &SubroutineDecl{$1, parseLineNumber}
}

instructionStatement : tokInstruction tokPound tokInteger {
	// immediate address
	$$ = &ImmediateInstruction{$1, $3, parseLineNumber, 0, 0}
} | tokInstruction {
	// no address
	$$ = &ImpliedInstruction{$1, parseLineNumber, 0, 0}
} | tokInstruction labelName tokComma tokRegister {
	$$ = &DirectWithLabelIndexedInstruction{$1, $2, $4, parseLineNumber, 0, 0, 0}
} | tokInstruction tokInteger tokComma tokRegister {
	$$ = &DirectIndexedInstruction{$1, $2, $4, parseLineNumber, []byte{}}
} | tokInstruction labelName {
	$$ = &DirectWithLabelInstruction{$1, $2, parseLineNumber, 0, 0, 0}
} | tokInstruction tokInteger {
	$$ = &DirectInstruction{$1, $2, parseLineNumber, []byte{}}
} | tokInstruction tokLParen tokInteger tokComma tokRegister tokRParen {
	if $5 != "x" && $5 != "X" {
		yylex.Error("Register argument must be X.")
	}
	$$ = &IndirectXInstruction{$1, $3, parseLineNumber, []byte{}}
} | tokInstruction tokLParen tokInteger tokRParen tokComma tokRegister {
	if $6 != "y" && $6 != "Y" {
		yylex.Error("Register argument must be Y.")
	}
	$$ = &IndirectYInstruction{$1, $3, parseLineNumber, []byte{}}
} | tokInstruction tokLParen tokInteger tokRParen {
	$$ = &IndirectInstruction{$1, $3, parseLineNumber, []byte{}}
}

labelName : tokDot {
	$$ = "."
} | tokIdentifier {
	$$ = $1
} | tokDot tokIdentifier {
	$$ = "." + $2
}

%%

