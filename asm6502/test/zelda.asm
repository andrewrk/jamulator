;Zelda Title Screen Program
;-------------------
;Binary created using DAsm 2.12 running on an Amiga.

   PROCESSOR   6502

   ORG   $C000    ;16Kb PRG-ROM, 8Kb CHR-ROM

   dc.b "Zelda Simulator, ©1998 Chris Covell (ccovell@direct.ca)"

Reset_Routine  SUBROUTINE
   cld         ;Clear decimal flag
   sei         ;Disable interrupts
.WaitV   lda $2002
   bpl .WaitV     ;Wait for vertical blanking interval
   ldx #$00
   stx $2000
   stx $2001      ;Screen display off, amongst other things
   dex
   txs         ;Top of stack at $1FF

;Clear (most of) the NES' WRAM. This routine is ripped from "Duck Hunt" - I should probably clear all
;$800 bytes.
   ldy #$06    ;To clear 7 x $100 bytes, from $000 to $6FF?
   sty $01        ;Store count value in $01
   ldy #$00
   sty $00
   lda #$00

.Clear   sta ($00),y    ;Clear $100 bytes
   dey
   bne .Clear

   dec $01        ;Decrement "banks" left counter
   bpl .Clear     ;Do next if >= 0

;********* Initialize Palette to specified colour ********

   ldx   #$3F
   stx   $2006
   ldx   #$00
   stx   $2006

   ldx   #$27     ;Colour Value (Peach)
   ldy   #$20     ;Clear BG & Sprite palettes.
.InitPal stx $2007
   dey
   bne   .InitPal
;*********************************************************

   ldx   #$3F
   stx   $2006
   ldx   #$00
   stx   $2006

   ldx   #$0   ;Beginning of Palette
   ldy   #$20  ;# of colours

.SetPal lda   .CMAP,X
   sta   $2007
   inx
   dey
   bne .SetPal

;************************************************

;********** Set up Attribute Table

   ldx   #$23
   stx   $2006
   ldx   #$C0
   stx   $2006

   ldx   #$0   ;Beginning of Attribute Map
   ldy   #$40  ;Fill out Table

.SetAtt  lda   .AttrMap,X
   sta   $2007
   inx
   dey
   bne .SetAtt

;********** Set up Name Table

   ldx   #$20
   stx   $2006
   ldx   #$00
   stx   $2006

   ldx   #$0   ;Beginning of Map
   ldy   #$0   ;256 Tiles

.SetMap1  lda   .TitleMap1,X
   sta   $2007
   inx
   dey
   bne .SetMap1

   ldx   #$0   ;Beginning of Map
   ldy   #$0   ;256 Tiles

.SetMap2  lda   .TitleMap2,X
   sta   $2007
   inx
   dey
   bne .SetMap2

   ldx   #$0   ;Beginning of Map
   ldy   #$0   ;256 Tiles

.SetMap3  lda   .TitleMap3,X
   sta   $2007
   inx
   dey
   bne .SetMap3

   ldx   #$0   ;Beginning of Map
   ldy   #$C0  ;192 Tiles

.SetMap4  lda   .TitleMap4,X
   sta   $2007
   inx
   dey
   bne .SetMap4

;*************************************

;***** Set up Sprites **************************

   ldx   #$0
   stx   $2003

   ldx   #$0   ;First Sprite
   ldy   #$0   ;# of Sprites * 4

.SetSpr lda .SprMap,X
   sta   $2004
   inx
   dey
   bne   .SetSpr

;***********************************************

;Enable vblank interrupts, etc.
   lda   #%10010000
   sta   $2000
   lda   #%00011110    ;Screen on, sprites on, show leftmost 8 pixels, colour
   sta   $2001
;  cli            ;Enable interrupts(?)

;Now just loop forever?
.Loop jmp   .Loop

.CMAP dc.b #$36,#$0D,#$00,#$10,#$36,#$17,#$27,#$0D
      dc.b #$36,#$08,#$1A,#$28,#$36,#$30,#$31,#$22
      dc.b #$36,#$30,#$31,#$11,#$36,#$15,#$15,#$15
      dc.b #$36,#$08,#$1A,#$28,#$36,#$30,#$31,#$22

.AttrMap dc.b #$05,#$05,#$05,#$05,#$05,#$05,#$05,#$05
         dc.b #$08,#$6A,#$5A,#$5A,#$5A,#$5A,#$9A,#$22
         dc.b #$00,#$66,#$55,#$55,#$55,#$55,#$99,#$00
         dc.b #$00,#$6E,#$5F,#$55,#$5D,#$DF,#$BB,#$00
         dc.b #$00,#$0A,#$0A,#$0A,#$0A,#$0A,#$0A,#$00
         dc.b #$00,#$00,#$C0,#$30,#$00,#$00,#$00,#$00
         dc.b #$00,#$00,#$CC,#$33,#$00,#$00,#$00,#$00
         dc.b #$00,#$20,#$FC,#$F3,#$00,#$00,#$F0,#$F0

