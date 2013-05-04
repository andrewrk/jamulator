#include "stdbool.h"
#include "stdint.h"
#include "nametable.h"

// TODO: namespace everything
typedef enum {
    INTERRUPT_NONE,
    INTERRUPT_IRQ,
    INTERRUPT_RESET,
    INTERRUPT_NMI,
};

typedef enum {
    STATUS_SPRITE_OVERFLOW,
    STATUS_SPRITE0HIT,
    STATUS_VBLANK_STARTED,
};

typedef struct {
    uint8_t tiles[256];
    uint8_t yCoordinates[256];
    uint8_t attributes[256];
    uint8_t xCoordinates[256];
} SpriteData;

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
    uint32_t color;
    int value;
    int pindex;
} Pixel;

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

    int cycle;
    int scanline;
    int timestamp;
    int vblankTime;
    int frameCount;
    int frameCycles;

    bool suppressNmi;
    bool suppressVbl;
    bool overscanEnabled;
    bool spriteLimitEnabled;

    int cycleCount;
} Ppu;

// don't forget to call Ppu_dispose
Ppu* Ppu_new();
void Ppu_dispose(Ppu* p);

void Ppu_writeMirroredVram(Ppu* p, int a, uint8_t v);
void Ppu_writeControl(Ppu* p, uint8_t v);
void Ppu_writeMask(Ppu* p, uint8_t v);
void Ppu_raster(Ppu* p);
void Ppu_step(Ppu* p);
void Ppu_updateEndScanlineRegisters(Ppu* p);
void Ppu_clearStatus(Ppu* p, uint8_t s);
void Ppu_setStatus(Ppu* p, uint8_t s);
uint8_t Ppu_readStatus(Ppu* p);
void Ppu_writeOamAddress(Ppu* p, uint8_t v);
void Ppu_writeOamData(Ppu* p, uint8_t v);
void Ppu_updateBufferedSpriteMem(Ppu* p, int a, uint8_t v);
uint8_t Ppu_readOamData(Ppu* p);
void Ppu_writeScroll(Ppu* p, uint8_t v);
void Ppu_writeAddress(Ppu* p, uint8_t v);
void Ppu_writeData(Ppu* p, uint8_t v);
uint8_t Ppu_readData(Ppu* p);
void Ppu_incrementVramAddress(Ppu* p);
int Ppu_sprPatternTableAddress(Ppu* p, int i);
int Ppu_bgPatternTableAddress(Ppu* p, uint8_t i);
int Ppu_bgPaletteEntry(Ppu* p, uint8_t a, uint16_t pix);
void Ppu_renderTileRow(Ppu* p);
void Ppu_fetchTileAttributes(Ppu* p, PpuTileAttributes* attrs);
