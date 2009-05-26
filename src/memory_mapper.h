/*
	memory_mapper.h - interface to handle CPU read/write operations

	An NES cartridge can have one of many possible memory mappers.
	This interface should be implemented in plugins named mapper_X.so
	in the mappers/ directory, where X is the mapper number as
	specified in the iNES file header.
	
*/

#ifndef _MEMORY_MAPPER_H_
#define _MEMORY_MAPPER_H_

#include "nes.h"

class Nes;

class MemoryMapper {
	public:
		// must be 1 byte
		typedef unsigned char byte;
		// must be 2 bytes
		typedef unsigned short int word;

		//MemoryMapper();
		//virtual ~MemoryMapper();
		
		// override these functions to provide memory mapping
		// called when the ROM file is loaded
		virtual void initialize(Nes * system);
		// read a byte from memory - called when there is a read from
		// PRG-ROM memory ($8000-$FFFF)
		virtual byte readByte(word address);
		// write a byte to memory - called when there is a write to
		// PRG-RAM memory ($8000-$FFFF)
		virtual void writeByte(word address, byte value);
	private:
};

#endif
