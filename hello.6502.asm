out  = $4018   ; write to this memory address to output a char
exit = $4019   ; exit with return code

LDX #$00       ; starting index in X register

loop:

LDA msg, X     ; read 1 char
BEQ loopend    ; end loop if we hit the \0
STA out
INX
JMP loop       ; repeat

loopend:
LDA #$00       ; return code 0
STA exit       ; exit

msg: .data "Hello, world!", 10, 0
