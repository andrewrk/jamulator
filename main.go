package main

import (
	"./asm6502"
	"fmt"
)


func main() {
	f, err := asm6502.ParseFile("test.6502.asm")
	if err != nil { panic(err) }
	fmt.Println(f)
}
