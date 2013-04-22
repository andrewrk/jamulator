%{
package asm6502

import "fmt"

type Program struct {
	statements * StatementList
}

type StatementList struct {
	This * Statement
	Next * StatementList
}

type Statement struct {
	VarName string
	Value int
}

var program *Program
%}

%union {
	integer int
	identifier string
	statementList StatementList
	statement Statement
	program Program
}

%type <statementList> statementList
%type <statement> statement
%type <program> program

%token <identifier> tokIdentifier
%token <integer> tokInteger
%token tokEqual

%%

program : statementList {
	program = &Program{&$1}
}

statementList : statement statementList {
	$$ = StatementList{&$1, &$2}
} | statement {
	$$ = StatementList{&$1, nil}
}

statement : tokIdentifier tokEqual tokInteger {
	$$ = Statement{$1, $3}
}

%%

