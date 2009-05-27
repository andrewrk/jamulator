SUBDIRS = mappers
noinst_LTLIBRARIES = libnes.la
libnes_la_SOURCES = cpu_6502.cpp nes.cpp util_string.cpp controller.h cpu_6502.h debug.h memory_mapper.h nes_exceptions.h nes.h util_string.h
libnes_la_CPPFLAGS = $(GMODULE_CFLAGS) $(SDL_CFLAGS)
libnes_la_LIBADD = $(GMODULE_LIBS) $(SDL_LIBS)

