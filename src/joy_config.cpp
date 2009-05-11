#include "joy_config.h"

JoyConfig JCfgFromByte(unsigned char byte) {
	JoyConfig ret;
	ret.a = byte&0x01;
	ret.b = byte&0x02;
	ret.select = byte&0x04;
	ret.start = byte&0x08;
	ret.up = byte&0x10;
	ret.down = byte&0x20;
	ret.left = byte&0x40;
	ret.right = byte&0x80;
	return ret;
}
