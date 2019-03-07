import json

from kat.harness import Query, Test, variants

from abstract_tests import AmbassadorTest, ServiceType, HTTP

class TCPMapping(AmbassadorTest):
    single_namespace = True
    namespace = "tcp-namespace"
    extra_ports = [ 6789, 8765, 9876 ] # 7654

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Namespace
metadata:
  name: tcp-namespace
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

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: {self.name}-tlscontext
hosts:
- tls-context-host-1
- tls-context-host-2
secret: supersecret
""")

    def scheme(self):
        return "https"

    @classmethod
    def variants(cls):
        yield cls(variants(TCPMappingTest))

class TCPMappingTest(Test):

    target1: ServiceType
    target2: ServiceType
    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        yield cls(HTTP(name="target1"), HTTP(name="target2"), name="{self.target1.name}")

    def init(self, target1: ServiceType, target2: ServiceType) -> None:
        self.target1 = target1
        self.target2 = target2

class TCPPortMapping(TCPMappingTest):

    parent: AmbassadorTest
    #
    # @classmethod
    # def variants(cls):
    #     yield cls(HTTP(), name="{self.target.name}")

    def config(self):
        # Any host. Does this even need TLS?
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
# ---
# apiVersion: ambassador/v1
# kind:  TCPMapping
# name:  {self.name}-collider-1
# address: 172.17.0.1 # This needs to be a local, non-loopback, IP address. Hmmmm.
# port: 7654
# service: {self.target1.path.fqdn}:443
# ---
# apiVersion: ambassador/v1
# kind:  TCPMapping
# name:  {self.name}-collider-2
# address: 127.0.0.1
# port: 7654
# service: {self.target2.path.fqdn}:443
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
service: {self.target2.path.fqdn}:80
""")

    # def requirements(self):
    #     yield Query(self.parent.url(self.name + "/wtfo", scheme='https'))

    def queries(self):
        # 0: should hit target1
        yield Query(self.parent.url(self.name + "/wtfo/", port=9876),
                    insecure=True)

        # 1: should hit target1 via SNI
        yield Query(self.parent.url(self.name + "/wtfo/", port=6789),
                    headers={"Host": "tls-context-host-1"},
                    insecure=True,
                    sni=True)

        # 2: should hit target2 via SNI
        yield Query(self.parent.url(self.name + "/wtfo/", port=6789),
                    headers={"Host": "tls-context-host-2"},
                    insecure=True,
                    sni=True)

        # 3: should error since port 8765 is bound only to localhost
        yield Query(self.parent.url(self.name + "/wtfo/", port=8765),
                    error=['connection reset by peer', 'EOF'],
                    insecure=True)

        # # 4: should hit target1, since the target2 route is localhost-only
        # yield Query(self.parent.url(self.name + "/wtfo/", port=7654),
        #             insecure=True)


    def check(self):
        for idx, wanted in [
            ( 0, self.target1.path.fqdn ),
            ( 1, self.target1.path.fqdn ),
            ( 2, self.target2.path.fqdn )
            # ( 4, self.target1.path.fqdn),
        ]:
            r = self.results[idx]
            backend_fqdn = self.parent.get_fqdn(r.backend.name)

            assert backend_fqdn == wanted, f'{idx}: backend {backend_fqdn} != expected {wanted}'
