#ifndef DEBUG_H
#define DEBUG_H

enum LogVerbosity {
	lvSilent, // nothing is logged
	lvNormal, // things that should print under "normal" circumstances
	lvDebug, // things that are useful to know when debugging
	lvRaw //excess noise
};

// if you're going to crash, do it early and noisily.
void crashIf(int value, char* msg, char* param); 

// log to a certain verbosity level. See LogVerbosity enum
void logg(int level, char* msg);

// set the log level
void setLogLevel(int newLevel);
#endif
