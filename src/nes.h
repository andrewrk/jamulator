/*
	nes.h - Nes emulator object

	Synopsis:

	// create a new emulator to emulate smb3.nes without a screen
	Nes emu("smb3.nes"); 
	// run emulation for 46 cycles
	emu.runCycles(46);

	// press Start on player 1's controller right now
	JoyConfig cfg(JCFG_START);
	emu.input(0, cfg);
	emu.runCycles(10);

	// press A on player 1's controller at 100 cycles
	cfg = JoyConfig(JCFG_A);
	emu.input(100, 0, cfg);
	emu.runCycles(45);
	
	//provide an input map and emulate it
	Vector<unsigned long long int, JoyConfig> data;
	data.push_back(400, JoyConfig(JCFG_B)); // press B at 400
	data.push_back(450, JoyConfig(JCFG_A_AND_B)); // press A at 450
	data.push_back(620, JoyConfig(JCFG_B)); // let go of A at 620
	data.push_back(700, JoyConfig(JCFG_NONE)); // let go of B at 700
	emu.inputMap(data);
	emu.runUntil(800); // run until absolute cycle count is 800

*/

#include <string>
#include "SDL.h"
#include "M6502/M6502.h"

#include "joy_config.h"

#define CYCLES_PER_VBLANK 29829.541666667 // nestreme uses 27120

#define ulong unsigned long long int


class Nes {
	public:
		// create an Nes emulator from .nes file file with no audio/video
		Nes(string &file);
		// attach the screen to surface
		Nes(string &file, SDL_Surface* surface);
		// destructor
		~Nes();
		
		// run the cpu for cycles cycles
		runCycles(ulong cycles);
		// run the cpu until a specific cycle count
		runUntil(ulong stop_cycle);

		// input joystick data
		input
		
		// detach the screen and stop emulating video and audio
		detachScreen();
		// (re?)attach a screen (video and sound) to emulation
		attachScreen(SDL_Surface* surface);
	private:

};
