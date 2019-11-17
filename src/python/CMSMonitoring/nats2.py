import traceback
# python2 NATS implementation via tornado module
import tornado.gen
from nats.io.client import Client as NATS


# python2 implementation of NATS python client
@tornado.gen.coroutine
def nats(server, subject, msg):
    """
    NATS client implemented via tornado (NATS py2 approach), see
    https://github.com/nats-io/nats.py2
    """
    nc = NATS()
    try:
        yield nc.connect(server, max_reconnect_attempts=3)
    except Exception as exp:
        print("failed to connect to server: error {}".format(str(exp)))
        traceback.print_exc()
        return
    if isinstance(msg, list):
        for item in msg:
            yield nc.publish(subject, item)
    else:
        yield nc.publish(subject, msg)

    # Drain gracefully closes the connection, allowing all subscribers to
    # handle any pending messages inflight that the server may have sent.
    yield nc.drain()
    # Drain works async in the background
    #yield tornado.gen.sleep(1)
    yield nc.close()
