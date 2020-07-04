from kat.harness import Query, EDGE_STACK

from abstract_tests import AmbassadorTest, ServiceType, HTTP

# STILL TO ADD:
# Host referencing a Secret in another namespace?
# Mappings without host attributes (infer via Host resource)
# Host where a TLSContext with the inferred name already exists

class HostCRDSingle(AmbassadorTest):
    """
    HostCRDSingle: a single Host with a manually-configured TLS. Since the Host is handling the
    TLSContext, we expect both OSS and Edge Stack to redirect cleartext from 8080 to 8443 here.
    """
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.allow_edge_stack_redirect = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.name.k8s}-secret
  selector:
    matchLabels:
      hostname: {self.path.fqdn}
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.name.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') +  super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True)
        yield Query(self.url("target/", scheme="http"), expected=301)


class HostCRDNo8080(AmbassadorTest):
    """
    HostCRDNo8080: a single Host with manually-configured TLS that explicitly turns off redirection
    from 8080.
    """
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.allow_edge_stack_redirect = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.name.k8s}-secret
  selector:
    matchLabels:
      hostname: {self.path.fqdn}
  requestPolicy:
    insecure:
      additionalPort: -1
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.name.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True)

        if EDGE_STACK:
            yield Query(self.url("target/", scheme="http"), expected=404)
        else:
            yield Query(self.url("target/", scheme="http"), error=[ "EOF", "connection refused" ])


class HostCRDManualContext(AmbassadorTest):
    """
    A single Host with a manually-specified TLS secret and a manually-specified TLSContext,
    too. Since the Host is _not_ handling the TLSContext, we do _not_ expect automatic redirection
    on port 8080.
    """
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.allow_edge_stack_redirect = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: manual-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: manual-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  selector:
    matchLabels:
      hostname: manual-hostname
  tlsSecret:
    name: manual-secret
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: manual-host-context
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hosts:
  - {self.path.fqdn}
  secret: manual-secret
  min_tls_version: v1.2
  max_tls_version: v1.3
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: manual-target-mapping
  labels:
    hostname: manual-hostname
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.0",  maxTLSv="v1.0",
                    error=["tls: server selected unsupported protocol version 303",
                           "tls: no supported versions satisfy MinVersion and MaxVersion",
                           "tls: protocol version not supported"])

        if EDGE_STACK:
            yield Query(self.url("target/", scheme="http"), expected=404)
        else:
            yield Query(self.url("target/", scheme="http"), error=[ "EOF", "connection refused" ])


class HostCRDManualContextDifferentTLSContext(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.allow_edge_stack_redirect = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: manual-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: manual-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  selector:
    matchLabels:
      hostname: manual-hostname
  tlsSecret:
    name: manual-secret
  tlsContext: separate-tls-context
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: separate-tls-context
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: manual-secret
  min_tls_version: v1.2
  max_tls_version: v1.3
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: manual-target-mapping
  labels:
    hostname: manual-hostname
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.0",  maxTLSv="v1.0",
                    error=["tls: server selected unsupported protocol version 303",
                           "tls: no supported versions satisfy MinVersion and MaxVersion",
                           "tls: protocol version not supported"])

        # if EDGE_STACK:
        #     yield Query(self.url("target/", scheme="http"), expected=404)
        # else:
        #     yield Query(self.url("target/", scheme="http"), error=[ "EOF", "connection refused" ])
        #


class HostCRDManualContextTLSConfig(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.allow_edge_stack_redirect = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: manual-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: manual-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  selector:
    matchLabels:
      hostname: manual-hostname
  tlsSecret:
    name: manual-secret
  tls:
    min_tls_version: v1.2
    max_tls_version: v1.3
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: manual-target-mapping
  labels:
    hostname: manual-hostname
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.0",  maxTLSv="v1.0",
                    error=["tls: server selected unsupported protocol version 303",
                           "tls: no supported versions satisfy MinVersion and MaxVersion",
                           "tls: protocol version not supported"])

        # if EDGE_STACK:
        #     yield Query(self.url("target/", scheme="http"), expected=404)
        # else:
        #     yield Query(self.url("target/", scheme="http"), error=[ "EOF", "connection refused" ])


