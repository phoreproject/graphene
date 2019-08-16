import logging
import time
import sys

_logger = None
_logHandler = None
_verbose = False

def _create_logger(fileName = None) :
    global _logger

    _logger = logging.getLogger('synapse')

    _logHandler = logging.StreamHandler(sys.stdout)
    #formatter = logging.Formatter('%(name)s %(asctime)s [%(process)d] %(levelname)s: %(message)s')
    formatter = logging.Formatter('%(asctime)s %(levelname)s: %(message)s', "%H:%M:%S")
    formatter.converter = time.gmtime  # UTC time
    _logHandler.setFormatter(formatter)
    _logger.addHandler(_logHandler)
    
    _logger.setLevel(logging.DEBUG)

def set_Level(level) :    
    loggingLevel = logging.INFO
    if level == 'fatal' or level == 'critical' :
        loggingLevel = logging.CRITICAL
    elif level == 'error' :
        loggingLevel = logging.ERROR
    elif level == 'warning' :
        loggingLevel = logging.WARNING
    elif level == 'info' :
        loggingLevel = logging.INFO
    elif level == 'debug' :
        loggingLevel = logging.DEBUG
    get_logger().setLevel(loggingLevel)
    
def set_verbose(verbose) :
    global _verbose
    _verbose = verbose
    
def allow_log(verbose) :
    global _verbose
    return _verbose or not verbose

def get_logger() :
    global _logger
    if _logger == None :
        _create_logger()
    return _logger

def critical(message, verbose = False) :
    if allow_log(verbose) :
        get_logger().critical(message)

def fatal(message, verbose = False) :
    if allow_log(verbose) :
        get_logger().critical(message)

def error(message, verbose = False) :
    if allow_log(verbose) :
        get_logger().error(message)

def warning(message, verbose = False) :
    if allow_log(verbose) :
        get_logger().warning(message)

def info(message, verbose = False) :
    if allow_log(verbose) :
        get_logger().info(message)

def debug(message, verbose = False) :
    if allow_log(verbose) :
        get_logger().debug(message)

def exception(message, verbose = False) :
    if allow_log(verbose) :
        get_logger().exception(message)
