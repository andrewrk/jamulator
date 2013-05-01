org $C000

msg: .data "Hello, world!", 10, 0

Reset_Routine:

LDX #$00       ; starting index in X register

loop: LDA msg, X     ; read 1 char
BEQ loopend    ; end loop if we hit the \0
STA $2008      ; putchar (custom ABI I made for testing)
INX
JMP loop       ; repeat

loopend:
LDA #$00       ; return code 0
STA $2009      ; exit (custom ABI I made for testing)

IRQ_Routine: rti ; do nothing
NMI_Routine: rti ; do nothing

org   $FFFA
dc.w  NMI_Routine
dc.w  Reset_Routine
dc.w  IRQ_Routine
