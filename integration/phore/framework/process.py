import subprocess
import threading
import select
import time
import queue
import logging


_sentinel = object()


class Process(threading.Thread):
    output_stdout = 1
    output_pipe = 2

    def __init__(self, process_name="none", *args):
        self.stdout = None
        self.stderr = None
        self.stop_queue = queue.Queue()
        self.args = args
        self.stop_lock = threading.Lock()
        self.stop_lock.acquire()
        self.process_name = process_name
        self._queue = queue.Queue()
        self.logger = logging.getLogger(self.process_name)
        threading.Thread.__init__(self, name=process_name.title())

    def signal_stop(self):
        self._queue.put(_sentinel)

    def run(self):
        process = subprocess.Popen(
            self.args,
            universal_newlines=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=0
        )

        while True:
            try:
                if self._queue.get_nowait() is _sentinel:
                    process.terminate()
            except queue.Empty as e:
                pass

            more_to_read = True
            while more_to_read and process.poll() is None:
                rlist, _, _ = select.select([process.stdout.fileno(), process.stderr.fileno()], [], [], 0.1)
                if len(rlist) == 0:
                    more_to_read = False
                for descriptor in rlist:
                    if descriptor == process.stdout.fileno():
                        read = process.stdout.readline()
                        if read:
                            self.logger.info(read.strip())

                    if descriptor == process.stderr.fileno():
                        read = process.stderr.readline()
                        if read:
                            self.logger.info(read.strip())

            if process.poll() is not None:
                break

            time.sleep(0.1)
