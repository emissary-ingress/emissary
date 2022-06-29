from typing import Optional, Union

from ipaddress import ip_address, IPv4Address, IPv6Address


class CIDRRange:
    """
    A CIDRRange is an IP address (either v4 or v6) plus a prefix length. It
    corresponds to an envoy.api.v3.core.CidrRange.
    """

    def __init__(self, spec: str) -> None:
        """
        Initialize a CIDRRange from a spec, which can look like any of:

        127.0.0.1 -- an exact IPv4 match
        ::1 -- an exact IPv6 match
        192.168.0.0/16 -- an IPv4 range
        2001:2000::/64 -- an IPv6 range

        If the prefix is not a valid IP address, or if the prefix length
        isn't a valid length for the class of IP address, the CIDRRange
        object will evaluate False, with information about the error in
        self.error.

        :param spec: string specifying the CIDR block in question
        """

        self.error: Optional[str] = None
        self.address: Optional[str] = None
        self.prefix_len: Optional[int] = None

        prefix: Optional[str] = None
        pfx_len: Optional[int] = None
        addr: Optional[Union[IPv4Address, IPv6Address]] = None

        if "/" in spec:
            # CIDR range! Try to separate the address and its length.
            address, lenstr = spec.split("/", 1)

            try:
                pfx_len = int(lenstr)
            except ValueError:
                self.error = f"CIDR range {spec} has an invalid length, ignoring"
                return
        else:
            address = spec

        try:
            addr = ip_address(address)
        except ValueError:
            pass

        if addr is None:
            self.error = f"Invalid IP address {address}"
            return

        if pfx_len is None:
            pfx_len = addr.max_prefixlen
        elif pfx_len > addr.max_prefixlen:
            self.error = f"Invalid prefix length for IPv{addr.version} address {address}/{pfx_len}"
            return

        # Convert the parsed address to a string, so that any normalization
        # appropriate to the IP version can happen.
        self.address = str(addr)
        self.prefix_len = pfx_len

    def __bool__(self) -> bool:
        """
        A CIDRRange will evaluate as True IFF there is no error, the address
        is not None, and the prefix_len is not None.
        """

        return (not self.error) and (self.address is not None) and (self.prefix_len is not None)

    def __str__(self) -> str:
        if self:
            return f"{self.address}/{self.prefix_len}"
        else:
            raise RuntimeError("cannot serialize an invalid CIDRRange!")

    def as_dict(self) -> dict:
        """
        Return a dictionary version of a CIDRRange, suitable for use in
        an Envoy config as an envoy.api.v3.core.CidrRange.
        """

        return {"address_prefix": self.address, "prefix_len": self.prefix_len}
