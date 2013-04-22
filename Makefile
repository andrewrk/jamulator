build:
	go tool yacc -o asm6502/y.go -v /dev/null asm6502/asm6502.y
	${GOPATH}/bin/nex -e asm6502/asm6502.nex
	go build -o jamulator main.go

clean:
	rm -f asm6502/asm6502.nn.go asm6502/y.go jamulator

.PHONY: build clean dev
