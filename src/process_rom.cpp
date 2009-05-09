#include <iostream>

#include <cstdio>
#include <cstdlib>
#include <cstring>

#include "cpu6502.h"
#include "debug.h"


#define PRGROM_SIZE 16384 // 16 KB
#define CHRROM_SIZE 8192 // 8 KB
#define TRAINER_SIZE 512


typedef enum {
	No_mapper  = 0,
	Nintendo_MMC1  = 1,
	CNROM_switch  = 2,
	UNROM_switch  = 3,
	Nintendo_MMC3 = 4,
	Nintendo_MMC5  = 5,
	FFE_F4xxx  = 6,
	AOROM_switch  = 7,
	FFE_F3xxx  = 8,
	Nintendo_MMC2  = 9,
	Nintendo_MMC4  = 10,
	ColorDreams_chip = 11,
	FFE_F6xxx  = 12,
	CPROM_switch = 13,
	hundred_in_1_switch = 15,
	Bandai_chip  = 16,
	FFE_F8xxx  = 17,
	Jaleco_SS8806_chip = 18,
	Namcot_106_chip = 19,
	Nintendo_DiskSystem = 20,
	Konami_VRC4a  = 21,
	Konami_VRC2a  = 22,
	Konami_VRC2a_2  = 23,
	Konami_VRC6  = 24,
	Konami_VRC4b  = 25,
	Irem_G_101_chip = 32,
	Taito_TC0190_TC0350 = 33,
	Nina_1_board  = 34,
	Tengen_RAMBO_1_chip = 64,
	Irem_H_3001_chip = 65,
	GNROM_switch = 66,
	SunSoft3_chip = 67,
	SunSoft4_chip = 68,
	SunSoft5_FME_7_chip = 69,
	Camerica_chip = 71,
	Irem_74HC161_32_based = 78,
	AVE_Nina_3_board = 79,
	AVE_Nina_6_board = 81,
	pirate_hk_sf3_chip = 91
} MemoryMapperType;

typedef struct {
	unsigned char id[3]; // "NES"
	unsigned char h1A; // 0x1A
	
	unsigned char num_rom_banks; //number of 16 KB ROM banks
	unsigned char num_vrom_banks; //number of 8 KB VROM banks
	
	unsigned int mirroring:1; // 1 for vertical, 0 for horizontal
	unsigned int battery_ram:1; // 1 for battery-backed RAM $6000-$7FFF
	unsigned int trainer:1; // 1 for a 512-byte trainer at $7000-$71FF

	// 1 for a 4-screen VRAM layout. overrides mirroring. only available
	// with certain types of mappers.
	unsigned int four_screen:1; 
	unsigned int rom_mapper_low:4;
	unsigned int vs_system:1; /* 1 for VS-System cartridges */
	unsigned int reserved_1:3;
	unsigned int rom_mapper_high:4; /* higher 4 bits */

	// number of 8KB RAM banks. assume 1x8kB RAM page when this byte is zero.
	unsigned char num_ram_banks;

	unsigned int screen_type:1; // 0: NTSC, 1: PAL
	unsigned int reserved_2:7;

	unsigned char reserved_3[6];
	
} NES_header;

unsigned char** vrom_banks;
unsigned char** rom_banks;
unsigned char trainer[TRAINER_SIZE];
int chr_index = 0;
int prg_index = 0;
int inst_ptr = 0;
NES_header head;

#ifdef DEBUG
void test6502(char * file);
#endif

