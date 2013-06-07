# jamulator

See the writeup for this project:
[Statically Recompiling NES Games into Native Executables with LLVM and Go](http://andrewkelley.me/post/jamulator.html)

Note: This project is discontinued. See the above article for
the explanation.

## Features

 * Recompile NES games into native executables
   - Only known supported game is Super Mario Brothers 1
 * 6502 assembler / disassembler
 * Unpack & disassemble NES roms and then put them back together
 * Adds a small custom ABI which gives you `putchar` and `exit`
   for making test roms.

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

6. Install the C dependencies:

    ```
    sudo apt-get install libsdl1.2-dev libsdl-gfx1.2-dev libsdl-image1.2-dev libglew1.6-dev libxrandr-dev
    ```

7. Compile, run the tests, and then try it!

    ```
    make
    make test
    ./jamulator -recompile game.nes
    ```

