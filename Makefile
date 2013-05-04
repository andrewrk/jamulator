build: asm6502/y.go asm6502/asm6502.nn.go runtime/runtime.a
	go build -o jamulator main.go

asm6502/y.go: asm6502/asm6502.y
	go tool yacc -o asm6502/y.go -v /dev/null asm6502/asm6502.y

asm6502/asm6502.nn.go: asm6502/asm6502.nex
	${GOPATH}/bin/nex -e asm6502/asm6502.nex

clean:
	rm -f asm6502/asm6502.nn.go asm6502/y.go jamulator

test:
	go test asm6502/*.go
	go test nes/*.go
	go test

runtime/runtime.a: runtime/main.o runtime/ppu.o runtime/nametable.o
	ar rcs runtime/runtime.a runtime/main.o runtime/ppu.o runtime/nametable.o

runtime/main.o: runtime/main.c
	clang -o runtime/main.o -c runtime/main.c

runtime/ppu.o: runtime/ppu.c
	clang -o runtime/ppu.o -c runtime/ppu.c

runtime/nametable.o: runtime/nametable.c
	clang -o runtime/nametable.o -c runtime/nametable.c

.PHONY: build clean dev test
