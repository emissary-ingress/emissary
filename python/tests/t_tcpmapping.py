import json

from kat.harness import Query, Test, variants

from abstract_tests import AmbassadorTest, ServiceType, HTTP
from kat.utils import namespace_manifest

# An AmbassadorTest subclass will actually create a running Ambassador.
# "self" in this class will refer to the Ambassador.

class TCPMappingTest(AmbassadorTest):
    # single_namespace = True
    namespace = "tcp-namespace"
    extra_ports = [ 6789, 7654, 8765, 9876 ]

    # If you set debug = True here, the results of every Query will be printed
    # when the test is run.
    # debug = True

    target1: ServiceType
    target2: ServiceType
    target3: ServiceType

    # init (not __init__) is the method that initializes a KAT Node (including
    # Test, AmbassadorTest, etc.).

    def init(self):
        self.target1 = HTTP(name="target1")
        # print("TCP target1 %s" % self.target1.namespace)

        self.target2 = HTTP(name="target2", namespace="other-namespace")
        # print("TCP target2 %s" % self.target2.namespace)

        self.target3 = HTTP(name="target3")
        # print("TCP target3 %s" % self.target3.namespace)

    # manifests returns a string of Kubernetes YAML that will be applied to the
    # Kubernetes cluster before running any tests.

    def manifests(self) -> str:
        return namespace_manifest("tcp-namespace") + namespace_manifest("other-namespace") + """
---
apiVersion: v1
kind: Secret
metadata:
  name: supersecret
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURnRENDQW1pZ0F3SUJBZ0lKQUlIWTY3cFNoZ3NyTUEwR0NTcUdTSWIzRFFFQkN3VUFNRlV4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVFzd0NRWURWUVFLREFKRQpWekViTUJrR0ExVUVBd3dTZEd4ekxXTnZiblJsZUhRdGFHOXpkQzB5TUI0WERURTRNVEV3TVRFME1EUXhObG9YCkRUSTRNVEF5T1RFME1EUXhObG93VlRFTE1Ba0dBMVVFQmhNQ1ZWTXhDekFKQmdOVkJBZ01BazFCTVE4d0RRWUQKVlFRSERBWkNiM04wYjI0eEN6QUpCZ05WQkFvTUFrUlhNUnN3R1FZRFZRUUREQkowYkhNdFkyOXVkR1Y0ZEMxbwpiM04wTFRJd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUURjQThZdGgvUFdhT0dTCm9ObXZFSFoyNGpRN1BLTitENG93TEhXZWl1UmRtaEEwWU92VTN3cUczVnFZNFpwbFpBVjBQS2xELysyWlNGMTQKejh3MWVGNFFUelphWXh3eTkrd2ZITmtUREVwTWpQOEpNMk9FYnlrVVJ4VVJ2VzQrN0QzMEUyRXo1T1BseG1jMApNWU0vL0pINUVEUWhjaURybFlxZTFTUk1SQUxaZVZta2FBeXU2TkhKVEJ1ajBTSVB1ZExUY2grOTBxK3Jkd255CmZrVDF4M09UYW5iV2pub21FSmU3TXZ5NG12dnFxSUh1NDhTOUM4WmQxQkdWUGJ1OFYvVURyU1dROXpZQ1g0U0cKT2FzbDhDMFhtSDZrZW1oUERsRC9UdjB4dnlINXE1TVVjSGk0bUp0Titnem9iNTREd3pWR0VqZWY1TGVTMVY1RgowVEFQMGQrWEFnTUJBQUdqVXpCUk1CMEdBMVVkRGdRV0JCUWRGMEdRSGRxbHRoZG5RWXFWaXVtRXJsUk9mREFmCkJnTlZIU01FR0RBV2dCUWRGMEdRSGRxbHRoZG5RWXFWaXVtRXJsUk9mREFQQmdOVkhSTUJBZjhFQlRBREFRSC8KTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFBbUFLYkNsdUhFZS9JRmJ1QWJneDBNenV6aTkwd2xtQVBiOGdtTwpxdmJwMjl1T1ZzVlNtUUFkZFBuZEZhTVhWcDFaaG1UVjVDU1F0ZFgyQ1ZNVyswVzQ3Qy9DT0Jkb1NFUTl5akJmCmlGRGNseG04QU4yUG1hR1FhK3hvT1hnWkxYZXJDaE5LV0JTWlIrWktYTEpTTTlVYUVTbEhmNXVuQkxFcENqK2oKZEJpSXFGY2E3eElGUGtyKzBSRW9BVmMveFBubnNhS2pMMlV5Z0dqUWZGTnhjT042Y3VjYjZMS0pYT1pFSVRiNQpINjhKdWFSQ0tyZWZZK0l5aFFWVk5taWk3dE1wY1UyS2pXNXBrVktxVTNkS0l0RXEyVmtTZHpNVUtqTnhZd3FGCll6YnozNFQ1MENXbm9HbU5SQVdKc0xlVmlPWVUyNmR3YkFXZDlVYitWMDFRam43OAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2d0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktrd2dnU2xBZ0VBQW9JQkFRRGNBOFl0aC9QV2FPR1MKb05tdkVIWjI0alE3UEtOK0Q0b3dMSFdlaXVSZG1oQTBZT3ZVM3dxRzNWcVk0WnBsWkFWMFBLbEQvKzJaU0YxNAp6OHcxZUY0UVR6WmFZeHd5OSt3ZkhOa1RERXBNalA4Sk0yT0VieWtVUnhVUnZXNCs3RDMwRTJFejVPUGx4bWMwCk1ZTS8vSkg1RURRaGNpRHJsWXFlMVNSTVJBTFplVm1rYUF5dTZOSEpUQnVqMFNJUHVkTFRjaCs5MHErcmR3bnkKZmtUMXgzT1RhbmJXam5vbUVKZTdNdnk0bXZ2cXFJSHU0OFM5QzhaZDFCR1ZQYnU4Vi9VRHJTV1E5ellDWDRTRwpPYXNsOEMwWG1INmtlbWhQRGxEL1R2MHh2eUg1cTVNVWNIaTRtSnROK2d6b2I1NER3elZHRWplZjVMZVMxVjVGCjBUQVAwZCtYQWdNQkFBRUNnZ0VCQUk2U3I0anYwZForanJhN0gzVnZ3S1RYZnl0bjV6YVlrVjhZWUh3RjIyakEKbm9HaTBSQllIUFU2V2l3NS9oaDRFWVM2anFHdkptUXZYY3NkTldMdEJsK2hSVUtiZVRtYUtWd2NFSnRrV24xeQozUTQwUytnVk5OU2NINDRvYUZuRU0zMklWWFFRZnBKMjJJZ2RFY1dVUVcvWnpUNWpPK3dPTXc4c1plSTZMSEtLCkdoOENsVDkrRGUvdXFqbjNCRnQwelZ3cnFLbllKSU1DSWFrb2lDRmtIcGhVTURFNVkyU1NLaGFGWndxMWtLd0sKdHFvWFpKQnlzYXhnUTFRa21mS1RnRkx5WlpXT01mRzVzb1VrU1RTeURFRzFsYnVYcHpUbTlVSTlKU2lsK01yaAp1LzVTeXBLOHBCSHhBdFg5VXdiTjFiRGw3Sng1SWJyMnNoM0F1UDF4OUpFQ2dZRUE4dGNTM09URXNOUFpQZlptCk9jaUduOW9STTdHVmVGdjMrL05iL3JodHp1L1RQUWJBSzhWZ3FrS0dPazNGN1krY2txS1NTWjFnUkF2SHBsZEIKaTY0Y0daT1dpK01jMWZVcEdVV2sxdnZXbG1nTUlQVjVtbFpvOHowMlNTdXhLZTI1Y2VNb09oenFlay9vRmFtdgoyTmxFeTh0dEhOMUxMS3grZllhMkpGcWVycThDZ1lFQTUvQUxHSXVrU3J0K0dkektJLzV5cjdSREpTVzIzUTJ4CkM5ZklUTUFSL1Q4dzNsWGhyUnRXcmlHL3l0QkVPNXdTMVIwdDkydW1nVkhIRTA5eFFXbzZ0Tm16QVBNb1RSekMKd08yYnJqQktBdUJkQ0RISjZsMlFnOEhPQWovUncrK2x4bEN0VEI2YS8xWEZIZnNHUGhqMEQrWlJiWVZzaE00UgpnSVVmdmpmQ1Y1a0NnWUVBMzdzL2FieHJhdThEaTQ3a0NBQ3o1N3FsZHBiNk92V2d0OFF5MGE5aG0vSmhFQ3lVCkNML0VtNWpHeWhpMWJuV05yNXVRWTdwVzR0cG5pdDJCU2d1VFlBMFYrck8zOFhmNThZcTBvRTFPR3l5cFlBUkoKa09SanRSYUVXVTJqNEJsaGJZZjNtL0xnSk9oUnp3T1RPNXFSUTZHY1dhZVlod1ExVmJrelByTXUxNGtDZ1lCbwp4dEhjWnNqelVidm5wd3hTTWxKUStaZ1RvZlAzN0lWOG1pQk1POEJrclRWQVczKzFtZElRbkFKdWRxTThZb2RICmF3VW03cVNyYXV3SjF5dU1wNWFadUhiYkNQMjl5QzVheFh3OHRtZlk0TTVtTTBmSjdqYW9ydGFId1pqYmNObHMKdTJsdUo2MVJoOGVpZ1pJU1gyZHgvMVB0ckFhWUFCZDcvYWVYWU0wVWtRS0JnUUNVbkFIdmRQUGhIVnJDWU1rTgpOOFBEK0t0YmhPRks2S3MvdlgyUkcyRnFmQkJPQWV3bEo1d0xWeFBLT1RpdytKS2FSeHhYMkcvREZVNzduOEQvCkR5V2RjM2ZCQWQ0a1lJamZVaGRGa1hHNEFMUDZBNVFIZVN4NzNScTFLNWxMVWhPbEZqc3VPZ0NKS28wVlFmRC8KT05paDB6SzN5Wmc3aDVQamZ1TUdGb09OQWc9PQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==
""" + super().manifests()

    # config() must _yield_ tuples of Node, Ambassador-YAML where the
    # Ambassador-YAML will be annotated onto the Node.

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: {self.name}-tlscontext
hosts:
- tls-context-host-1
- tls-context-host-2
- tls-context-host-3
secret: supersecret
""")

        yield self.target1, self.format("""
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}
port: 9876
service: {self.target1.path.fqdn}:443
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-local-only
address: 127.0.0.1
port: 8765
service: {self.target1.path.fqdn}:443
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-clear-to-tls
port: 7654
tls: true
service: {self.target2.path.fqdn}:443
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-1
port: 6789
host: tls-context-host-1
service: {self.target1.path.fqdn}:80
""")

        # Host-differentiated.
        yield self.target2, self.format("""
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-2
port: 6789
host: tls-context-host-2
service: {self.target2.path.fqdn}
tls: {self.name}-tlscontext
""")

        # Host-differentiated.
        yield self.target3, self.format("""
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-3
port: 6789
host: tls-context-host-3
service: {self.target3.path.fqdn}
tls: true
""")

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without a Host header they can't work.
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))

    # scheme defaults to HTTP; if you need to use HTTPS, have it return
    # "https"...
    def scheme(self):
        return "https"

    # Any Query object yielded from queries() will be run as a test. Also,
    # you can add a keyword argument debug=True to any Query() call and the
    # complete response object will be dumped.

    def queries(self):
        # 0: should hit target1, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=9876),
                    insecure=True)

        # 1: should hit target2, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=7654, scheme='http'),
                    insecure=True)

        # 2: should hit target1 via SNI, and use cleartext
        yield Query(self.url(self.name + "/wtfo/", port=6789),
                    headers={"Host": "tls-context-host-1"},
                    insecure=True,
                    sni=True)

        # 3: should hit target2 via SNI, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=6789),
                    headers={"Host": "tls-context-host-2"},
                    insecure=True,
                    sni=True)

        # 4: should hit target3 via SNI, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=6789),
                    headers={"Host": "tls-context-host-3"},
                    insecure=True,
                    sni=True)

        # 5: should error since port 8765 is bound only to localhost
        yield Query(self.url(self.name + "/wtfo/", port=8765),
                    error=[ 'connection reset by peer', 'EOF', 'connection refused' ],
                    insecure=True)

    # Once in check(), self.results is an ordered list of results from your
    # Queries. (You can also look at self.parent.results if you really want
    # to.)

    def check(self):
        for idx, target, tls_wanted in [
            ( 0, self.target1, True ),
            ( 1, self.target2, True ),
            ( 2, self.target1, False ),
            ( 3, self.target2, True ),
            ( 4, self.target3, True ),
            # ( 5, self.target1 ),
        ]:
            r = self.results[idx]
            wanted_fqdn = target.path.fqdn
            backend_fqdn = target.get_fqdn(r.backend.name)
            tls_enabled = r.backend.request.tls.enabled

            assert backend_fqdn == wanted_fqdn, f'{idx}: backend {backend_fqdn} != expected {wanted_fqdn}'
            assert tls_enabled == tls_wanted, f'{idx}: TLS status {tls_enabled} != wanted {tls_wanted}'
