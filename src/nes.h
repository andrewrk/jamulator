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
#include "SDL.h"
#include "M6502/M6502.h"

#include "joy_config.h"

#define CYCLES_PER_VBLANK 29829.541666667 // nestreme uses 27120


class Nes {
	public:
		typedef ulong unsigned long long int;
		typedef JoyInputMap Vector<ulong, JoyConfig>;

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
		
		// NONE because this is vaporware lolollol
};
