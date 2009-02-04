#include <stdio.h>
#include <stdlib.h>


#define PRGROM_SIZE 16384 // 16 KB
#define CHRROM_SIZE 8192 // 8 KB
#define TRAINER_SIZE 512

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

unsigned char** prg_banks;
unsigned char** chr_banks;
unsigned char trainer[TRAINER_SIZE];
int chr_index = 0;
int prg_index = 0;
int inst_ptr = 0;
NES_header head;


int main(int argc, char* argv[]) {
	FILE* in;
	int i;
	size_t num_read;
	
	
	
	if( argc != 2 ){
		printf("Usage: process_rom <rom_file>\n");
		exit(-1);
	}
	
	in = fopen(argv[1], "r");
	
	// read NES header
	num_read = fread(&head, sizeof(NES_header), 1, in);
	
	// check NES header
	if( 		num_read != 1 ||
		 		!	(head.id[0] == 'N' && head.id[1] == 'E' && 
					head.id[2] == 'S' && head.h1A == 0x1A)				){
		printf("I don't think this is an iNES file.\n");
		exit(-1);
	}
		 
	 // print info
	printf("looks like an NES rom\n");
	
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
	
	for(i=0; i<9; ++i){
		printf("Reserved byte %i: %i\n", i, (int) head.reserved[i]);
	}
	
	//read trainer
	if( head.trainer ){
		printf("Reading trainer...\n");
		num_read = fread(trainer, sizeof(unsigned char), TRAINER_SIZE, in);
		
		if( num_read != TRAINER_SIZE ){
			printf("Error reading trainer from file: I/O error.\n");
			exit(-1);
		}
	}
	
	//read PRG-ROM banks
	printf("Reading PRG-ROM banks...\n");
	prg_banks = (unsigned char **) malloc( sizeof(unsigned char *) * head.prg_banks );
	if( prg_banks == NULL ){
		printf("Out of memory allocating PRG-ROM banks.\n");
		exit(-1);
	}
	for(i=0; i<head.prg_banks; ++i){
		prg_banks[i] = (unsigned char *) malloc( sizeof(unsigned char) * PRGROM_SIZE );
		
		if( prg_banks[i] == NULL ){
			printf("Out of memory allocating PRG-ROM bank %i.\n", i);
			exit(-1);
		}
		
		num_read = fread( prg_banks[i], PRGROM_SIZE, 1, in);
		
		if( num_read != 1 ){
			printf("Error reading PRG-ROM bank %i from file: I/O error.\n", i);
			exit(-1);
		}
	}
	
	
	//read CHR-ROM banks
	printf("Reading CHR-ROM banks...\n");
	chr_banks = (unsigned char **) malloc( sizeof(unsigned char *) * head.chr_banks );
	if( chr_banks == NULL ){
		printf("Out of memory allocating CHR-ROM banks.\n");
		exit(-1);
	}
	for(i=0; i<head.chr_banks; ++i){
		chr_banks[i] = (unsigned char *) malloc( sizeof(unsigned char) * CHRROM_SIZE );
		
		if( chr_banks[i] == NULL ){
			printf("Out of memory allocating CHR-ROM bank %i.\n", i);
			exit(-1);
		}
		
		num_read = fread( chr_banks[i], CHRROM_SIZE, 1, in);
		
		if( num_read != 1 ){
			printf("Error reading CHR-ROM bank %i from file: I/O error.\n", i);
			exit(-1);
		}
	}
	
	fclose(in);
	
	// emulate
	
	
	
	// clean up
	
	for(i=0; i<head.prg_banks; ++i){
		free(prg_banks[i]);
	}
	for(i=0; i<head.chr_banks; ++i){
		free(chr_banks[i]);
	}
	free(prg_banks);
	free(chr_banks);
	
	return 0;
}
