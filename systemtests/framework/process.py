import subprocess

class Process :
    output_stdout = 1
    output_pipe = 2

    def __init__(
            self,
            stdout = None
        ) :
        if stdout == None :
            stdout = Process.output_pipe
        self._stdout_file = None
        if isinstance(stdout, str) :
            self._stdout_file = open(stdout, 'a')
            self._stdout = self._stdout_file
        elif stdout == Process.output_pipe :
            self._stdout = subprocess.PIPE
        self._process = None

    def run(self, *args) :
        self._do_run(*args)
        return self

    def _do_run(self, *args) :
        self._process = subprocess.Popen(
            args,
            stdin = subprocess.PIPE,
            stdout = self._stdout,
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
        if self._stdout_file != None :
            self._stdout_file.close()
    
    # !!!Will block if nothing to read
    def read_stdout(self) :
        if self._process.stdout :
            return self._process.stdout.readline()
        else :
            return ''

