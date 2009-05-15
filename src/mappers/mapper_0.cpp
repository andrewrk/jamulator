/*
	mapper_0.cpp - dummy mapper for use when there is no memory mapper

	This mapper simply fetches the byte at the address without doing
	any translation.

*/

#include "../memory_mapper.h"

export "C" {

class Mapper0 : public MemoryMapper {
	public:

		void initialize(){
			// nothing to do
		}

		byte readByte(word address){
			if( address >= 0x8000 && address < 0xC000 )
				return system->bank1[address-0x8000];
			else if( address >= 0xC000 && address <= 0xFFFF )
				return system->bank2[address-0xC000];
		}

		void writeByte(word address, byte value){
			if( address >= 0x8000 && address < 0xC000 )
				system->bank1[address-0x8000] = value;
			else if( address >= 0xC000 && address <= 0xFFFF )
				system->bank2[address-0xC000] = value;
		}
	private:

};

// factory methods
MemoryMapper * create() {
	return new Mapper0;
}

void destroy(MemoryMapper * obj){
	delete obj;
}

}
