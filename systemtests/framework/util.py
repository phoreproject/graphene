from . import logger

import os
import sys
import time
import codecs
import json

def write_file(fileName, message) :
    file = codecs.open(fileName, "w", "utf-8")
    file.write(str(message))
    file.close()

def read_file(fileName) :
    file = codecs.open(fileName, "r", "utf-8")
    content = file.read()
    file.close()
    return content

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
	
def get_current_timestamp() :
    return int(time.time())

def encode_json(obj) :
    try :
        return json.dumps(obj, indent = 4)
    except :
        logger.exception('Util.encode_json')
        return None
		
def decode_json(content) :
    try :
        return json.loads(content)
    except :
        logger.exception('Util.decode_json')
        return None
		
def merge_dicts(a, b) :
    if a == None :
        a = {}
    if b == None :
        b = {}
    return {**a, **b}
    
def clone_dict(a) :
    return merge_dicts(a, None)

def make_local_address(port) :
    return '/ip4/127.0.0.1/tcp/%d' % (port)
    
