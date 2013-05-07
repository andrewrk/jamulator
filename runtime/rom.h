#include "stdint.h"

enum {
    ROM_MIRRORING_VERTICAL,
    ROM_MIRRORING_HORIZONTAL,
    ROM_MIRRORING_SINGLE_UPPER,
    ROM_MIRRORING_SINGLE_LOWER,
};

enum {
    ROM_INTERRUPT_NONE,
    ROM_INTERRUPT_NMI,
    ROM_INTERRUPT_RESET,
    ROM_INTERRUPT_IRQ,
};


uint8_t rom_mirroring;
uint8_t rom_chr_bank_count;

// write the chr rom into dest
void rom_read_chr(uint8_t* dest);

// starts executing the PRG ROM.
// this function returns when the RTI instruction is executed,
// or the program exits.
// when an interrupt occurs, call rom_start with the interrupt
// index.
void rom_start(uint32_t interrupt);

// called after every instruction with the number of
// cpu cycles that have passed.
void rom_cycle(uint8_t);

// PPU hooks
uint8_t rom_ppustatus();
void rom_ppuctrl(uint8_t);
void rom_ppumask(uint8_t);
void rom_ppuaddr(uint8_t);
void rom_setppudata(uint8_t);
void rom_oamaddr(uint8_t);
void rom_setoamdata(uint8_t);
void rom_setppuscroll(uint8_t);
void rom_ppu_writedma(uint8_t);

// APU hooks
void rom_apu_write_square1control(uint8_t);
void rom_apu_write_square1sweeps(uint8_t);
void rom_apu_write_square1low(uint8_t);
void rom_apu_write_square1high(uint8_t);
void rom_apu_write_square2control(uint8_t);
void rom_apu_write_square2sweeps(uint8_t);
void rom_apu_write_square2low(uint8_t);
void rom_apu_write_square2high(uint8_t);
void rom_apu_write_trianglecontrol(uint8_t);
void rom_apu_write_trianglelow(uint8_t);
void rom_apu_write_trianglehigh(uint8_t);
void rom_apu_write_noisebase(uint8_t);
void rom_apu_write_noiseperiod(uint8_t);
void rom_apu_write_noiselength(uint8_t);
void rom_apu_write_dmcflags(uint8_t);
void rom_apu_write_dmcdirectload(uint8_t);
void rom_apu_write_dmcsampleaddress(uint8_t);
void rom_apu_write_dmcsamplelength(uint8_t);
void rom_apu_write_controlflags1(uint8_t);
void rom_apu_write_controlflags2(uint8_t);

// controller hooks
void rom_pad_write1(uint8_t);
void rom_pad_write2(uint8_t);
