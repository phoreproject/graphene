import subprocess

class Process :
    def __init__(
            self,
            capture_stdout = True
        ) :
        self._capture_stdout = capture_stdout
        self._process = None

    def run(self, *args) :
        self._do_run(*args)
        return self

    def _do_run(self, *args) :
        stdout = None
        if self._capture_stdout :
            stdout = subprocess.PIPE
        self._process = subprocess.Popen(
            args,
            stdin = subprocess.PIPE,
            stdout = stdout,
            stderr = subprocess.STDOUT,
            universal_newlines = True,
            bufsize = 0
        )

    def wait_until_finish(self) :
        return self._process.communicate()
        
    def kill(self) :
        if self._process != None :
            self._process.kill()
        self._process = None
    
    # !!!Will block if nothing to read
    def read_stdout(self) :
        if self._process.stdout :
            return self._process.stdout.readline()
        else :
            return ''