class HostCRDClearText(AmbassadorTest):
    """
    A single Host specifying cleartext only. Since it's just cleartext, no redirection comes
    into play.
    """
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.allow_edge_stack_redirect = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: cleartext-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  selector:
    matchLabels:
      hostname: host-cleartext
  requestPolicy:
    insecure: 
      action: Route
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: cleartext-target-mapping
  labels:
    hostname: host-cleartext
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "http"

    def queries(self):
        yield Query(self.url("target/"), insecure=True)
        yield Query(self.url("target/", scheme="https"),
                    error=[ "EOF", "connection refused" ])


class HostCRDDouble(AmbassadorTest):
    """
    HostCRDDouble: two Hosts with manually-configured TLS secrets, and Mappings specifying host matches.
    Since the Hosts are handling TLSContexts, we expect both OSS and Edge Stack to redirect cleartext
    from 8080 to 8443 here.

    XXX In the future, the hostname matches should be unnecessary.
    """
    target1: ServiceType
    target2: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.allow_edge_stack_redirect = False
        self.target1 = HTTP(name="target1")
        self.target2 = HTTP(name="target2")

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
kind: Secret
metadata:
  name: test-tlscontext-secret-0
  labels:
    kat-ambassador-id: hostdouble
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURnRENDQW1pZ0F3SUJBZ0lKQUpycUl0ekY2MTBpTUEwR0NTcUdTSWIzRFFFQkN3VUFNRlV4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVFzd0NRWURWUVFLREFKRQpWekViTUJrR0ExVUVBd3dTZEd4ekxXTnZiblJsZUhRdGFHOXpkQzB4TUI0WERURTRNVEV3TVRFek5UTXhPRm9YCkRUSTRNVEF5T1RFek5UTXhPRm93VlRFTE1Ba0dBMVVFQmhNQ1ZWTXhDekFKQmdOVkJBZ01BazFCTVE4d0RRWUQKVlFRSERBWkNiM04wYjI0eEN6QUpCZ05WQkFvTUFrUlhNUnN3R1FZRFZRUUREQkowYkhNdFkyOXVkR1Y0ZEMxbwpiM04wTFRFd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUUM5T2dDOHd4eUlyUHpvCkdYc0xwUEt0NzJERXgyd2p3VzhuWFcyd1dieWEzYzk2bjJuU0NLUEJuODVoYnFzaHpqNWloU1RBTURJb2c5RnYKRzZSS1dVUFhUNEtJa1R2M0NESHFYc0FwSmxKNGxTeW5ReW8yWnYwbytBZjhDTG5nWVpCK3JmenRad3llRGhWcAp3WXpCVjIzNXp6NisycWJWbUNabHZCdVhiVXFUbEVZWXZ1R2xNR3o3cFBmT1dLVXBlWW9kYkcyZmIraEZGcGVvCkN4a1VYclFzT29SNUpkSEc1aldyWnVCTzQ1NVNzcnpCTDhSbGU1VUhvMDVXY0s3YkJiaVF6MTA2cEhDSllaK3AKdmxQSWNOU1g1S2gzNEZnOTZVUHg5bFFpQTN6RFRLQmZ5V2NMUStxMWNabExjV2RnUkZjTkJpckdCLzdyYTFWVApnRUplR2tQekFnTUJBQUdqVXpCUk1CMEdBMVVkRGdRV0JCUkRWVUtYWWJsRFdNTzE3MUJuWWZhYlkzM0NFVEFmCkJnTlZIU01FR0RBV2dCUkRWVUtYWWJsRFdNTzE3MUJuWWZhYlkzM0NFVEFQQmdOVkhSTUJBZjhFQlRBREFRSC8KTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFBUE8vRDRUdDUyWHJsQ0NmUzZnVUVkRU5DcnBBV05YRHJvR2M2dApTVGx3aC8rUUxRYk5hZEtlaEtiZjg5clhLaituVXF0cS9OUlpQSXNBSytXVWtHOVpQb1FPOFBRaVY0V1g1clE3CjI5dUtjSmZhQlhrZHpVVzdxTlFoRTRjOEJhc0JySWVzcmtqcFQ5OVF4SktuWFFhTitTdzdvRlBVSUFOMzhHcWEKV2wvS1BNVHRicWt3eWFjS01CbXExVkx6dldKb0g1Q2l6Skp3aG5rWHh0V0tzLzY3clROblBWTXorbWVHdHZTaQpkcVg2V1NTbUdMRkVFcjJoZ1VjQVpqazNWdVFoLzc1aFh1K1UySXRzQys1cXBsaEc3Q1hzb1huS0t5MVhsT0FFCmI4a3IyZFdXRWs2STVZNm5USnpXSWxTVGtXODl4d1hyY3RtTjlzYjlxNFNuaVZsegotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2UUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktjd2dnU2pBZ0VBQW9JQkFRQzlPZ0M4d3h5SXJQem8KR1hzTHBQS3Q3MkRFeDJ3andXOG5YVzJ3V2J5YTNjOTZuMm5TQ0tQQm44NWhicXNoemo1aWhTVEFNRElvZzlGdgpHNlJLV1VQWFQ0S0lrVHYzQ0RIcVhzQXBKbEo0bFN5blF5bzJadjBvK0FmOENMbmdZWkIrcmZ6dFp3eWVEaFZwCndZekJWMjM1eno2KzJxYlZtQ1psdkJ1WGJVcVRsRVlZdnVHbE1HejdwUGZPV0tVcGVZb2RiRzJmYitoRkZwZW8KQ3hrVVhyUXNPb1I1SmRIRzVqV3JadUJPNDU1U3NyekJMOFJsZTVVSG8wNVdjSzdiQmJpUXoxMDZwSENKWVorcAp2bFBJY05TWDVLaDM0Rmc5NlVQeDlsUWlBM3pEVEtCZnlXY0xRK3ExY1psTGNXZGdSRmNOQmlyR0IvN3JhMVZUCmdFSmVHa1B6QWdNQkFBRUNnZ0VBQmFsN3BpcE1hMGFKMXNRVWEzZkhEeTlQZlBQZXAzODlQVGROZGU1cGQxVFYKeFh5SnBSQS9IaWNTL05WYjU0b05VZE5jRXlnZUNCcFJwUHAxd3dmQ3dPbVBKVmo3SzF3aWFqbmxsQldpZUJzMgpsOWFwcDdFVE9DdWJ5WTNWU2dLQldWa0piVzBjOG9uSFdEL0RYM0duUjhkTXdGYzRrTUdadkllUlo4bU1acmdHCjZPdDNKOHI2eVZsZWI2OGF1WmtneXMwR2VGc3pNdVRubHJCOEw5djI1UUtjVGtESjIvRWx1Y1p5aER0eGF0OEIKTzZOUnNubmNyOHhwUVdPci9sV3M5VVFuZEdCdHFzbXMrdGNUN1ZUNU9UanQ4WHY5NVhNSHB5Z29pTHk3czhvYwpJMGprNDJabzRKZW5JT3c2Rm0weUFEZ0E3eWlXcks0bEkzWGhqaTVSb1FLQmdRRGRqaWNkTUpYVUZWc28rNTJkCkUwT2EwcEpVMFNSaC9JQmdvRzdNakhrVWxiaXlpR1pNanA5MEo5VHFaL1ErM1pWZVdqMmxPSWF0OG5nUzB6MDAKVzA3T1ZxYXprMVNYaFZlY2tGNWFEcm5PRDNhU2VWMSthV3JUdDFXRWdqOVFxYnJZYVA5emd4UkpkRzV3WENCUApGNDNFeXE5ZEhXOWF6SSt3UHlJQ0JqNnZBd0tCZ1FEYXBTelhPR2ViMi9SMWhlWXdWV240czNGZEtYVjgzemtTCnFSWDd6d1pLdkk5OGMybDU1Y1ZNUzBoTGM0bTVPMXZCaUd5SG80eTB2SVAvR0k0Rzl4T1FhMXdpVnNmUVBiSU4KLzJPSDFnNXJLSFdCWVJUaHZGcERqdHJRU2xyRHVjWUNSRExCd1hUcDFrbVBkL09mY2FybG42MjZEamthZllieAp3dWUydlhCTVVRS0JnQm4vTmlPOHNiZ0RFWUZMbFFEN1k3RmxCL3FmMTg4UG05aTZ1b1dSN2hzMlBrZmtyV3hLClIvZVBQUEtNWkNLRVNhU2FuaVVtN3RhMlh0U0dxT1hkMk85cFI0Skd4V1JLSnkrZDJSUmtLZlU5NTBIa3I4M0gKZk50KzVhLzR3SWtzZ1ZvblorSWIvV05wSUJSYkd3ZHMwaHZIVkxCdVpjU1h3RHlFQysrRTRCSVZBb0dCQUoxUQp6eXlqWnRqYnI4NkhZeEpQd29teEF0WVhLSE9LWVJRdUdLVXZWY1djV2xrZTZUdE51V0dsb1FTNHd0VkdBa1VECmxhTWFaL2o2MHJaT3dwSDhZRlUvQ2ZHakl1MlFGbmEvMUtzOXR1NGZGRHpjenh1RVhDWFR1Vmk0eHdtZ3R2bVcKZkRhd3JTQTZrSDdydlp4eE9wY3hCdHloc3pCK05RUHFTckpQSjJlaEFvR0FkdFJKam9vU0lpYURVU25lZUcyZgpUTml1T01uazJkeFV3RVF2S1E4eWNuUnpyN0QwaEtZVWIycThHKzE2bThQUjNCcFMzZDFLbkpMVnI3TUhaWHpSCitzZHNaWGtTMWVEcEZhV0RFREFEWWI0ckRCb2RBdk8xYm03ZXdTMzhSbk1UaTlhdFZzNVNTODNpZG5HbFZiSmsKYkZKWG0rWWxJNHFkaXowTFdjWGJyREE9Ci0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
kind: Secret
metadata:
  name: test-tlscontext-secret-1
  labels:
    kat-ambassador-id: hostdouble
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURnRENDQW1pZ0F3SUJBZ0lKQUlIWTY3cFNoZ3NyTUEwR0NTcUdTSWIzRFFFQkN3VUFNRlV4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVFzd0NRWURWUVFLREFKRQpWekViTUJrR0ExVUVBd3dTZEd4ekxXTnZiblJsZUhRdGFHOXpkQzB5TUI0WERURTRNVEV3TVRFME1EUXhObG9YCkRUSTRNVEF5T1RFME1EUXhObG93VlRFTE1Ba0dBMVVFQmhNQ1ZWTXhDekFKQmdOVkJBZ01BazFCTVE4d0RRWUQKVlFRSERBWkNiM04wYjI0eEN6QUpCZ05WQkFvTUFrUlhNUnN3R1FZRFZRUUREQkowYkhNdFkyOXVkR1Y0ZEMxbwpiM04wTFRJd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUURjQThZdGgvUFdhT0dTCm9ObXZFSFoyNGpRN1BLTitENG93TEhXZWl1UmRtaEEwWU92VTN3cUczVnFZNFpwbFpBVjBQS2xELysyWlNGMTQKejh3MWVGNFFUelphWXh3eTkrd2ZITmtUREVwTWpQOEpNMk9FYnlrVVJ4VVJ2VzQrN0QzMEUyRXo1T1BseG1jMApNWU0vL0pINUVEUWhjaURybFlxZTFTUk1SQUxaZVZta2FBeXU2TkhKVEJ1ajBTSVB1ZExUY2grOTBxK3Jkd255CmZrVDF4M09UYW5iV2pub21FSmU3TXZ5NG12dnFxSUh1NDhTOUM4WmQxQkdWUGJ1OFYvVURyU1dROXpZQ1g0U0cKT2FzbDhDMFhtSDZrZW1oUERsRC9UdjB4dnlINXE1TVVjSGk0bUp0Titnem9iNTREd3pWR0VqZWY1TGVTMVY1RgowVEFQMGQrWEFnTUJBQUdqVXpCUk1CMEdBMVVkRGdRV0JCUWRGMEdRSGRxbHRoZG5RWXFWaXVtRXJsUk9mREFmCkJnTlZIU01FR0RBV2dCUWRGMEdRSGRxbHRoZG5RWXFWaXVtRXJsUk9mREFQQmdOVkhSTUJBZjhFQlRBREFRSC8KTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFBbUFLYkNsdUhFZS9JRmJ1QWJneDBNenV6aTkwd2xtQVBiOGdtTwpxdmJwMjl1T1ZzVlNtUUFkZFBuZEZhTVhWcDFaaG1UVjVDU1F0ZFgyQ1ZNVyswVzQ3Qy9DT0Jkb1NFUTl5akJmCmlGRGNseG04QU4yUG1hR1FhK3hvT1hnWkxYZXJDaE5LV0JTWlIrWktYTEpTTTlVYUVTbEhmNXVuQkxFcENqK2oKZEJpSXFGY2E3eElGUGtyKzBSRW9BVmMveFBubnNhS2pMMlV5Z0dqUWZGTnhjT042Y3VjYjZMS0pYT1pFSVRiNQpINjhKdWFSQ0tyZWZZK0l5aFFWVk5taWk3dE1wY1UyS2pXNXBrVktxVTNkS0l0RXEyVmtTZHpNVUtqTnhZd3FGCll6YnozNFQ1MENXbm9HbU5SQVdKc0xlVmlPWVUyNmR3YkFXZDlVYitWMDFRam43OAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2d0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktrd2dnU2xBZ0VBQW9JQkFRRGNBOFl0aC9QV2FPR1MKb05tdkVIWjI0alE3UEtOK0Q0b3dMSFdlaXVSZG1oQTBZT3ZVM3dxRzNWcVk0WnBsWkFWMFBLbEQvKzJaU0YxNAp6OHcxZUY0UVR6WmFZeHd5OSt3ZkhOa1RERXBNalA4Sk0yT0VieWtVUnhVUnZXNCs3RDMwRTJFejVPUGx4bWMwCk1ZTS8vSkg1RURRaGNpRHJsWXFlMVNSTVJBTFplVm1rYUF5dTZOSEpUQnVqMFNJUHVkTFRjaCs5MHErcmR3bnkKZmtUMXgzT1RhbmJXam5vbUVKZTdNdnk0bXZ2cXFJSHU0OFM5QzhaZDFCR1ZQYnU4Vi9VRHJTV1E5ellDWDRTRwpPYXNsOEMwWG1INmtlbWhQRGxEL1R2MHh2eUg1cTVNVWNIaTRtSnROK2d6b2I1NER3elZHRWplZjVMZVMxVjVGCjBUQVAwZCtYQWdNQkFBRUNnZ0VCQUk2U3I0anYwZForanJhN0gzVnZ3S1RYZnl0bjV6YVlrVjhZWUh3RjIyakEKbm9HaTBSQllIUFU2V2l3NS9oaDRFWVM2anFHdkptUXZYY3NkTldMdEJsK2hSVUtiZVRtYUtWd2NFSnRrV24xeQozUTQwUytnVk5OU2NINDRvYUZuRU0zMklWWFFRZnBKMjJJZ2RFY1dVUVcvWnpUNWpPK3dPTXc4c1plSTZMSEtLCkdoOENsVDkrRGUvdXFqbjNCRnQwelZ3cnFLbllKSU1DSWFrb2lDRmtIcGhVTURFNVkyU1NLaGFGWndxMWtLd0sKdHFvWFpKQnlzYXhnUTFRa21mS1RnRkx5WlpXT01mRzVzb1VrU1RTeURFRzFsYnVYcHpUbTlVSTlKU2lsK01yaAp1LzVTeXBLOHBCSHhBdFg5VXdiTjFiRGw3Sng1SWJyMnNoM0F1UDF4OUpFQ2dZRUE4dGNTM09URXNOUFpQZlptCk9jaUduOW9STTdHVmVGdjMrL05iL3JodHp1L1RQUWJBSzhWZ3FrS0dPazNGN1krY2txS1NTWjFnUkF2SHBsZEIKaTY0Y0daT1dpK01jMWZVcEdVV2sxdnZXbG1nTUlQVjVtbFpvOHowMlNTdXhLZTI1Y2VNb09oenFlay9vRmFtdgoyTmxFeTh0dEhOMUxMS3grZllhMkpGcWVycThDZ1lFQTUvQUxHSXVrU3J0K0dkektJLzV5cjdSREpTVzIzUTJ4CkM5ZklUTUFSL1Q4dzNsWGhyUnRXcmlHL3l0QkVPNXdTMVIwdDkydW1nVkhIRTA5eFFXbzZ0Tm16QVBNb1RSekMKd08yYnJqQktBdUJkQ0RISjZsMlFnOEhPQWovUncrK2x4bEN0VEI2YS8xWEZIZnNHUGhqMEQrWlJiWVZzaE00UgpnSVVmdmpmQ1Y1a0NnWUVBMzdzL2FieHJhdThEaTQ3a0NBQ3o1N3FsZHBiNk92V2d0OFF5MGE5aG0vSmhFQ3lVCkNML0VtNWpHeWhpMWJuV05yNXVRWTdwVzR0cG5pdDJCU2d1VFlBMFYrck8zOFhmNThZcTBvRTFPR3l5cFlBUkoKa09SanRSYUVXVTJqNEJsaGJZZjNtL0xnSk9oUnp3T1RPNXFSUTZHY1dhZVlod1ExVmJrelByTXUxNGtDZ1lCbwp4dEhjWnNqelVidm5wd3hTTWxKUStaZ1RvZlAzN0lWOG1pQk1POEJrclRWQVczKzFtZElRbkFKdWRxTThZb2RICmF3VW03cVNyYXV3SjF5dU1wNWFadUhiYkNQMjl5QzVheFh3OHRtZlk0TTVtTTBmSjdqYW9ydGFId1pqYmNObHMKdTJsdUo2MVJoOGVpZ1pJU1gyZHgvMVB0ckFhWUFCZDcvYWVYWU0wVWtRS0JnUUNVbkFIdmRQUGhIVnJDWU1rTgpOOFBEK0t0YmhPRks2S3MvdlgyUkcyRnFmQkJPQWV3bEo1d0xWeFBLT1RpdytKS2FSeHhYMkcvREZVNzduOEQvCkR5V2RjM2ZCQWQ0a1lJamZVaGRGa1hHNEFMUDZBNVFIZVN4NzNScTFLNWxMVWhPbEZqc3VPZ0NKS28wVlFmRC8KT05paDB6SzN5Wmc3aDVQamZ1TUdGb09OQWc9PQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==
kind: Secret
metadata:
  name: test-tlscontext-secret-2
  labels:
    kat-ambassador-id: hostdouble