.SprMap dc.b #$27,#$CA,#$02,#$28,#$2F,#$CB,#$02,#$28  ;Sprites 1 & 2
        dc.b #$27,#$CC,#$02,#$30,#$2F,#$CD,#$02,#$30  ;Sprites 3 & 4
        dc.b #$27,#$D6,#$02,#$50,#$2C,#$D6,#$02,#$A0  ;...etc...
        dc.b #$27,#$CC,#$42,#$C8,#$2F,#$CD,#$42,#$C8
        dc.b #$27,#$CA,#$42,#$D0,#$2F,#$CB,#$42,#$D0
        dc.b #$31,#$D2,#$02,#$57,#$31,#$D4,#$02,#$5F
        dc.b #$3F,#$D4,#$02,#$24,#$41,#$D4,#$02,#$63
        dc.b #$4F,#$D6,#$02,#$2C,#$4F,#$D2,#$02,#$CB  ;A few Leaves
        dc.b #$57,#$CE,#$02,#$73,#$5F,#$CF,#$02,#$73  ;Middle Leaves...
        dc.b #$57,#$D0,#$02,#$7B,#$5F,#$D1,#$02,#$7B
        dc.b #$67,#$D2,#$02,#$7A,#$7B,#$D4,#$02,#$90
        dc.b #$7B,#$D6,#$02,#$BC,#$82,#$D2,#$02,#$50  ;More Leaves
        dc.b #$77,#$CB,#$82,#$28,#$7F,#$CA,#$82,#$28
        dc.b #$77,#$CD,#$82,#$30,#$7F,#$CC,#$82,#$30
        dc.b #$77,#$CD,#$C2,#$C8,#$7F,#$CC,#$C2,#$C8
        dc.b #$77,#$CB,#$C2,#$D0,#$7F,#$CA,#$C2,#$D0
        dc.b #$AF,#$A3,#$00,#$50,#$AF,#$A5,#$00,#$58  ;Waterfall
        dc.b #$AF,#$A7,#$00,#$60,#$AF,#$A9,#$00,#$68
        dc.b #$B7,#$B2,#$00,#$50,#$B7,#$B4,#$00,#$58
        dc.b #$B7,#$B6,#$00,#$60,#$B7,#$B8,#$00,#$68
        dc.b #$C9,#$C2,#$00,#$50,#$D1,#$C3,#$00,#$50
        dc.b #$C9,#$C4,#$00,#$58,#$D1,#$C5,#$00,#$58
        dc.b #$C9,#$C6,#$00,#$60,#$D1,#$C7,#$00,#$60
        dc.b #$C9,#$C8,#$00,#$68,#$D1,#$C9,#$00,#$68
        dc.b #$D9,#$C2,#$00,#$50,#$E1,#$C3,#$00,#$50
        dc.b #$D9,#$C4,#$00,#$58,#$E1,#$C5,#$00,#$58
        dc.b #$D9,#$C6,#$00,#$60,#$E1,#$C7,#$00,#$60
        dc.b #$D9,#$C8,#$00,#$68,#$E1,#$C9,#$00,#$68
        dc.b #$67,#$A0,#$03,#$58,#$67,#$A0,#$03,#$60  ;Sword
        dc.b #$67,#$A0,#$03,#$68,#$67,#$A0,#$03,#$70
        dc.b #$67,#$A0,#$03,#$78,#$67,#$A0,#$03,#$80
        dc.b #$67,#$A0,#$03,#$88,#$00,#$B0,#$00,#$00  ;Dummy Sprite

.TitleMap1 dc.b "Begin                                                           "
          dc.b "                                                                "
          dc.b "                                                                "
          dc.b "                                                                "
.TitleMap2 dc.b "                                                                "
          dc.b "                                                                "
          dc.b "                                                                "
          dc.b "                                                                "
.TitleMap3 dc.b "                                                                "
          dc.b "                                                                "
          dc.b "                                                                "
          dc.b "                                                                "
.TitleMap4 dc.b "                                                                "
          dc.b "                                                                "
          dc.b "                                                             End"

NMI_Routine SUBROUTINE

   ldx   #$0
   stx   $2005
   stx   $2005

   rti

IRQ_Routine       ;Dummy label
   rti

;That's all the code. Now we just need to set the vector table approriately.

   ORG   $FFFA,0
   dc.w  NMI_Routine
   dc.w  Reset_Routine
   dc.w  IRQ_Routine    ;Not used, just points to RTI


;The end.
