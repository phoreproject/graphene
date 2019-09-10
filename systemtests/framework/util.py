from . import logger

import os
import sys
import time
import codecs
import json
import random
import re

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

def make_random_hash() :
    return bytes(random.getrandbits(8) for _ in range(32))
    
def revert_hex_string(hex_string) :
    result = ''
    for i in range(0, len(hex_string), 2) :
        result = hex_string[i : i + 2] + result
    return result

def parse_genesis_hash(text) :
    matches = re.search(r'genesisHash\=([\w\d]+)', text)
    if matches == None :
        return ''
    return matches.group(1)

