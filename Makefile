build: asm6502/y.go asm6502/asm6502.nn.go
	go build -o jamulator main.go

asm6502/y.go: asm6502/asm6502.y
	go tool yacc -o asm6502/y.go -v /dev/null asm6502/asm6502.y

asm6502/asm6502.nn.go: asm6502/asm6502.nex
	${GOPATH}/bin/nex -e asm6502/asm6502.nex

clean:
	rm -f asm6502/asm6502.nn.go asm6502/y.go jamulator

.PHONY: build clean dev
