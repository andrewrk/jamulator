# jamulator

## Features

 * 6502 assembler / disassembler
 * unpack & disassemble NES roms and then put them back together

## Roadmap

 * More intelligent disassembling
 * Unpack CHR ROM into PCX files and vice versa
 * Ability to emulate an NES ROM
 * When emulating, capture how memory addresses are used to figure
   out what is data and what is code
 * Use LLVM to recompile 6502 assembly with a custom ABI into
   native executables.
 * Use LLVM to recompile NES games into native executables.


## Getting Started Developing

1. Set up your `$GOPATH`. Make sure the `bin` folder from your go path
   is in `$PATH`.
2. Install the lexer:

    ```
    go get github.com/blynn/nex
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
    make test
    ```

7. Compile & run:

    ```
    make
    ./jamulator
    ```

8. If you want to compile a .bc file to a native EXE:

    ```
    llc -filetype=obj file.bc
    gcc hello.6502.asm.o
    ./a.out
    ```

