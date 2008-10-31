#include <stdio.h>
#include <stdlib.h>

#include "debug.h"

int log_level = lvNormal;
FILE* log_fh = NULL;

void crashIf(int value, char* msg, char* param){
	if( value){
		if( param == NULL)
			fprintf(stderr, "crash: %s\n", msg);
		else
			fprintf(stderr, "crash: %s - %s\n", msg, param);

		exit(1);
	}
}

void logg(int level, char* msg){
	if( log_fh == NULL)
		log_fh = stdout;
	if( level >= log_level )
		fprintf(log_fh, msg);
	
}

void setLogLevel(int newLevel){
	log_level = newLevel;
}
