# jamulator

Currently an in progress 6502 assembler.

## Getting Started

1. Set up your `$GOPATH`. Make sure the `bin` folder from your go path
   is in `$PATH`.
2. Install the lexer:

    ```
    go get github.com/superjoe30/nex
    ```

3. Install llvm-3.1 or llvm-3.2 from your package manager.
4. Install gollvm (replace 3.2 with 3.1 if you want):

    ```
    export CGO_CFLAGS=$(llvm-config-3.2 --cflags)
    export CGO_LDFLAGS="$(llvm-config-3.2 --ldflags) -Wl,-L$(llvm-config-3.2 --libdir) -lLLVM-$(llvm-config-3.2 --version)"
    go get github.com/axw/gollvm/llvm
    ```

5. Install the rest of the go dependencies:

    ```
    go install
    ```

6. Run the tests:

    ```
    go test asm6502/*.go
    ```

6. Compile & run:

    ```
    make
    ./jamulator
    ```

7. If you want to compile a .bc file to a native EXE:

    ```
    llc -filetype=obj file.bc
    gcc hello.6502.asm.o
    ./a.out
    ```

## Roadmap

 * Get a 6502 assembler working.
 * Get a 6502 disassembler working.
 * Use LLVM to recompile 6502 assembly with a custom ABI into
   native executables.
 * Get a NES ROM disassembler working.
 * Use LLVM to recompile NES games into native executables.

