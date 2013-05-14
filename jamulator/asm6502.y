%{
package jamulator

import (
	"fmt"
	"strconv"
	"container/list"
)

type AssignStatement struct {
	VarName string
	Value int
}

type LabelStatement struct {
	LabelName string
	Line int
}

type LabeledStatement struct {
	Label *LabelStatement
	Stmt interface{}
}

type OrgPseudoOp struct {
	Value int
	Fill byte
	Line int
}

type InstructionType int
const (
	ImmediateInstruction InstructionType = iota
	ImpliedInstruction
	DirectWithLabelIndexedInstruction
	DirectIndexedInstruction
	DirectWithLabelInstruction
	DirectInstruction
	IndirectXInstruction
	IndirectYInstruction
	IndirectInstruction
)

type Instruction struct {
	Type InstructionType
	OpName string
	Line int

	// not all fields are used by all instruction types.
	Value int
	LabelName string
	RegisterName string

	// filled in later
	OpCode byte
	Offset int
	Payload []byte
}

type DataStmtType int
const (
	ByteDataStmt DataStmtType = iota
	WordDataStmt
)

type DataStatement struct {
	Type DataStmtType
	dataList *list.List
	Line int

	// filled in later
	Offset int
	Payload []byte
}


type IntegerDataItem int
type StringDataItem string
type LabelCall struct {
	LabelName string
}
type ProgramAst struct {
	List *list.List
}

var programAst ProgramAst
%}

%union {
	integer int
	str string
	list *list.List
	assignStatement *AssignStatement
	orgPsuedoOp *OrgPseudoOp
	node interface{}
}

%type <list> statementList
%type <assignStatement> assignStatement
%type <node> statement
%type <node> instructionStatement
%type <node> dataStatement
%type <list> dataList
%type <list> wordList
%type <node> dataItem
%type <str> processorDecl
%type <str> labelName
%type <orgPsuedoOp> orgPsuedoOp
%type <node> subroutineDecl
%type <node> numberExpr
%type <node> numberExprOptionalPound

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
%token tokColon
%token tokOrg
%token tokSubroutine

%%

programAst : statementList {
	programAst = ProgramAst{$1}
}

statementList : statementList tokNewline statement {
	if $3 == nil {
		$$ = $1
	} else {
		$$ = $1
		$$.PushBack($3)
	}
} | statement {
	if $1 == nil {
		$$ = list.New()
	} else {
		$$ = list.New()
		$$.PushBack($1)
	}
}

statement : tokDot tokIdentifier instructionStatement {
	$$ = &LabeledStatement{
		&LabelStatement{"." + $2, parseLineNumber},
		$3,
	}
} | tokIdentifier tokColon instructionStatement {
	$$ = &LabeledStatement{
		&LabelStatement{$1, parseLineNumber},
		$3,
	}
} | orgPsuedoOp {
	$$ = $1
} | subroutineDecl {
	$$ = $1
} | instructionStatement {
	$$ = $1
} | tokDot tokIdentifier dataStatement {
	$$ = &LabeledStatement{
		&LabelStatement{"." + $2, parseLineNumber},
		 $3,
	 }
} | tokIdentifier tokColon dataStatement {
	$$ = &LabeledStatement{
		&LabelStatement{$1, parseLineNumber},
		$3,
	}
} | dataStatement {
	$$ = $1
} | assignStatement {
	$$ = $1
} | tokIdentifier {
	$$ = &LabelStatement{$1, parseLineNumber}
} | tokIdentifier tokColon {
	$$ = &LabelStatement{$1, parseLineNumber}
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
	$$ = &DataStatement{
		Type: ByteDataStmt,
		dataList: $2,
		Line: parseLineNumber,
	}
} | tokDataWord wordList {
	$$ = &DataStatement{
		Type: WordDataStmt,
		dataList: $2,
		Line: parseLineNumber,
	}
}

