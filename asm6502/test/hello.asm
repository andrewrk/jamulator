out  = $2008   ; write to this memory address to output a char
exit = $2009   ; exit with return code

org $C000

msg: .data "Hello, world!", 10, 0

Reset_Routine SUBROUTINE

LDX #$00       ; starting index in X register

loop: LDA msg, X     ; read 1 char
BEQ loopend    ; end loop if we hit the \0
STA out
INX
JMP loop       ; repeat

loopend:
LDA #$00       ; return code 0
STA exit       ; exit

IRQ_Routine: rti ; do nothing
NMI_Routine: rti ; do nothing

org   $FFFA
dc.w  NMI_Routine
dc.w  Reset_Routine
dc.w  IRQ_Routine