type: kubernetes.io/tls
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: host-1
  labels:
    kat-ambassador-id: hostdouble
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  hostname: tls-context-host-1
  selector:
    matchLabels:
      hostname: tls-context-host-1
  tlsSecret:
    name: test-tlscontext-secret-1
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: host-2
  labels:
    kat-ambassador-id: hostdouble
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-2
  acmeProvider:
    authority: none
  hostname: tls-context-host-2
  selector:
    matchLabels:
      hostname: tls-context-host-2
  tlsSecret:
    name: test-tlscontext-secret-2
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: host-1-mapping
  labels:
    hostname: tls-context-host-1
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "tls-context-host-1"
  prefix: /target/
  service: {self.target1.path.fqdn}
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: host-2-mapping
  labels:
    hostname: tls-context-host-2
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "tls-context-host-2"
  prefix: /target/
  service: {self.target2.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # 0
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"),
                    headers={"Host": "tls-context-host-1"},
                    insecure=True,
                    sni=True)

        # 1 - Correct host #1
        yield Query(self.url("target/"),
                    headers={"Host": "tls-context-host-1"},
                    expected=200,
                    insecure=True,
                    sni=True)
        # 2 - Correct host #2
        yield Query(self.url("target/"),
                    headers={"Host": "tls-context-host-2"},
                    expected=200,
                    insecure=True,
                    sni=True)

        # Setting the Host header really shouldn't be necessary here.
        yield Query(self.url("target/", scheme="http"),
                    headers={ "Host": "tls-context-host-1" },
                    expected=301)
        yield Query(self.url("target/", scheme="http"),
                    headers={ "Host": "tls-context-host-2" },
                    expected=301)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers:
                host_header = result.query.headers['Host']
                tls_common_name = result.tls[0]['Issuer']['CommonName']

                assert host_header == tls_common_name, "test %d wanted CN %s, but got %s" % (idx, host_header, tls_common_name)

            idx += 1

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without a Host header they can't work.
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))

