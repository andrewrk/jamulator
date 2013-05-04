#include "stdint.h"

typedef struct {
    int mirroring;
    uint8_t* logicalTables[4];
    uint8_t nametable0[0x400];
    uint8_t nametable1[0x400];
} Nametable;

void Nametable_writeNametableData(Nametable* n, int a, uint8_t v);
uint8_t Nametable_readNametableData(Nametable* n, int a);
void Nametable_setMirroring(Nametable* n, int m);
