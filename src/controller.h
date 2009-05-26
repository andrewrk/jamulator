/*
	controller.h -	structure containing controller configuration data
					and presets.

*/

#ifndef _CONTROLLER_H_
#define _CONTROLLER_H_

class Controller {
	public:
		typedef enum {
			BtnA = 0x01,
			BtnB = 0x02,
			BtnSelect = 0x04,
			BtnStart = 0x08,
			BtnUp = 0x10,
			BtnDown = 0x20,
			BtnLeft = 0x40,
			BtnRight = 0x80
		} Status;
};




#endif
