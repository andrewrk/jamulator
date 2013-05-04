#include "nametable.h"
#include "rom.h"

void Nametable_setMirroring(Nametable* n, int m) {
    n->mirroring = m;

    switch (m) {
        case ROM_MIRRORING_HORIZONTAL:
            n->logicalTables[0] = n->nametable0;
            n->logicalTables[1] = n->nametable0;
            n->logicalTables[2] = n->nametable1;
            n->logicalTables[3] = n->nametable1;
            break;
        case ROM_MIRRORING_VERTICAL:
            n->logicalTables[0] = n->nametable0;
            n->logicalTables[1] = n->nametable1;
            n->logicalTables[2] = n->nametable0;
            n->logicalTables[3] = n->nametable1;
            break;
        case ROM_MIRRORING_SINGLE_UPPER:
            n->logicalTables[0] = n->nametable0;
            n->logicalTables[1] = n->nametable0;
            n->logicalTables[2] = n->nametable0;
            n->logicalTables[3] = n->nametable0;
            break;
        case ROM_MIRRORING_SINGLE_LOWER:
            n->logicalTables[0] = n->nametable1;
            n->logicalTables[1] = n->nametable1;
            n->logicalTables[2] = n->nametable1;
            n->logicalTables[3] = n->nametable1;
            break;
    }
}

void Nametable_writeNametableData(Nametable* n, int a, uint8_t v) {
    n->logicalTables[(a&0xC00)>>10][a&0x3FF] = v;
}

uint8_t Nametable_readNametableData(Nametable* n, int a) {
    return n->logicalTables[(a&0xC00)>>10][a&0x3FF];
}
