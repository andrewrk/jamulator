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
void rom_start(uint8_t interrupt);

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
