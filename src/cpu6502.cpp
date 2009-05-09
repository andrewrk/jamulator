#include <string>
#include <iostream>
#include <fstream>
#include <cstring>
#include <sstream>

#include "cpu6502.h"
#include "debug.h"
#include "utility.h"

#define SET_SIGN(x)	status &= (((x) & 0x80) << 7)
#define SET_ZERO(x) if( x ) status &= 0xfd; else status &= 0x02

//#define IF_CARRY()	

// static information
bool Cpu6502::init_static = false;
map<unsigned char, Instruction> Cpu6502::spec;

void Cpu6502::initializeStaticData() {
	// load the disassembly information into the map
	ifstream in("cpu6502.spec", ifstream::in);
	string line;
	while(in.good()){
		getline(in, line);
		
		// check if it's empty or starts with #
		// trim
		trim(line);
		// if empty or first char is #
		if( line.size() == 0 || line[0] == '#' ){
			// ignore
			continue;
		} else {
			// split by ':'
			vector<string> fields;
			tokenize(line, fields, ":");

			Instruction inst;
			inst.opcode = (unsigned char) hexToInt(fields[0]);
			inst.assembly = fields[1];
			inst.num_bytes = (unsigned int) stringToInt(fields[2]);
			inst.num_cycles = (unsigned int) stringToInt(fields[3]);
			inst.flag = hexToInt(fields[4]);

			// insert into map
			spec.insert( pair<unsigned char, Instruction>(inst.opcode, inst) );
		}
	}
	in.close();
}

Cpu6502::Cpu6502(unsigned char * rom, int rom_bytes) :
	memory_size(rom_bytes)
{
	// initialize static data
	if( ! init_static ){
		initializeStaticData();
		init_static = true;
	}

	// initialize registers
	SP = 0x0100;
	PC = 0;
	AC = XR = YR = 0;
	status = 0x20; // bit 5 is 1, everything else 0

	// copy memory
	memory = new unsigned char[rom_bytes];
	memcpy(memory, rom, rom_bytes);

}


Cpu6502::~Cpu6502(){
	delete[] memory;
}

void Cpu6502::step(int num_steps){
	int i;
	/*for(i=0;i<num_steps;++i){
		unsigned char op_code = read_memory(PC++);
		unsigned char * src = read_memory(PC);
		switch(op_code){
			
		}
	}*/
}

string Cpu6502::disassemble(){
	stringstream ss;

	// for each instruction in memory
	unsigned char * ptr = memory;
	while( ptr-memory < memory_size ){
		// get the instruction associated with the opcode
		if( spec.count(*ptr) == 0){
			ss << "Invalid opcode: " << *ptr << "\n"
				<< "Unable to continue diassembly.\n";
			return ss.str();
		} else {
			Instruction inst = spec[*ptr];

			// go to next byte
			++ptr;

			// translate to assembly
			// figure out what to replace %i with
			string i = "";
			unsigned int immediate;
			switch( inst.num_bytes ){
				case 1:
					// there is no i
					break;
				case 2:
					// one-byte immediate
					i = "$" + intToHex((unsigned int) *ptr);
					++ptr;
					break;
				case 3:
					// two-byte immediate
					immediate = *ptr;
					++ptr;
					immediate |= *ptr << 8;
					++ptr;

					i = "$" + intToHex(immediate);
					break;
				default:
					ASSERT(false);
			}

			// replace %i with the generated value
			replace( inst.assembly, "%i", i);

			// add to disassembly
			ss << inst.assembly << "\n";
		}
	}

	return ss.str();
}