processorDecl : tokProcessor tokInteger {
	$$ = strconv.FormatInt(int64($2), 10)
} | tokProcessor tokIdentifier {
	$$ = $2
}

wordList : wordList tokComma numberExprOptionalPound {
	$$ = $1
	$$.PushBack($3)
} | numberExprOptionalPound {
	$$ = list.New()
	$$.PushBack($1)
}

dataList : dataList tokComma dataItem {
	$$ = $1
	$$.PushBack($3)
} | dataItem {
	$$ = list.New()
	$$.PushBack($1)
}

numberExpr : tokPound tokInteger {
	tmp := IntegerDataItem($2)
	$$ = &tmp
} | labelName {
	$$ = &LabelCall{$1}
}

numberExprOptionalPound : numberExpr {
	$$ = $1
} | tokInteger {
	tmp := IntegerDataItem($1)
	$$ = &tmp
}

dataItem : tokQuotedString {
	tmp := StringDataItem($1)
	$$ = &tmp
} | numberExprOptionalPound {
	$$ = $1
}

assignStatement : tokIdentifier tokEqual tokInteger {
	$$ = &AssignStatement{$1, $3}
}

orgPsuedoOp : tokOrg tokInteger {
	$$ = &OrgPseudoOp{$2, 0xff, parseLineNumber}
} | tokOrg tokInteger tokComma tokInteger {
	if $4 > 0xff {
		yylex.Error("ORG directive fill parameter must be a single byte.")
	}
	$$ = &OrgPseudoOp{$2, byte($4), parseLineNumber}
}

subroutineDecl : tokIdentifier tokSubroutine {
	$$ = &LabelStatement{$1, parseLineNumber}
}

instructionStatement : tokInstruction tokPound tokInteger {
	$$ = &Instruction{
		Type: ImmediateInstruction,
		OpName: $1,
		Value: $3,
		Line: parseLineNumber,
	}
} | tokInstruction {
	$$ = &Instruction{
		Type: ImpliedInstruction,
		OpName: $1,
		Line: parseLineNumber,
	}
} | tokInstruction labelName tokComma tokRegister {
	$$ = &Instruction{
		Type: DirectWithLabelIndexedInstruction,
		OpName: $1,
		LabelName: $2,
		RegisterName: $4,
		Line: parseLineNumber,
	}
} | tokInstruction tokInteger tokComma tokRegister {
	$$ = &Instruction{
		Type: DirectIndexedInstruction,
		OpName: $1,
		Value: $2,
		RegisterName: $4,
		Line: parseLineNumber,
	}
} | tokInstruction labelName {
	$$ = &Instruction{
		Type: DirectWithLabelInstruction,
		OpName: $1,
		LabelName: $2,
		Line: parseLineNumber,
	}
} | tokInstruction tokInteger {
	$$ = &Instruction{
		Type: DirectInstruction,
		OpName: $1,
		Value: $2,
		Line: parseLineNumber,
	}
} | tokInstruction tokLParen tokInteger tokComma tokRegister tokRParen {
	if $5 != "x" && $5 != "X" {
		yylex.Error("Register argument must be X.")
	}
	$$ = &Instruction{
		Type: IndirectXInstruction,
		OpName: $1,
		Value: $3,
		Line: parseLineNumber,
	}
} | tokInstruction tokLParen tokInteger tokRParen tokComma tokRegister {
	if $6 != "y" && $6 != "Y" {
		yylex.Error("Register argument must be Y.")
	}
	$$ = &Instruction{
		Type: IndirectYInstruction,
		OpName: $1,
		Value: $3,
		Line: parseLineNumber,
	}
} | tokInstruction tokLParen tokInteger tokRParen {
	$$ = &Instruction{
		Type: IndirectInstruction,
		OpName: $1,
		Value: $3,
		Line: parseLineNumber,
	}
}

labelName : tokDot {
	$$ = "."
} | tokIdentifier {
	$$ = $1
} | tokDot tokIdentifier {
	$$ = "." + $2
}

%%

