.org $c000
Reset_Routine:
    lda #$28
    sta $86

    lda #$ff
    ldx #$00
    sec
    sbc $86, X
NMI_Routine:
    rti
.org $fffa
    .dw NMI_Routine
    .dw Reset_Routine
    .dw NMI_Routine
