/*
	cpu_6502.h - Emulate a 6502 CPU

	This class is threadsafe - you can create multiple instances
	of it.

*/


#ifndef CPU_6502_H_
#define CPU_6502_H_

class Nes;
class MemoryMapper;

class Cpu6502 {
	public:
		// Loop6502 returns:
		enum InteruptType {
			IntNone = 0,
			IntIrq = 1,
			IntNmi = 2,
			IntQuit = 3
		}; 

		// 6502 status flags
		enum StatusFlag {
			FlagC = 0x01, // carry occured
			FlagZ = 0x02, // result is zero
			FlagI = 0x04, // interupts disabled
			FlagD = 0x08, // decimal mode
			FlagB = 0x10, // break (0 on stk after int)
			FlagR = 0x20, // always 1
			FlagV = 0x40, // overflow occured
			FlagN = 0x80  // result is negative
		};
		
		// sizeof(byte) = 1 and sizeof(word) = 2
		typedef unsigned char byte;
		typedef unsigned short int word;
		typedef signed char offset;

		typedef union {
#ifdef LSB_FIRST
			struct { byte l,h; } b;
#else
			struct { byte h,l; } b;
#endif
			word w;
		} pair;

		// constructor
		Cpu6502(
			// CPU clock speed in Hz
			int clock_speed,
			// how many cycles to wait before calling the callback function
			int interupt_period, 
			// arbitrary data which will be passed along with callbacks
			void * context,
			// after interupt_check cycles, call this function
			// to check for an interupt.
			InteruptType (*loopCallback)(void * context),
			// read from memory callback. this function should
			// return the byte at the address
			byte (*readFunc)(void * context, word address),
			// write to memory callback. this function should
			// write byte to memory at address
			void (*writeFunc)(void * context, word address, byte value)
		);
		~Cpu6502();

		// reset the registers before starting execution with run().
		// it sets registers to their initial values.
		void reset();

		// generate an interupt of a given type. IntNmi will cause
		// a non-maskable interupt. IntIrq will cause a normal
		// interupt, unless FlagI set in the status register.
		void interupt(byte type);

		// run 6502 code until loopCallback() call returns IntQuit. When
		// it's done, it will return the number of cycles executed,
		// possibly negative.
		int run();

	private:
		// constants
		// how many cycles each opcode takes. There are special cases
		// which get handled in the big switch statement
		static const byte op_cycles[256];
		
		// what to set Z and N in the status register to
		// based on the instruction
		static const byte zn_table[256];

		// variables necessary to emulate the CPU
		byte ac; // accumulator
		byte xr; // x register
		byte yr; // y register
		byte st; // status register
		byte sp; // stack pointer
		pair pc; // program counter

		// how fast the CPU runs in Hz
		int clock_speed;

		// callback functions
		void * callback_context;
		InteruptType (*loopCallback)(void * context);
		byte (*readFunc)(void * context, word address);
		void (*writeFunc)(void * context, word address, byte value);
		
		// how often to check for interupts
		int interupt_period;
		// how many cycles until the next interupt check
		int cycles_left;
};

#endif 