int main(int argc, char* argv[]) {
	FILE* in;
	int i;
	size_t num_read;
	
#ifdef DEBUG
	if(argc == 3 && strcmp(argv[1], "--test") == 0 ){
		test6502(argv[2]);
		exit(0);
	}
#endif
	
	
	if( argc != 2 ){
		printf("Usage: process_rom <rom_file>\n");
		exit(-1);
	}
	
	in = fopen(argv[1], "rb");
	
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
	
	printf("number of 16 KB ROM banks: %i\n", (int) head.num_rom_banks);
	printf("number of 8 KB VROM banks: %i\n", (int) head.num_vrom_banks);
	

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

	unsigned char mem_mapper = 
		(head.rom_mapper_high << 4) | head.rom_mapper_low;
	
	printf("Mapper number: %i\n", (int) mem_mapper);
	
	if( head.trainer ){
		printf("Trainer present: yes\n");
	} else {
		printf("Trainer present: no\n");
	}

	if( head.vs_system ){
		printf("VS System: yes\n");
	} else {
		printf("VS System: no\n");
	}

	printf("Number of 8 kB RAM banks: %i\n", (int) head.num_ram_banks );

	if( head.screen_type ){
		printf("Screen type: PAL\n");
	} else {
		printf("Screen type: NTSC\n");
	}

	/*for(i=0; i<9; ++i){
		printf("Reserved byte %i: %i\n", i, (int) head.reserved[i]);
	} */
	
	//read trainer
	if( head.trainer ){
		printf("Reading trainer...\n");
		num_read = fread(trainer, sizeof(unsigned char), TRAINER_SIZE, in);
		
		if( num_read != TRAINER_SIZE ){
			printf("Error reading trainer from file: I/O error.\n");
			exit(-1);
		}
	}
	
	
	//read ROM banks
	printf("Reading ROM banks...\n");
	rom_banks = (unsigned char **) malloc( sizeof(unsigned char *) * head.num_rom_banks );
	if( rom_banks == NULL ){
		printf("Out of memory allocating CHR-ROM banks.\n");
		exit(-1);
	}
	for(i=0; i<head.num_rom_banks; ++i){
		rom_banks[i] = (unsigned char *) malloc( sizeof(unsigned char) * CHRROM_SIZE );
		
		if( rom_banks[i] == NULL ){
			printf("Out of memory allocating CHR-ROM bank %i.\n", i);
			exit(-1);
		}
		
		num_read = fread( rom_banks[i], CHRROM_SIZE, 1, in);
		
		if( num_read != 1 ){
			printf("Error reading CHR-ROM bank %i from file: I/O error.\n", i);
			exit(-1);
		}
	} 
	
	//read VROM banks
	printf("Reading VROM banks...\n");
	vrom_banks = (unsigned char **) malloc( sizeof(unsigned char *) * head.num_vrom_banks );
	if( vrom_banks == NULL ){
		printf("Out of memory allocating PRG-ROM banks.\n");
		exit(-1);
	}
	for(i=0; i<head.num_vrom_banks; ++i){
		vrom_banks[i] = (unsigned char *) malloc( sizeof(unsigned char) * PRGROM_SIZE );
		
		if( vrom_banks[i] == NULL ){
			printf("Out of memory allocating PRG-ROM bank %i.\n", i);
			exit(-1);
		}
		
		num_read = fread( vrom_banks[i], PRGROM_SIZE, 1, in);
		
		if( num_read != 1 ){
			printf("Error reading PRG-ROM bank %i from file: I/O error.\n", i);
			exit(-1);
		}
	}
	
	fclose(in);

	int sum = sizeof(head) + head.num_vrom_banks * 8 * 1024 
		+ (head.trainer	? 512 : 0) + head.num_rom_banks * 16 * 1024;
	printf("read %i bytes.\n", sum);

	// disassemble
	Cpu6502 emulator(rom_banks[0], 16 * 1024);
	cout << emulator.disassemble() << endl;
	
	// emulate
	
	
	
	// clean up
	
	for(i=0; i<head.num_vrom_banks; ++i){
		free(vrom_banks[i]);
	}
	free(vrom_banks);

	for(i=0; i<head.num_rom_banks; ++i){
		free(rom_banks[i]);
	} 
	free(rom_banks); 
	
	return 0;
}

#ifdef DEBUG
void test6502(char * file){
	FILE* in;
	in = fopen(file, "rb");
	fseek(in, 0, SEEK_END);
	size_t size = ftell(in);
	fseek(in, 0, SEEK_SET);
	
	unsigned char * program = (unsigned char *) malloc(size);
	fread(program, 1, size, in);

	Cpu6502 emulator(program, size);
	cout << emulator.disassemble() << endl;

}
#endif
