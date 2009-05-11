#ifndef _CPU6502_H_
#define _CPU6502_H_

using namespace std;
#include <string>
#include <map>


typedef struct {
	unsigned char opcode;
	string assembly;
	unsigned int num_bytes;
	unsigned int num_cycles;
	unsigned int flag;
} Instruction;

class Cpu6502 {
	public:
		// instantiate a processor which will copy the memory
		Cpu6502(unsigned char * rom, int rom_size);
		// specify a starting program counter
		Cpu6502(unsigned char * rom, int rom_size, int init_pc);
		~Cpu6502();
		
		// run the cpu for num_cycles. 0 
		void step(int num_cycles = 1);

		// show the assembly language
		string disassemble();

	private:
		// disassembly information
		static map<unsigned char, Instruction> spec;

		// have we initialized static data yet?
		static bool init_static;
		
		// registers
		signed char AC; // accumulator
		signed char XR; // x register
		signed char YR; // y register

		// status register: 7 6 5 4 3 2 1 0
		// 					S V 1 B D I Z C
		signed char status;

		// the program counter
		unsigned short int PC;

		// the stack pointer, from $0100 to $01FF
		unsigned short int SP;
		

		// the chip's memory
		unsigned char * memory;
		int memory_size;

		// initialize static data
		static void initializeStaticData();
};

#endif

