out  = $4018         ; write to this memory address to output a char
exit = $4019         ; exit with return code

LDX #$00             ; starting index in X register

loop: LDA msg, X     ; read 1 char
BEQ loopend          ; end loop if we hit the \0
