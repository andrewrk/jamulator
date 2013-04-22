
//line asm6502/asm6502.y:2
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
	statements StatementList
}

func (p Program) Ast(v Visitor) {
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

var program *Program

//line asm6502/asm6502.y:131
type yySymType struct {
	yys int
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
	program Program
}

const tokIdentifier = 57346
const tokInteger = 57347
const tokQuotedString = 57348
const tokEqual = 57349
const tokPound = 57350
const tokColon = 57351
const tokComma = 57352
const tokNewline = 57353
const tokData = 57354

var yyToknames = []string{
	"tokIdentifier",
	"tokInteger",
	"tokQuotedString",
	"tokEqual",
	"tokPound",
	"tokColon",
	"tokComma",
	"tokNewline",
	"tokData",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line asm6502/asm6502.y:235



//line yacctab:1
var yyExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyNprod = 20
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 28

var yyAct = []int{

	20, 9, 18, 14, 13, 16, 17, 15, 8, 10,
	12, 26, 25, 22, 21, 24, 23, 27, 3, 1,
	19, 11, 7, 6, 5, 4, 2, 28,
}
var yyPact = []int{

	-3, -1000, -3, -1000, -1, -7, -1000, -8, -1000, -2,
	8, -1000, -1000, -1000, -1000, -1000, 11, 10, 2, 1,
	-1000, -1000, -1000, -1000, -1000, 13, 8, -1000, -1000,
}
var yyPgo = []int{

	0, 26, 25, 18, 24, 23, 22, 20, 0, 19,
}
var yyR1 = []int{

	0, 9, 1, 1, 3, 3, 3, 3, 3, 6,
	7, 7, 8, 8, 5, 2, 4, 4, 4, 4,
}
var yyR2 = []int{

	0, 1, 2, 1, 2, 2, 1, 2, 1, 2,
	3, 1, 1, 1, 2, 3, 3, 1, 4, 2,
}
var yyChk = []int{

	-1000, -9, -1, -3, -2, -4, -5, -6, 11, 4,
	12, -3, 11, 11, 11, 9, 7, 8, 4, -7,
	-8, 6, 5, 5, 5, 10, 10, 4, -8,
}
var yyDef = []int{

	0, -2, 1, 3, 0, 0, 6, 0, 8, 17,
	0, 2, 4, 5, 7, 14, 0, 0, 19, 9,
	11, 12, 13, 15, 16, 0, 0, 18, 10,
}
var yyTok1 = []int{

	1,
}
var yyTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12,
}
var yyTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c > 0 && c <= len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return fmt.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return fmt.Sprintf("state-%v", s)
}

func yylex1(lex yyLexer, lval *yySymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		c = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			c = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		c = yyTok3[i+0]
		if c == char {
			c = yyTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		fmt.Printf("lex %U %s\n", uint(char), yyTokname(c))
	}
	return c
}

func yyParse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		fmt.Printf("char %v in %v\n", yyTokname(yychar), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar = yylex1(yylex, &yylval)
	}
	yyn += yychar
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yychar { /* valid shift */
		yychar = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yychar {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error("syntax error")
			Nerrs++
			if yyDebug >= 1 {
				fmt.Printf("%s", yyStatname(yystate))
				fmt.Printf("saw %s\n", yyTokname(yychar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					fmt.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				fmt.Printf("error recovery discards %s\n", yyTokname(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}
			yychar = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		fmt.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		//line asm6502/asm6502.y:168
		{
		program = &Program{yyS[yypt-0].statementList}
	}
	case 2:
		//line asm6502/asm6502.y:172
		{
		if yyS[yypt-0].statement == nil {
			yyVAL.statementList = yyS[yypt-1].statementList
		} else {
			yyVAL.statementList = append(yyS[yypt-1].statementList, yyS[yypt-0].statement)
		}
	}
	case 3:
		//line asm6502/asm6502.y:178
		{
		if yyS[yypt-0].statement == nil {
			yyVAL.statementList = []Node{}
		} else {
			yyVAL.statementList = []Node{yyS[yypt-0].statement}
		}
	}
	case 4:
		//line asm6502/asm6502.y:186
		{
		yyVAL.statement = yyS[yypt-1].assignStatement
	}
	case 5:
		//line asm6502/asm6502.y:188
		{
		yyVAL.statement = yyS[yypt-1].instructionStatement
	}
	case 6:
		//line asm6502/asm6502.y:190
		{
		yyVAL.statement = yyS[yypt-0].labelStatement
	}
	case 7:
		//line asm6502/asm6502.y:192
		{
		yyVAL.statement = yyS[yypt-1].dataStatement
	}
	case 8:
		//line asm6502/asm6502.y:194
		{
		// empty statement
	yyVAL.statement = nil
	}
	case 9:
		//line asm6502/asm6502.y:199
		{
		yyVAL.dataStatement = DataStatement{yyS[yypt-0].dataList}
	}
	case 10:
		//line asm6502/asm6502.y:203
		{
		yyVAL.dataList = append(yyS[yypt-2].dataList, yyS[yypt-0].dataItem)
	}
	case 11:
		//line asm6502/asm6502.y:205
		{
		yyVAL.dataList = []Node{yyS[yypt-0].dataItem}
	}
	case 12:
		//line asm6502/asm6502.y:209
		{
		yyVAL.dataItem = StringDataItem(yyS[yypt-0].quotedString)
	}
	case 13:
		//line asm6502/asm6502.y:211
		{
		yyVAL.dataItem = IntegerDataItem(yyS[yypt-0].integer)
	}
	case 14:
		//line asm6502/asm6502.y:215
		{
		yyVAL.labelStatement = LabelStatement{yyS[yypt-1].identifier}
	}
	case 15:
		//line asm6502/asm6502.y:219
		{
		yyVAL.assignStatement = AssignStatement{yyS[yypt-2].identifier, yyS[yypt-0].integer}
	}
	case 16:
		//line asm6502/asm6502.y:223
		{
		// immediate address
	yyVAL.instructionStatement = ImmediateInstruction{yyS[yypt-2].identifier, yyS[yypt-0].integer}
	}
	case 17:
		//line asm6502/asm6502.y:226
		{
		// no address
	yyVAL.instructionStatement = ImpliedInstruction{yyS[yypt-0].identifier}
	}
	case 18:
		//line asm6502/asm6502.y:229
		{
		yyVAL.instructionStatement = AbsoluteWithLabelIndexedInstruction{yyS[yypt-3].identifier, yyS[yypt-2].identifier, yyS[yypt-0].identifier}
	}
	case 19:
		//line asm6502/asm6502.y:231
		{
		yyVAL.instructionStatement = AbsoluteWithLabelInstruction{yyS[yypt-1].identifier, yyS[yypt-0].identifier}
	}
	}
	goto yystack /* stack new state and value */
}
