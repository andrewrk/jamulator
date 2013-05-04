#include "rom.h"
#include "assert.h"
#include "ppu.h"

Ppu* p;
int main() {
    p = Ppu_new();
    Nametable_setMirroring(&p->nametables, rom_mirroring);
    // copy chr data to vram
    assert(rom_chr_bank_count == 1);
    for (int i = 0; i < 0x2000; ++i) {
        p->vram[i] = rom_chr_data[i];
    }
    rom_start();
    Ppu_dispose(p);
}

void rom_cycle(uint8_t b){}

uint8_t rom_ppustatus() {
    return Ppu_readStatus(p);
}

void rom_ppuctrl(uint8_t b) {
    Ppu_writeControl(p, b);
}

void rom_ppumask(uint8_t b) {
    Ppu_writeMask(p, b);
}

void rom_ppuaddr(uint8_t b) {
    Ppu_writeAddress(p, b);
}

void rom_setppudata(uint8_t b) {
    Ppu_writeData(p, b);
}

void rom_oamaddr(uint8_t b) {
    Ppu_writeOamAddress(p, b);
}

void rom_setoamdata(uint8_t b) {
    Ppu_writeOamData(p, b);
}

void rom_setppuscroll(uint8_t b) {
    Ppu_writeScroll(p, b);
}
