#include "memory_mapper.h"
#include "nes.h"

MemoryMapper::MemoryMapper(Nes * system) :
	sys(system)
{
	initialize();
}
