import os
import subprocess

def make_directories(path) :
    os.makedirs(path, exist_ok = True)

def run_process(*args) :
    process = subprocess.Popen(
        args,
        stdin = subprocess.PIPE,
        stdout = subprocess.PIPE,
        stderr = subprocess.PIPE,
        universal_newlines=True
    )
    cli_stdout, cli_stderr = process.communicate(input = None)
    return_code = process.poll()
    return {
        'stdout' : cli_stdout.strip(),
        'stderr' : cli_stderr.strip()
    }
	
def get_dict_value(dict, key, default = None) :
    if dict != None and key in dict :
        return dict[key]
    return default
