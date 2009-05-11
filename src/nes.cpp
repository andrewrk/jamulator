#include "nes.h"


// create an Nes emulator from .nes file file with no audio/video
Nes::Nes(string &file){

}
// attach the screen to surface
Nes::Nes(string &file, SDL_Surface* surface){

}
// destructor
Nes::~Nes(){

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
