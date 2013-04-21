package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"regexp"
	"strings"
	"strconv"
)

const (
	numberExpression = "(\\$\\d+)"
	identifier = "(\\w[\\w\\d_]+)"
	comment = "(;.*$)?$"
	instructionName = "(\\w\\w\\w)"
	label = "(" + identifier + ":)?"
	args = "(" + identifier + "|" +
		identifier + ",\\s*[xXyY]" + "|" +
		"#" + numberExpression + ")?"

	varAssign = "^\\s*" + identifier + "\\s*=\\s*" + numberExpression +
		"\\s*" + comment
	instruction = "^\\s*" + label + "\\s*" + instructionName + "\\s*" +
		args + "\\s*" + comment
	labelLine = "^\\s*" + label + "\\s*" + comment
	dataLine = "^\\s*" + label + "\\s*.data\\s*(.+)\\s*" + comment
)

var varRE * regexp.Regexp
var instructionRE * regexp.Regexp
var labelLineRE * regexp.Regexp
var dataLineRE * regexp.Regexp
func init() {
	// initialize regexps
	var err error
	varRE, err = regexp.Compile(varAssign)
	if err != nil { panic(err) }
	instructionRE, err = regexp.Compile(instruction)
	if err != nil { panic(err) }
	labelLineRE, err = regexp.Compile(labelLine)
	if err != nil { panic(err) }
	dataLineRE, err = regexp.Compile(dataLine)
	if err != nil { panic(err) }
}

func evaluateNumberExpression(numberExpression string) int {
	if (numberExpression[0] == '$') {
		n64, err := strconv.ParseUint(numberExpression[1:], 16, 16)
		if err != nil { panic(err) }
		return int(n64)
	}
	n64, err := strconv.ParseUint(numberExpression, 10, 16)
	if err != nil { panic(err) }
	return int(n64)
}

func main() {
	// read all lines
	bytes, err := ioutil.ReadFile("hello.6502.asm")
	if err != nil { panic(err) }
	lines := strings.Split(string(bytes), "\n")
	variables := make(map[string]int)
	for i, line := range(lines) {
		var results [][]int
		results = varRE.FindAllStringSubmatchIndex(line, 1)
		if (len(results) == 1) {
			variables[line[results[0][2]:results[0][3]]] = evaluateNumberExpression(line[results[0][4]:results[0][5]])
			continue
		}
		results = instructionRE.FindAllStringSubmatchIndex(line, 1)
		if (len(results) == 1) {
			fmt.Println("instruction", results)
			continue
		}
		results = labelLineRE.FindAllStringSubmatchIndex(line, 1)
		if (len(results) == 1) {
			fmt.Println("label line", results)
			continue
		}
		results = dataLineRE.FindAllStringSubmatchIndex(line, 1)
		if (len(results) == 1) {
			fmt.Println("data line", results)
			continue
		}
		fmt.Fprintln(os.Stderr, "syntax error at line", i, ":", line)
	}
	fmt.Println("variables", variables)
}
