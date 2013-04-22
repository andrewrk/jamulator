# jamulator

Currently an in progress 6502 assembler.

## Getting Started

1. Set up your `$GOPATH`. Make sure the `bin` folder from your go path
   is in `$PATH`.
2. Install the lexer:

```
go get github.com/superjoe30/nex
```

3. Install llvm-3.1 from your package manager.
4. Install gollvm:

```
export CGO_CFLAGS=$(llvm-config-3.1 --cflags)
export CGO_LDFLAGS="$(llvm-config-3.1 --ldflags) -Wl,-L$(llvm-config-3.1 --libdir) -lLLVM-$(llvm-config-3.1 --version)"
go get github.com/axw/gollvm/llvm
```

5. Install the rest of the go dependencies:

```
go install
```

6. Compile & run:

```
make && ./jamulator hello.6502.asm
```

## Roadmap

 * Get a 6502 assembler working.
 * Get a 6502 disassembler working.
 * Use LLVM to recompile 6502 assembly with a custom ABI into
   native executables.
 * Get a NES ROM disassembler working.
 * Use LLVM to recompile NES games into native executables.

