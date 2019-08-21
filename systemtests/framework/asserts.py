from . import logger

def assert_true(b, message = None, exception = True) :
    if b :
        return
    msg = 'Assert failure: condition should be true. '
    if message != None :
        msg += message
    if exception :
        raise AssertionError(msg)
    else :
        logger.error(msg)

def assert_false(b, message = None, exception = True) :
    if not b :
        return
    msg = 'Assert failure: condition should be false. '
    if message != None :
        msg += message
    if exception :
        raise AssertionError(msg)
    else :
        logger.error(msg)

def assert_exception_contain_text(e, text, exception = True) :
    if str(e).find(text) >= 0 :
        return
    msg = 'Assert failure: exception should contain text "%s"' % (text)
    msg += "\nException message:\n%s" % (str(e))
    if exception :
        raise AssertionError(msg)
    else :
        logger.error(msg)

def assert_not_here(message = None, exception = True) :
    msg = 'Assert failure: here should not be reached. '
    if message != None :
        msg += message
    if exception :
        raise AssertionError(msg)
    else :
        logger.error(msg)

