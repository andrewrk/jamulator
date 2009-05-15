#include <string>
#include <sstream>

#include "utility.h"

// remove leading and trailing spaces from a string
void trim(string& str, const string& what_to_trim){
	string::size_type pos = str.find_last_not_of(what_to_trim);
	if( pos != string::npos ){
		str.erase(pos + 1);
		pos = str.find_first_not_of(what_to_trim);
		if( pos != string::npos ) str.erase(0, pos);
	} else {
		str.erase(str.begin(), str.end());
	}
}

// split a string by delimiters into tokens
void tokenize(	const string& str, 
				vector<string>& tokens, 
				const string& delimiters)
{
	// skip delimiters at beginning
	string::size_type lastPos = str.find_first_not_of(delimiters, 0);

	// find first non-delimiter
	string::size_type pos = str.find_first_of(delimiters, lastPos);

	while( string::npos != pos || string::npos != lastPos){
		// found a token, add it to the vector
		tokens.push_back(str.substr(lastPos, pos - lastPos));

		// skip delimiters
		lastPos = str.find_first_not_of(delimiters, pos);

		// find next non-delimiter
		pos = str.find_first_of(delimiters, lastPos);
	}
}


// replace occurences in a string with another string
void replace(string& str, const string& find, const string& replacement){
	string::size_type pos = str.find(find);
	while( pos != string::npos ){
		str.replace( pos, find.size(), replacement );
		pos = str.find(find, pos+1);
	}
}

// convert a string representation of a hex number to an int
unsigned int hexToInt(const string& str){
	unsigned int ret;

	std::istringstream iss(str);
	iss >> std::hex >> ret; 

	return ret;
}


string intToHex(unsigned int i){
	string ret;
	std::stringstream ss;
	ss << std::hex << i;
	ss >> ret;
	return ret;
}

// convert a string to an int
int stringToInt(const string& str){
	int ret;
	std::istringstream iss(str);
	iss >> ret;
	return ret;
}
