import traceback
import asyncio
from nats.aio.client import Client as NATS
#from nats.aio.errors import ErrConnectionClosed, ErrTimeout, ErrNoServers


# python3 implementation of NATS python client
async def nats(server, subject, msg, loop):
    """
    NATS client implemented via asyncio
    python3 implementation, see
    https://github.com/nats-io/nats.py
    """
    nc = NATS()
    try:
        await nc.connect(server, loop=loop, max_reconnect_attempts=3)
    except Exception as exp:
        print("failed to connect to server: error {}".format(str(exp)))
        traceback.print_exc()
        return
    if isinstance(msg, list):
        for item in msg:
            await nc.publish(subject, item)
    else:
        await nc.publish(subject, msg)
    await nc.close()
