#include "stdbool.h"
#include "rom.h"

int main() {
    rom_start();
}

uint8_t rom_ppustatus() {
    return 0;
}
void rom_cycle(uint8_t b){}
void rom_ppuctrl(uint8_t b){}
void rom_ppumask(uint8_t b){}
void rom_ppuaddr(uint8_t b){}
void rom_setppudata(uint8_t b){}
void rom_oamaddr(uint8_t b){}
void rom_setoamdata(uint8_t b){}
void rom_setppuscroll(uint8_t b){}
