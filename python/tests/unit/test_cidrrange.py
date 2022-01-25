import logging
import sys

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador.envoy.v3.v3cidrrange import CIDRRange

@pytest.mark.compilertest
def test_cidrrange():
    for spec, wanted_result, wanted_address, wanted_prefix_len, wanted_error in [
        ( "127.0.0.1",              True,   "127.0.0.1",    32,     None ), # IPv4 exact
        ( "::1",                    True,   "::1",          128,    None ), # IPv6 exact
        ( "192.168.0.0/16",         True,   "192.168.0.0",  16,     None ), # IPv4 range
        ( "2001:2000::/64",         True,   "2001:2000::",  64,     None ), # IPv6 range
        ( "2001:2000:0:0:0::/64",   True,   "2001:2000::",  64,     None ), # IPv6 range
        ( "10",                     False,  None,           None,   "Invalid IP address 10" ),
        ( "10/8",                   False,  None,           None,   "Invalid IP address 10" ),
        ( "10.0.0.0/a",             False,  None,           None,   "CIDR range 10.0.0.0/a has an invalid length, ignoring" ),
        ( "10.0.0.0/99",            False,  None,           None,   "Invalid prefix length for IPv4 address 10.0.0.0/99" ),
        ( "2001:2000::/99",         True,   "2001:2000::",  99,     None ),
        ( "2001:2000::/199",        False,  None,           None,   "Invalid prefix length for IPv6 address 2001:2000::/199" )
    ]:
        c = CIDRRange(spec)

        if wanted_result:
            assert bool(c), f"{spec} should be a valid CIDRRange but is not: {c.error}"

            assert c.address == wanted_address
            assert c.prefix_len == wanted_prefix_len
            assert c.error == None
        else:
            assert not bool(c), f"{spec} should be an invalid CIDRRange but is valid? {c}"

            assert c.address == None
            assert c.prefix_len == None
            assert c.error == wanted_error

if __name__ == '__main__':
    pytest.main(sys.argv)
