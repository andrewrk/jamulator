/*
	nes.h - Nes emulator object

	Synopsis:

	// create a new emulator to emulate smb3.nes without a screen
	Nes emu("smb3.nes"); 
	// run emulation for 46 cycles
	emu.runCycles(46);

	// press Start on player 1's controller right now
	JoyConfig cfg = JCfgFromByte(JCFG_START);
	emu.input(0, cfg);
	emu.runCycles(10);

	// press A on player 1's controller at 100 cycles
	cfg = JCfgFromByte(JCFG_A);
	emu.input(100, 0, cfg);
	emu.runCycles(45);
	
	//provide an input map and emulate it
	JoyInputMap data(); //JoyInputMap is a Vector<ulong, JoyConfig>;
	data.push_back(400, JoyCfgFromByte(JCFG_B)); // press B at 400
	data.push_back(450, JoyCfgFromByte(JCFG_A_AND_B)); // press A at 450
	data.push_back(620, JoyCfgFromByte(JCFG_B)); // let go of A at 620
	data.push_back(700, JoyCfgFromByte(JCFG_NONE)); // let go of B at 700
	emu.inputMap(data);
	emu.runUntil(800); // run until absolute cycle count is 800

*/

#include <string>
#include <gmodule>
#include "SDL.h"
#include "cpu_6502.h"

#include "joy_config.h"
#include "nes_exceptions.h"
#include "memory_mapper.h"

class Nes {
	public:
		typedef unsigned long long int ulong;
		typedef Vector<ulong, JoyConfig> JoyInputMap;

		// create an Nes emulator from .nes file file with no audio/video
		Nes(string &file);
		// attach the screen to surface
		Nes(string &file, SDL_Surface* surface);
		// destructor
		~Nes();
		
		// run the cpu for cycles cycles, returns how many it actually ran,
		// which could be greater than cycles by up to 3 cycles
		ulong runCycles(ulong cycles);
		// run the cpu until a specific cycle count, returns the current cycle
		// count. could be greater than stop_cycle by up to 3
		ulong runUntil(ulong stop_cycle);
		// return the current cycle count since the emulation started
		ulong cycleCount();

		// input joystick data. player_index - 0: player 1, 1: player 2.
		void input(int player_index, JoyConfig &cfg);
		// input joystick data at a specific CPU cycle
		void input(int input_cycle, int player_index, JoyConfig &cfg);
		// input a set of data which maps cycles to joystick configurations
		void inputMap(JoyInputMap &data);
		
		// detach the screen and stop emulating video and audio
		void detachScreen();
		// (re?)attach a screen (video and sound) to emulation
		void attachScreen(SDL_Surface* surface);


	private:
		typedef unsigned char byte; // must be 1 byte
		typedef unsigned short int word; // must be 2 bytes

		typedef struct {
			byte id[4]; // "NES\01A"

			byte num_prg_banks; //number of 16 KB ROM banks
			byte num_chr_banks; //number of 8 KB VROM banks
			
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
			byte num_ram_banks;

			unsigned int screen_type:1; // 0: NTSC, 1: PAL
			unsigned int reserved_2:7;

			byte reserved_3[6];
			
		} NES_header;

		// constants
		const int cpu_mem_size = 65536; // 64 KB
		const int prgrom_size = 16384; // 16 KB
		const int chrrom_size = 8192; // 8 KB
		const int trainer_size = 512;
		const int ram_size = 8192; // 8 KB

		const int pal_cycles_per_sec = 1773447;
		const int pal_screen_width = 256;
		const int pal_screen_height = 240;
		const int pal_nmi = 35469 // pal_cycles_per_sec / 50

		const int ntsc_cycles_per_sec =	1789773; // it's actually 1789772.5
		const int ntsc_screen_width = 256;
		const int ntsc_screen_height = 224;
		const int ntsc_nmi = 29830; // ntsc_cycles_per_sec / 60

		// cartridge data
		NES_header cart_data;


		// cartridge memory data
		byte * prg_banks;
		byte * chr_banks;
		byte * trainer; // 512 byte trainer, if present

		// pointers to the prgrom that the CPU sees in its address space
		// this replaces $C000 - $FFFF
		byte * bank1;
		byte * bank2;

		//CPU memory data
		byte cpu_memory[cpu_mem_size];
		Cpu6502 * cpu; // the CPU emulator

		// dependent on PAL/NTSC
		int clock_speed;
		int screen_width, screen_height;
		int nmi_period;
		
		// memory mapper class
		MemoryMapper * mmc;

		// connection module for the MemoryMapper plugin
		GModule * mm_module;
		void (*mm_destroy)(MemoryMapper * mm);

		// callback function from the CPU to check for interupts 
		Cpu6502::InteruptType cpuLoop();

		// PPU data
		byte * patternTables[2]; // pattern table pointers
		byte * nameTables[4]; // name table pointers
		byte * ppu_memory[0x4000]; // memory for the PPU
	
	// MemoryMapper needs to be extremely fast and have access
	// to all emulated memory
	friend class MemoryMapper;
};

