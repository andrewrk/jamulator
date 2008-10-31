#include <stdio.h>
#include <stdlib.h>

#include "debug.h"

typedef struct {
	unsigned char id[3]; // "NES"
	unsigned char h1A; // 0x1A
	
	unsigned char prg_banks; //number of 16 KB PRG-ROM banks
	unsigned char chr_banks; //number of 8 KB CHR-ROM / VROM banks
	
	unsigned int mirroring:1; // 1 for vertical, 0 for horizontal
	unsigned int battery_ram:1; // 1 for battery-backed RAM $6000-$7FFF
	unsigned int trainer:1; // 1 for a 512-byte trainer at $7000-$71FF
	unsigned int four_screen:1; /* 1 for a four-screen VRAM layout. overrides
											 mirroring. only available with certain types
											 of mappers, for example type #1 (boulderdash)
											 and type #5 (castlevania3). */
	unsigned int rom_mapper:4; /* type of rom mapper
											0 - None
											1 - Megaman2, Bomberman2, Destiny, etc.
											2 - Castlevania,LifeForce,etc.
											3 - QBert,PipeDream,Cybernoid,etc.
											4 - SilverSurfer,SuperContra,Immortal,etc.
											5 - Castlevania3
											6 - F4xxx carts off FFE CDROM (experimental)
											8 - F3xxx carts off FFE CDROM (experimental)
											11 - CrystalMines,TaginDragon,etc. (experiment)
											15 - 100-in-1 cartridge */
	unsigned char reserved[9];
	
} NES_header;

int main(int argc, char* argv[]) {
	FILE* in;
	NES_header head;
	
	if( argc != 2 ){
		printf("Usage: process_rom <rom_file>\n");
		exit(-1);
	}
	
	in = fopen(argv[1], "r");
	
	fread(&head, sizeof(NES_header), 1, in);
	
	if( head.id[0] == 'N' && head.id[1] == 'E' && 
		 head.id[2] == 'S' && head.h1A == 0x1A ){
		logg(lvDebug, "looks like an NES rom\n");
		
		printf("number of 16 KB PRG-ROM banks: %i\n", (int) head.prg_banks);
		printf("number of 8 KB CHR-ROM banks: %i\n", (int) head.chr_banks);
		
		
		
	
		if( head.four_screen ){
			printf("Mirroring: four-screen\n");
		} else if( head.mirroring ) {
			printf("Mirroring: vertical\n");
		} else {
			printf("Mirroring: horizontal\n");
		}
		
		if( head.battery_ram ){
			printf("Battery packed ram: yes\n");
		} else {
			printf("Battery packed ram: no\n");
		}
		
		printf("Mapper number: %i\n", (int) head.rom_mapper);
		
		if( head.trainer ){
			printf("Trainer present: yes\n");
		} else {
			printf("Trainer present: no\n");
		}
		
		/*if( reserved ){
			printf("Reserved bits zero: yes\n");
		} else {
			printf("Reserved bits zero: no\n");
		}*/
		
		
	} else {
		logg(lvDebug, "I don't think this is an NES rom.\n");
	}
	
	fclose(in);
	return 0;
}
