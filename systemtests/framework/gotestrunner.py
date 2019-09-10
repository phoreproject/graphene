from .process import Process

class GoTestRunner :
    def __init__(self, goTestExecutable, peers, commands, genesis) :
        self._goTestExecutable = goTestExecutable
        self._peers = peers
        self._commands = commands
        self._genesis = genesis

    def run(self) :
        process = Process(stdout = Process.output_stdout).run(
            self._goTestExecutable,
            '-connect',
            ','.join(self._peers),
            '-commands',
            ','.join(self._commands),
            '-genesis',
            self._genesis,
        )
        process.wait_until_finish()
