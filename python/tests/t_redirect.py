from kat.harness import Query, EDGE_STACK

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import ServiceType
from kat.utils import namespace_manifest


#####
# XXX This file is annoying.
#
# RedirectTestsWithProxyProto and RedirectTestsInvalidSecret used to be subclasses of RedirectTests,
# which makes a certain amount of sense. Problem is that when I wanted to modify just RedirectTests
# to have secrets defined, that ended up affecting the two subclasses in bad ways. There's basically
# no way to subclass an AmbassadorTest without having your base class be run separately, which isn't
# what I wanted here. Sigh.

class RedirectTests(AmbassadorTest):
    target: ServiceType
    edge_stack_cleartext_host = False

    def init(self):
        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

        self.target = HTTP()

    def requirements(self):
        # only check https urls since test readiness will only end up barfing on redirect
        yield from (r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("https"))

    def manifests(self):
        return namespace_manifest("redirect-namespace") + """
---
apiVersion: v1
kind: Secret
metadata:
  name: redirect-cert
  namespace: redirect-namespace
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
---
apiVersion: v1
kind: Secret
metadata:
  name: redirect-cert
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
""" + super().manifests()

    def config(self):
        # Use self here, not self.target, because we want the TLS module to
        # be annotated on the Ambassador itself.
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: Module
name: tls
ambassador_id: {self.ambassador_id}
config:
  server:
    enabled: True
    secret: redirect-cert
    redirect_cleartext_from: 8080
""")

        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # [0]
        yield Query(self.url("tls-target/", scheme="http"), expected=301)

        # [1] -- PHASE 2
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors",
                             scheme="https"),
                    insecure=True,
                    phase=2)

    def check(self):
        # For query 0, check the redirection target.
        assert len(self.results[0].headers['Location']) > 0 
        assert self.results[0].headers['Location'][0].find('/tls-target/') > 0

        # For query 1, we require no errors.
        # XXX Ew. If self.results[1].json is empty, the harness won't convert it to a response.
        errors = self.results[1].json
        assert(len(errors) == 0)

class RedirectTestsWithProxyProto(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def requirements(self):
        # only check https urls since test readiness will only end up barfing on redirect
        yield from (r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("https"))

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Module
name:  ambassador
config:
  use_proxy_proto: true
  enable_ipv6: true
""")

        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # TODO (concaf): FWIW, this query only covers one side of the story. This tests that this is the correct
        #  deviation from the normal behavior (301 response), but does not test a 301 when proxy proto is actually sent.
        #  This is because net/http does not yet support adding proxy proto to HTTP requests, and hence it's difficult
        #  to test with kat. We will need to open a raw TCP connection (e.g. telnet/nc) and send the entire HTTP Request
        #  in plaintext to test this behavior (or use curl with --haproxy-protocol).
        yield Query(self.url("tls-target/"), error=[ "EOF", "connection reset by peer" ])

    # We can't do the error check until we have the PROXY client mentioned above.
    #     # [1] -- PHASE 2
    #     yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)
    #
    # def check(self):
    #     # We don't have to check anything about query 0, the "expected" clause is enough.
    #
    #     # For query 1, we require no errors.
    #     # XXX Ew. If self.results[1].json is empty, the harness won't convert it to a response.
    #     errors = self.results[1].json
    #     assert(len(errors) == 0)


class RedirectTestsInvalidSecret(AmbassadorTest):
    """
    This test tests that even if the specified secret is invalid, the rest of TLS Context should
    go through. In this case, even though the secret does not exist, redirect_cleartext_from
    should still take effect.
    """

    target: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

        self.target = HTTP()

    def requirements(self):
        # only check https urls since test readiness will only end up barfing on redirect
        yield from (r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("https"))

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: Module
name: tls
ambassador_id: {self.ambassador_id}
config:
  server:
    enabled: True
    secret: does-not-exist-secret
    redirect_cleartext_from: 8080
""")

        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # [0]
        yield Query(self.url("tls-target/"), expected=301)

    # There's kind of no way to do this. Looks like we need to speak HTTP to the port on which we
    # think the server is listening for HTTPS? This is a bad config all the way around, really.
    #     # [1] -- PHASE 2
    #     yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors", scheme="https"), phase=2)
    #
    # def check(self):
    #     # We don't have to check anything about query 0, the "expected" clause is enough.
    #
    #     # For query 1, we require no errors.
    #     # XXX Ew. If self.results[1].json is empty, the harness won't convert it to a response.
    #     errors = self.results[1].json
    #     assert(len(errors) == 0)


class XFPRedirect(AmbassadorTest):
    parent: AmbassadorTest
    target: ServiceType
    edge_stack_cleartext_host = False

    def init(self):
        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

        self.target = HTTP()

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
config:
  x_forwarded_proto_redirect: true
  use_remote_address: false
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # [0]
        yield Query(self.url(self.name + "/target/"), headers={ "X-Forwarded-Proto": "http" }, expected=301)

        # [1]
        yield Query(self.url(self.name + "/target/"), headers={ "X-Forwarded-Proto": "https" }, expected=200)

        # [2] -- PHASE 2
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), headers={ "X-Forwarded-Proto": "https" }, phase=2)

    def check(self):
        # For query 0, check the redirection target.
        expected_location = ["https://" + self.path.fqdn + "/" + self.name + "/target/"]
        actual_location = self.results[0].headers['Location']
        assert actual_location == expected_location, "Expected redirect location to be {}, got {} instead".format(
            expected_location,
            actual_location
        )

        # For query 1, we don't have to check anything, the "expected" clause is enough.

        # For query 2, we require no errors.
        # XXX Ew. If self.results[2].json is empty, the harness won't convert it to a response.
        errors = self.results[2].json
        assert(len(errors) == 0)

    def requirements(self):
        # We're replacing super()'s requirements deliberately here: we need the XFP header or they can't work.
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"X-Forwarded-Proto": "https"}))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"X-Forwarded-Proto": "https"}))

