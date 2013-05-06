build: jamulator/y.go jamulator/asm6502.nn.go runtime/runtime.a
	go build -o jamulate main.go

jamulator/y.go: jamulator/asm6502.y
	go tool yacc -o jamulator/y.go -v /dev/null jamulator/asm6502.y

jamulator/asm6502.nn.go: jamulator/asm6502.nex
	${GOPATH}/bin/nex -e jamulator/asm6502.nex

clean:
	rm -f jamulator/asm6502.nn.go
	rm -f jamulator/y.go
	rm -f jam
	rm -f runtime/runtime.a
	rm -f runtime/main.o
	rm -f runtime/ppu.o
	rm -f runtime/nametable.o

test:
	go test jamulator/*.go
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
