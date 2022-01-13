import logging

logging.basicConfig(
    level=logging.DEBUG,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")
logger.setLevel(logging.DEBUG)

def test_crds_v2():
    logger.info("CRDS: v2")
    assert True
