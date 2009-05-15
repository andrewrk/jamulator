#include <iostream>
#include <gmodule>

using namespace std;

#include "nes.h"

// create an Nes emulator from .nes file file with no audio/video
Nes::Nes(string &file) :
	prg_banks(NULL),
	chr_banks(NULL),
	trainer(NULL),
	mmc(NULL),
	mm_module(NULL),
	mm_destroy(NULL)
{
	// load the file into memory
	ifstream in(file, ios::in|ios::binary|ios::ate);
	if( ! in.is_open() ){
		cerr << "Error opening " << file << " for binary input." << endl;
		throw InvalidRomException;
		return;
	}
	ifstream::pos_type file_size = in.tellg();
	in.seekg(0, ios::beg);
	in.read(&cart_data, sizeof(cart_data));

	// check NES header
	if( !(	cart_data.id[0] == 'N' && cart_data.id[1] == 'E' &&
			cart_data.id[2] == 'S' && cart_data.id[3] == 0x1a ) )
	{
		cerr << file << " does not look like an iNES format ROM." << endl;
		throw InvalidRomException;
		return;
	}

	// do a simple checksum
	ifstream::pos_type check_size = 
		sizeof(cart_data) + cart_data.num_rvom_banks * 8 * 1024 +
		(cart_data.trainer ? 512 : 0) + cart_data.rum_rom_banks * 16 * 1024;
	if( file_size != check_size ) {
		cerr << "File checksum failed for " << file << "." << endl;
		throw InvalidRomException;
		return;
	}
	
	// set up the memory mapper
	byte mem_mapper = 
		(cart_data.rom_mapper_high << 4) | cart_data.rom_mapper_low;
	// determine the file name of the library
	string mm_plugin = "./mappers/mapper_" + mem_mapper + G_MODULE_SUFFIX;
	// load the plugin
	mm_module = g_module_open(mm_plugin, G_MODULE_BIND_LAZY);
	// validate and bind the factory methods
	MemoryMapper (*mm_create)();
	if( ! mm_module || 
		// factory method create
		! g_module_symbol( mm_module, "create", (gpointer *) &mm_create) ||
		mm_create == NULL ||
		// factory method destroy
		! g_module_symbol( mm_module, "destroy", (gpointer *) &mm_destory) ||
		mm_destroy == NULL ||
		// create the MemoryMapper
		(mmc = mm_create()) == NULL)
	{
		cerr 	<< "Error loading " << mm_plugin 
				<< " or unable to bind factory methods create and destroy."
				<< endl;
		throw MissingMapperException;
		return;
	}

	// load the trainer
	if( cart_data.trainer ){
		trainer = new byte[trainer_size];
		in.read(trainer, trainer_size);
	}

	// configure for PAL or NTSC
	if( cart_data.screen_type ){
		// PAL
		
	} else {
		// NTSC

	}

	// create a 6502 emulator

}
// attach the screen to surface
Nes::Nes(string &file, SDL_Surface* surface){
	Nes(file);
	attachScreen(surface);
}
// destructor
Nes::~Nes(){
	if( prg_banks != NULL ) delete[] prg_banks;
	if( chr_banks != NULL ) delete[] chr_banks;
	if( trainer != NULL ) delete[] trainer;

	if( mm_destroy != NULL ) mm_destroy(); // this replaces delete mmc
	if( mm_module != NULL )	g_module_close(mm_module);
}

// run the cpu for cycles cycles, returns how many it actually ran,
// which could be greater than cycles by up to 3 cycles
Nes::ulong runCycles(ulong cycles){

}
// run the cpu until a specific cycle count, returns the current cycle
// count. could be greater than stop_cycle by up to 3
Nes::ulong runUntil(ulong stop_cycle){

}
// return the current cycle count since the emulation started
Nes::ulong cycleCount(){

}

// input joystick data. player_index - 0: player 1, 1: player 2.
Nes::void input(int player_index, JoyConfig &cfg){

}
// input joystick data at a specific CPU cycle
Nes::void input(int input_cycle, int player_index, JoyConfig &cfg){

}
// input a set of data which maps cycles to joystick configurations
Nes::void inputMap(JoyInputMap &data){

}

// detach the screen and stop emulating video and audio
Nes::void detachScreen(){

}
// (re?)attach a screen (video and sound) to emulation
Nes::void attachScreen(SDL_Surface* surface){

}
