#ifndef _DEBUG_H_
#define _DEBUG_H_

// comment this out for production
#define DEBUG





#ifndef DEBUG
#define ASSERT(x)
#else
#include <iostream>
#define ASSERT(x) \
	if( ! (x) ) \
	{ \
		cout << "ERROR! Assert " << #x << " failed\n" \
			<< " on line " << __LINE__ << "\n" \
			<< " in file " << __FILE__ << "\n"; \
	}

#endif


#endif
