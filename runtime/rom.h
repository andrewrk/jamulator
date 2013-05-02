#include "stdint.h"

void rom_start();

uint8_t rom_ppustatus();
void rom_cycle(uint8_t);
void rom_ppuctrl(uint8_t);
void rom_ppumask(uint8_t);
void rom_ppuaddr(uint8_t);
void rom_setppudata(uint8_t);
void rom_oamaddr(uint8_t);
void rom_setoamdata(uint8_t);
void rom_setppuscroll(uint8_t);
