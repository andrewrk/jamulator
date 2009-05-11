/*
	joy_config.h -	structure containing joystick configuration data
					and presets.

*/

#ifndef _JOYCONFIG_H_
#define _JOYCONFIG_H_

#define JCFG_A 0x01
#define JCFG_B 0x02
#define JCFG_SELECT 0x04
#define JCFG_START 0x08
#define JCFG_UP 0x10
#define JCFG_DOWN 0x20
#define JCFG_LEFT 0x40
#define JCFG_RIGHT 0x80

// for convenience
#define JCFG_A_AND_B 0x03

// the structure for holding joystick configuration
typedef struct {
	bool a;
	bool b;
	bool select;
	bool start;
	bool up;
	bool down;
	bool left;
	bool right;
} JoyConfig;

// turn one of those #defined bytes up there into our structure
JoyConfig JCfgFromByte(unsigned char byte);

#endif
