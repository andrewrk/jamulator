
;   EXAMPLE.ASM 	(6502 Microprocessor)
;

	    processor	6502

	    mac     ldax
	    lda     [{1}]
	    ldx     [{1}]+1
	    endm
	    mac     ldaxi
	    lda     #<[{1}]
	    ldx     #>[{1}]
	    endm
	    mac     stax
	    sta     [{1}]
	    stx     [{1}]+1
	    endm
	    mac     pushxy
	    txa
	    pha
	    tya
	    pha
	    endm
	    mac     popxy
	    pla
	    tay
	    pla
	    tax
	    endm
	    mac     inc16
	    inc     {1}
	    bne     .1
	    inc     {1}+1
.1
	    endm

STOP1	    equ %00000000	    ;CxCTL  1 Stop bit
STOP2	    equ %10000000	    ;CxCTL  2 Stop bits (WL5:1.5, WL8&par:1)
WL5	    equ %01100000	    ;CxCTL  Wordlength
WL6	    equ %01000000
WL7	    equ %00100000
WL8	    equ %00000000
RCS	    equ %00010000	    ;CxCTL  1=Select baud, 0=ext. receiver clk

B76800	    equ %0000		    ;CxCTL  Baud rates	(1.2288 Mhz clock)
B75	    equ %0001
B100	    equ %0010
B150	    equ %0011
B200	    equ %0100
B300	    equ %0101
B400	    equ %0110
B600	    equ %0111
B800	    equ %1000
B1200	    equ %1001
B1600	    equ %1010
B2400	    equ %1011
B3200	    equ %1100
B4800	    equ %1101
B6400	    equ %1110
B12800	    equ %1111

PARODD	    equ %00100000	    ;CxCMD  Select Parity
PAREVEN     equ %01100000
PARMARK     equ %10100000
PARSPACE    equ %11100000
PAROFF	    equ %00000000

RECECHO     equ %00010000	    ;CxCMD  Receiver Echo mode
TMASK	    equ %00001100
TDISABLE    equ %00000000	    ;CxCMD  Transmitter modes
TDISABLER   equ %00001000	    ;RTS stays asserted
TENABLE     equ %00000100
TBREAK	    equ %00001100	    ;send break

UA_IRQDSBL  equ %00000010
DTRRDY	    equ %00000001	    ;~DTR output is inverted (low)

SR_PE	    equ %00000001	    ;CxSTAT  Status
SR_FE	    equ %00000010	    ;NOTE: writing dummy data causes RESET
SR_OVRUN    equ %00000100
SR_RDRFULL  equ %00001000
SR_TDREMPTY equ %00010000
SR_DCD	    equ %00100000
SR_DSR	    equ %01000000
SR_INTPEND  equ %10000000


T1_OEPB7    equ %10000000	    ;x_ACR
T1_FREERUN  equ %01000000	    ;T1 free running mode
T1_ONESHOT  equ %00000000
T2_ICPB6    equ %00100000	    ;T2 counts pulses on PB6
T2_ONESHOT  equ %00000000	    ;T2 counts phase2 transitions
SRC_OFF     equ %00000000	    ;shift register control
SRC_INT2    equ %00000100
SRC_INPH2   equ %00001000
SRC_INEXT   equ %00001100
SRC_OUTFR   equ %00010000	    ;free running output using T2
SRC_OUTT2   equ %00010100
SRC_OUTPH2  equ %00011000
SRC_OUTEXT  equ %00011100
PBLE	    equ %00000010	    ;on CB1 transition (in/out).
PALE	    equ %00000001	    ;on CA1 transition (in).  data retained

				    ;x_PCR
CB2_I_NEG   equ %00000000	    ;interrupt on neg trans, r/w ORB clears
CB2_I_NEGI  equ %00100000	    ; same, but r/w ORB does not clear int
CB2_I_POS   equ %01000000	    ;interrupt on pos trans, r/w ORB clears
CB2_I_POSI  equ %01100000	    ; same, but r/w ORB does not clear int
CB2_O_HSHAK equ %10000000	    ;CB2=0 on r/w ORB, CB2=1 on CB1 transition
CB2_O_PULSE equ %10100000	    ;CB2=0 for one clock after r/w ORB
CB2_O_MANLO equ %11000000	    ;CB2=0
CB2_O_MANHI equ %11100000	    ;CB2=1

CA2_I_NEG   equ %00000000	    ;interrupt on neg trans, r/w ORA clears
CA2_I_NEGI  equ %00100000	    ; same, but r/w ORA does not clear int
CA2_I_POS   equ %01000000	    ;interrupt on pos trans, r/w ORA clears
CA2_I_POSI  equ %01100000	    ; same, but r/w ORA does not clear int
CA2_O_HSHAK equ %10000000	    ;CA2=0 on r/w ORA, CA2=1 on CA1 transition
CA2_O_PULSE equ %10100000	    ;CA2=0 for one clock after r/w ORA
CA2_O_MANLO equ %11000000	    ;CA2=0
CA2_O_MANHI equ %11100000	    ;CA2=1


