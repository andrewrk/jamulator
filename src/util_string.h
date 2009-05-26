/* 
	global utility functions

*/

#ifndef _UTILITY_H_
#define _UTILITY_H_

#include <string>
#include <vector>
using namespace std;

// remove leading and trailing spaces from a string
void trim(string& str, const string& what_to_trim = " \t\n\r");

// split a string by delimiters into tokens
void tokenize(const string& str, 
	vector<string>& tokens, const string& delimiters = " \t\n\r");

// replace occurences in a string with another string
void replace(string& str, const string& find, const string& replacement);

// convert a string representation of a hex number to an int and vice versa
unsigned int hexToInt(const string& str);
string intToHex(unsigned int i);

// convert a string to an int
int stringToInt(const string& str);

// convert an int to a string
string intToString(int x);

#endif
