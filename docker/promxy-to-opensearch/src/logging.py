import logging
import src.constants as const

APP_NAME = "promxy_to_opensearch"

def _get_logger() -> logging.Logger:
    logger = logging.getLogger(APP_NAME)
    logger.setLevel(const.LOG_LEVEL)

    # TODO: Look into redirecting full logs to logstash or some other cloud tool
    fh = logging.FileHandler(f'{__name__}.log')
    fh.setLevel(logging.DEBUG)
    ch = logging.StreamHandler()
    ch.setLevel(const.LOG_LEVEL)
    formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
    fh.setFormatter(formatter)
    ch.setFormatter(formatter)

    logger.addHandler(fh)
    logger.addHandler(ch)
    return logger

logger: logging.Logger = _get_logger()