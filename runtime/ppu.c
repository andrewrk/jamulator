#include "stdbool.h"
#include "stdint.h"
#include "rom.h"

const int INTERRUPT_NONE = 0;
const int INTERRUPT_IRQ = 1;
const int INTERRUPT_RESET = 2;
const int INTERRUPT_NMI = 3;

const int MIRRORING_VERTICAL = 0;
const int MIRRORING_HORIZONTAL = 1;
const int MIRRORING_SINGLE_UPPER = 2;
const int MIRRORING_SINGLE_LOWER = 3;

typedef struct {
    int mirroring;
    uint8_t* logicalTables[4];
    uint8_t nametable0[0x400];
    uint8_t nametable1[0x400];
} Nametable;

typedef struct {
    int interruptRequested;
    int cycleCount;
    bool accurate;
} Cpu;

typedef struct {
    uint8_t control;
    uint8_t mask;
    uint8_t status;
    uint8_t vramDataBuffer;
    int vramAddress;
    int vramLatch;
    int spriteRamAddress;
    uint8_t fineX;
    uint8_t data;
    bool writeLatch;
    uint16_t highBitShift;
    uint16_t lowBitShift;
} Registers;

typedef struct {
    uint8_t baseNametableAddress;
    uint8_t vramAddressInc;
    uint8_t spritePatternAddress;
    uint8_t backgroundPatternAddress;
    uint8_t spriteSize;
    uint8_t masterSlaveSel;
    uint8_t nmiOnVblank;
} Flags;

typedef struct {
    bool grayscale;
    bool showBackgroundOnLeft;
    bool showSpritesOnLeft;
    bool showBackground;
    bool showSprites;
    bool intensifyReds;
    bool intensifyGreens;
    bool intensifyBlues;
} Masks;

typedef struct {
    uint8_t tiles[256];
    uint8_t yCoordinates[256];
    uint8_t attributes[256];
    uint8_t xCoordinates[256];
} SpriteData;

typedef struct {
    uint32_t color;
    int value;
    int pindex;
} Pixel;

typedef struct {
    Registers registers;
    Flags flags;
    Masks masks;
    SpriteData spriteData;
    uint8_t vram[0xffff];
    uint8_t spriteRam[0x100];
    Nametable nametables;
    uint8_t paletteRam[0x20];
    unsigned int attributeLocation[0x400];
    unsigned int attributeShift[0x400];
    bool a12High;

    Pixel *palettebuffer;
    int palettebufferSize;
    uint32_t *framebuffer;
    int framebufferSize;

} Ppu;

void cpu_reset(Cpu *c) {
    c->interruptRequested = INTERRUPT_NONE;
    c->cycleCount = 0;
    c->accurate = true;
}

int main() {
    Cpu cpu;
    cpu_reset(&cpu);
    rom_start();
}

uint8_t rom_ppustatus(){}
void rom_ppuctrl(uint8_t b){}
void rom_ppumask(uint8_t b){}
void rom_ppuaddr(uint8_t b){}
void rom_setppudata(uint8_t b){}
void rom_oamaddr(uint8_t b){}
void rom_setoamdata(uint8_t b){}
void rom_setppuscroll(uint8_t b){}
