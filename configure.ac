AC_INIT([jamulator], [0.1], [superjoe30@gmail.com])
AM_INIT_AUTOMAKE([-Wall -Werror foreign])

# config for gmodule
PKG_CHECK_MODULES(GMODULE, gmodule-2.0 >= 0.22)
AC_SUBST(GMODULE_CFLAGS)
AC_SUBST(GMODULE_LIBS)

# config for SDL
SDL_CFLAGS=`sdl-config --cflags`
SDL_LIBS=`sdl-config --libs`
AC_SUBST(SDL_CFLAGS)
AC_SUBST(SDL_LIBS)

AC_PROG_CXX
AC_PROG_CC
AC_PROG_RANLIB
AC_PROG_LIBTOOL
AC_CONFIG_HEADERS([config.h])
AC_CONFIG_FILES([
	Makefile
	src/Makefile
	src/mappers/Makefile
])
AC_OUTPUT

