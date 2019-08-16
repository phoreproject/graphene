from . import logger

def assert_true(b, message = None, exception = True) :
    if not b :
        msg = 'Assert failure: condition should be true. '
        if message != None :
            msg += message
        if exception :
            raise AssertionError(msg)
        else :
            logger.error(msg)
