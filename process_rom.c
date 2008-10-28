#include "SDL.h"
#include <stdio.h>

typedef struct {
	char id[3]; // "NES"
	char h1A; // 0x1A
	
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
		printf("looks like an NES rom\n");
	} else {
		printf("I don't think this is an NES rom.\n");
	}
	
	fclose(in);
	return 0;
}