CB1_THI     equ %00010000
CB1_TLO     equ %00000000
CA1_THI     equ %00000001
CA1_TLO     equ %00000000

VIRPEND     equ %10000000	    ;x_IFR
IRENABLE    equ %10000000	    ;x_IER  1's enable ints  0=no change
IRDISABLE   equ %00000000	    ;x_IER  1's disable ints 0=no change

IRT1	    equ %01000000
IRT2	    equ %00100000
IRCB1	    equ %00010000
IRCB2	    equ %00001000
IRSR	    equ %00000100
IRCA1	    equ %00000010
IRCA2	    equ %00000001

	    seg.u   bss
	    org     $0000	    ;RAM (see below)
	    org     $2000	    ;unused
	    org     $4000	    ;unused

	    org     $6000	    ;6551 CHANNEL #1
C1DATA	    ds	    1
C1STAT	    ds	    1
C1CMD	    ds	    1
C1CTL	    ds	    1

	    org     $8000	    ;6551 CHANNEL #2
C2DATA	    ds	    1
C2STAT	    ds	    1
C2CMD	    ds	    1
C2CTL	    ds	    1

	    org     $A000	    ;6522 (HOST COMM)
H_ORB	    ds	    1
H_ORAHS     ds	    1		    ;with CA2 handshake
H_DDRB	    ds	    1
H_DDRA	    ds	    1
H_T1CL	    ds	    1		    ;read clears interrupt flag
H_T1CH	    ds	    1		    ;write clears interrupt flag
H_T1CLL     ds	    1
H_T1CHL     ds	    1		    ;write clears interrupt flag
H_T2CL	    ds	    1		    ;read clears interrupt flag
H_T2CH	    ds	    1		    ;write clears interrupt flag
H_SR	    ds	    1
H_ACR	    ds	    1
H_PCR	    ds	    1
H_IFR	    ds	    1
H_IER	    ds	    1
H_ORA	    ds	    1		    ;no CA2 handshake

	    org     $C000	    ;6522 (IO COMM)
I_ORB	    ds	    1
I_ORAHS     ds	    1		    ;	(same comments apply)
I_DDRB	    ds	    1
I_DDRA	    ds	    1
I_T1CL	    ds	    1
I_T1CH	    ds	    1
I_T1CLL     ds	    1
I_T1CHL     ds	    1
I_T2CL	    ds	    1
I_T2CH	    ds	    1
I_SR	    ds	    1
I_ACR	    ds	    1
I_PCR	    ds	    1
I_IFR	    ds	    1
I_IER	    ds	    1
I_ORA	    ds	    1



	    ;	--------------------------   ZERO PAGE	 -------------------
	    seg.u   data
	    org     $00

	    ;	--------------------------  NORMAL RAM	 -------------------
	    org     $0100

RAMEND	    equ     $2000

	    ;	--------------------------     CODE	 -------------------

	    seg     code
	    org     $F000
PROMBEG     equ     .

RESET	    subroutine
	    sei 		;disable interrupts
	    ldx     #$FF	;reset stack
	    txs

	    lda     #$FF
	    sta     H_DDRA
	    sta     C1STAT	;reset 6551#1 (garbage data)
	    sta     C2STAT	;reset 6551#2
	    lda     #$7F	;disable all 6522 interrupts
	    sta     H_IER
	    sta     I_IER

	    lda     #%00010000	;76.8 baud, 8 bits, 1 stop
	    sta     C1CTL
	    lda     #%00000101	;no parity, enable transmitter & int
	    sta     C1CMD
	    lda     #$AA	;begin transmision
	    sta     C1DATA

	    lda     #%00011111	;9600 baud, 8 bits, 1 stop
	    sta     C2CTL
	    lda     #%00000101
	    sta     C2CMD
	    lda     #$41
	    sta     C2DATA

	    cli 		;enable interrupts

.1	    jsr     LOAD
	    jsr     SAVE
	    jmp     .1

LOAD	    subroutine

	    ldx     #0
.1	    txa
	    sta     $0500,x
	    inx
	    bne     .1
	    rts

SAVE	    subroutine

	    ldx     #0
.2	    lda     $0500,x
	    sta     H_ORA
	    inx
	    bne     .2
	    rts

NMI	    rti

	    subroutine
IRQ	    bit     C1STAT
	    bpl     .1
	    pha
	    lda     #$AA
	    sta     C1DATA
	    lda     C1DATA
	    pla
	    rti
.1	    bit     C2STAT
	    bpl     .2
	    pha
	    lda     #$41
	    sta     C2DATA
	    lda     C2DATA
	    pla
.2	    rti

	    ;	VECTOR	------------------------------------------------

	    seg     vector
	    org     $FFFA
	    dc.w    NMI
	    dc.w    RESET
	    dc.w    IRQ

PROMEND     equ     .

