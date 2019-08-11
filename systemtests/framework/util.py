import os
import sys
import time

def die(message = None) :
    if message != None :
        print(message)
    sys.exit(1)

def make_directories(path) :
    os.makedirs(path, exist_ok = True)

def get_dict_value(dict, key, default = None) :
    if dict != None and key in dict :
        return dict[key]
    return default

def sleep_for_seconds(seconds) :
	if seconds < 0 :
		seconds = 0
	time.sleep(seconds)

def sleep_for_milliseconds(milliseconds) :
	if milliseconds < 0 :
		milliseconds = 0
	time.sleep(milliseconds * 0.001)
	
