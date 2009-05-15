#ifndef _NES_EXCEPTIONS_H_
#define _NES_EXCEPTIONS_H_

#include <exception>

class InvalidRomException: public exception {
	virtual const char* what() const throw() {
		return "The ROM file is invalid and cannot be emulated.";
	}
} InvalidRomException;

class RomEmulationException: public exception {
	virtual const char* what() const throw() {
		return	"An error ocurred while emulating the ROM. "
				"Emulation is in an invalid state and cannot continue.";
	}
} RomEmulationException;

class MissingMapperException: public exception {
	virtual const char* what() const throw() {
		return	"The ROM you tried to emulate has a memory mapper for "
				"which there is not a working plugin.";
	}
} MissingMapperException;

#endif
