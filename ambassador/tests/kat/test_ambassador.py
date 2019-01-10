import json
import pytest
import os
import base64

from typing import ClassVar, Dict, Sequence, Tuple, Union

from kat.harness import variants, Query, Runner, Test
from kat.manifests import AMBASSADOR

from abstract_tests import DEV, AmbassadorTest, HTTP
from abstract_tests import MappingTest, OptionTest, ServiceType, Node

from t_ratelimit import RateLimitTest
from t_tracing import TracingTest
from t_shadow import ShadowTest
from t_extauth import AuthenticationTest, AuthenticationTestV1, AuthenticationHTTPBufferedTest

# XXX: should test empty ambassador config

GRAPHITE_CONFIG = """
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: {0}
spec:
  replicas: 1
  template:
    metadata:
      labels:
        service: {0}
    spec:
      containers:
      - name: {0}
        image: hopsoft/graphite-statsd:latest
      restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: {0}
  name: {0}
spec:
  ports:
  - protocol: UDP
    port: 8125
    name: statsd-metrics
  - protocol: TCP
    port: 80
    name: graphite-www
  selector:
    service: {0}
"""


class TLSContextsTest(AmbassadorTest):
    """
    This test makes sure that TLS is not turned on when it's not intended to. For example, when an 'upstream'
    TLS configuration is passed, the port is not supposed to switch to 443
    """

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: Module
name: tls
ambassador_id: {self.ambassador_id}
config:
  upstream:
    enabled: True
    secret: test-certs-secret
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: {self.target.path.k8s}
""")

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url(self.name + "/"), error=['connection reset by peer', 'EOF'])

    def requirements(self):
        yield from (r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("http://"))


class ClientCertificateAuthentication(AmbassadorTest):
    tls_crt_base64 = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUdPVENDQkNHZ0F3SUJBZ0lKQU9FL3ZKZDhFQjI0TUEwR0NTcUdTSWIzRFFFQkJRVUFNSUd5TVFzd0NRWUQKVlFRR0V3SkdVakVQTUEwR0ExVUVDQXdHUVd4ellXTmxNUk13RVFZRFZRUUhEQXBUZEhKaGMySnZkWEpuTVJndwpGZ1lEVlFRS0RBOTNkM2N1Wm5KbFpXeGhiaTV2Y21jeEVEQU9CZ05WQkFzTUIyWnlaV1ZzWVc0eExUQXJCZ05WCkJBTU1KRVp5WldWc1lXNGdVMkZ0Y0d4bElFTmxjblJwWm1sallYUmxJRUYxZEdodmNtbDBlVEVpTUNBR0NTcUcKU0liM0RRRUpBUllUWTI5dWRHRmpkRUJtY21WbGJHRnVMbTl5WnpBZUZ3MHhNakEwTWpjeE1ERTNORFJhRncweApNakExTWpjeE1ERTNORFJhTUlHeU1Rc3dDUVlEVlFRR0V3SkdVakVQTUEwR0ExVUVDQXdHUVd4ellXTmxNUk13CkVRWURWUVFIREFwVGRISmhjMkp2ZFhKbk1SZ3dGZ1lEVlFRS0RBOTNkM2N1Wm5KbFpXeGhiaTV2Y21jeEVEQU8KQmdOVkJBc01CMlp5WldWc1lXNHhMVEFyQmdOVkJBTU1KRVp5WldWc1lXNGdVMkZ0Y0d4bElFTmxjblJwWm1sagpZWFJsSUVGMWRHaHZjbWwwZVRFaU1DQUdDU3FHU0liM0RRRUpBUllUWTI5dWRHRmpkRUJtY21WbGJHRnVMbTl5Clp6Q0NBaUl3RFFZSktvWklodmNOQVFFQkJRQURnZ0lQQURDQ0Fnb0NnZ0lCQU9EcCs4b1FjSytNVHVXUFpWeEoKWlI3NXBhSzR6Y1VuZ3VwWVhXU0dXRlhQVFY3dnNzRms2dkluZVBBclRMK1Q5S3dIZmlaMjlQcDNVYnpEbHlzWQpLejlmOUFlNTBqR0Q2eFZQd1hnUS9WSTk3OUd5Rlh6aGlFTXRTWXlrRjA0dEJKaURsMi9GWnhiSFBwTnhDMzl0CjE0a3d1RHFCaW45Ti9aYlQ1KzQ1dGJiUzh6aVhTK1FnTDVoRDJxMmVZQ1dheXJHRXQxWStqREFkSERIbUduWjgKZDRoYmdJTEpBczNJSW5PQ0RqQzRjMWd3SEZiOEc0UUhIVHdWaGpocXBrcTJoUUhneldCQzFsMkRrdS9vRFlldgpadS9wZnBUbzN6NitOT1lCclVXc2VRbUl1RytER01RQTlLT3VTUXZleVR5d0JtNEc0dlpLbjBzQ3UxL3YyKzlUCkJHdjQxdGdTL1lmNm9lZVFWcmJTNFJGWTFyOXFUSzZEVzl3a1RUZXNhNHhvREtRcldqU0o3K2FhOHR2QlhMR1gKeDJ4ZFJOV0xlUk11R0JTT2lod1htRHIrckNKUmF1VDdwSXRONVgrdVdOVFgxb2ZOa3NRU1VNYUZKNUs3TDBMVQppUXFVMll5dC84VXBoZFZaTDRFRmtHU0ExM1VEV3RiOW1NMWhZMGg2NUxsU1l3Q2NoRXBocnRJOWN1VitJVHJTCk5jTjZjUC9kcUR4MS9qV2Q2ZHFqTnU3K2R1Z3dYNWVsUVM5dVVZQ0ZtdWdSNXMxbTJlZUJnM1F1QzdnWkxFME4KTmJnUzdvU3hLSmU5S2VPY3c2OGpIV2ZCS3NDZkJmUTRmVTJ0L250TXliVDNoQ2RFTVF1NGRnTTVUeXcvVWVGcQowU2FKeVRsK0cxYlR6UzBGVzZ1VXA2TkxBZ01CQUFHalVEQk9NQjBHQTFVZERnUVdCQlFqYkMwOVBpbGRlTGhzClBxcml1eTRlYklmeVV6QWZCZ05WSFNNRUdEQVdnQlFqYkMwOVBpbGRlTGhzUHFyaXV5NGViSWZ5VXpBTUJnTlYKSFJNRUJUQURBUUgvTUEwR0NTcUdTSWIzRFFFQkJRVUFBNElDQVFDd1JKcEpDZ3A3UytrOUJUNlgza0JlZm9uRQpFT1l0eVdYQlBwdXlHM1FsbTFyZGhjNjZEQ0dGb3JEbVR4ak1tSFl0Tm1BVm5NMzdJTFc3TW9mbFdyQWthWTE5Cmd2ODhGendhNWU2cldLNGZUU3BpRU9jNVdCMkEzSFBOOXdKbmhRWHQxV1dNREQ3akpTTHhMSXdGcWt6cERiREUKOTEyMlR0bklibUtOdjBVUXB6UFYzWWdicW9qeTZlWkhVT1QwNU5hT1Q3dnZpdjVRd01BSDVXZVJmaUN5czhDRwpTbm8vbzgzME9uaUVIdmVQVFlzd0xsWDIyTHlmU0hlb1RRQ0NJOHBvY3l0bDdJd0FSS0N2QmdlRnF2UHJNaXFQCmNoMTZGaVU5SUk4S2FNZ3BlYnJVU3ozSjFCQXBPT2QxTEJkNDJCZVRBa05TeGpSdmJoOC9sRFdmbkU3T0RiS2MKYjZBZDNWOWZsRmI1T0JaSDRhVGk2UWZyRG5CbWJMZ0xMOG8vTUxNK2QzS2c5NFhSVTlMakMycmppdlE2TUM1MwpFbldOb2JjSkZZK3NvWHNKb2tHdEZ4S2dJeDhYcmhGNUdPc1QyZjFwbU1sWUw0Y2psVTB1V2tQT09raHE4dElwClI4Y0JZcGh6WHUxdjZoMkFhWkxScTE4NGUzMFpPOThvbUt5UW9RMktBbTVBWmF5UnJaWnRqdkVaUE5hbVN1VlEKaVBlM28vNHR5UUdxK2pFTUFFakxsREVDdTBkRWE2UkZudGNiQlBNQlAzd1p3RTJiSTlHWWd2eWFaZDYzRE5kbQpYZDY1bTBtbWZPV1l0dGZyRFQzUTk1WVA1NG5IcEl4S0J3MWVGT3pyblhPcWJLVm1KLzFGRFAyeVdlb29LVkxmCkt2YnhVY0RhVnZYQjBFVTBiZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return super().manifests() + """
---
apiVersion: v1
metadata:
  name: client-cert-secret
data:
  tls.crt: {}
kind: Secret
type: Opaque
---
apiVersion: v1
kind: Secret
metadata:
  name: client-cert-server-secret
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUdKekNDQkErZ0F3SUJBZ0lCQVRBTkJna3Foa2lHOXcwQkFRVUZBRENCc2pFTE1Ba0dBMVVFQmhNQ1JsSXgKRHpBTkJnTlZCQWdNQmtGc2MyRmpaVEVUTUJFR0ExVUVCd3dLVTNSeVlYTmliM1Z5WnpFWU1CWUdBMVVFQ2d3UApkM2QzTG1aeVpXVnNZVzR1YjNKbk1SQXdEZ1lEVlFRTERBZG1jbVZsYkdGdU1TMHdLd1lEVlFRRERDUkdjbVZsCmJHRnVJRk5oYlhCc1pTQkRaWEowYVdacFkyRjBaU0JCZFhSb2IzSnBkSGt4SWpBZ0Jna3Foa2lHOXcwQkNRRVcKRTJOdmJuUmhZM1JBWm5KbFpXeGhiaTV2Y21jd0hoY05NVEl3TkRJM01UQXpNVEU0V2hjTk1qSXdOREkxTVRBegpNVEU0V2pCK01Rc3dDUVlEVlFRR0V3SkdVakVQTUEwR0ExVUVDQXdHUVd4ellXTmxNUmd3RmdZRFZRUUtEQTkzCmQzY3VabkpsWld4aGJpNXZjbWN4RURBT0JnTlZCQXNNQjJaeVpXVnNZVzR4RGpBTUJnTlZCQU1NQldGc2FXTmwKTVNJd0lBWUpLb1pJaHZjTkFRa0JGaE5qYjI1MFlXTjBRR1p5WldWc1lXNHViM0puTUlJQ0lqQU5CZ2txaGtpRwo5dzBCQVFFRkFBT0NBZzhBTUlJQ0NnS0NBZ0VBM1cyOStJRDYxOTRiSDZlakxySUM0aGIyVWdvOHY2WkMrTXJjCmsyZE5ZTU5QamNPS0FCdnh4RXRCYW1uU2FlVS9JWTdGQy9naU42MjJMRXRWLzNvRGNydWEwK3lXdVZhZnl4bVoKeVRLVWI0L0dVZ2FmUlFQZi9laVg5dXJXdXJ0SUs3WGdOR0ZOVWpZUHE0ZFNKUVBQaHdDSEUvTEtBeWtXblpCWApSclgwRHE0WHlBcE5rdTBJcGpJakVYSCs4aXhFMTJ3SDh3dDdERXZkTzdUM04zQ2ZVYmFJVGwxcUJYK05tMlo2CnE0QWcvdTVybDhOSmZYZzcxWm1YQTNYT2o3ekZ2cHlhcFJJWmNQbWt2WlluN1NNQ3A4ZFh5WEhQZHBTaUlXTDIKdUIzS2lPNEpyVVl2dDJHekxCVVRocCtsTlNaYVovUTN5T2FBQVVrT3grMWgwODI4NVBpK1A4bE8rSDJYaWM0Uwp2TXExeHRMZzJiTm9QQzVLbmJSZnVGUHVVRDIvM2RTaWlyYWdKNnVZRExPeVdKRGl2S0d0LzcyT1ZURVBBTDlvCjZUMnBHWnJ3YlF1aUZHckdUTVpPdldNU3BRdE5sK3RDQ1hsVDRtV3FKRFJ3dU1Hckk0RG5uR3p0M0lLcU53UzQKUXlvOUtxak1JUHduWFpBbVdQbTNGT0tlNHNGd2M1ZnBhd0tPMDFKWmV3RHNZVER4VmorY3dYd0Z4YkUyeUJpRgp6MkZBSHdmb3B3YUgzNXAzQzZsa2NnUDJrL3pnQWxuQmx1ekFDVUkrTUtKL0cwZ3YvdUFoajFPSEpRM0w2a24xClNwdlE0MS91ZUJqbHVuRXhxUVNZRDdHdFoxS2c4dU9jcTJyK1dJU0UzUWM5TXBRRkZrVVZsbG1nV0d3WUR1TjMKWnNlejk1a0NBd0VBQWFON01Ia3dDUVlEVlIwVEJBSXdBREFzQmdsZ2hrZ0JodmhDQVEwRUh4WWRUM0JsYmxOVApUQ0JIWlc1bGNtRjBaV1FnUTJWeWRHbG1hV05oZEdVd0hRWURWUjBPQkJZRUZGbGZ5Uk82Rzh5NXFFRktpa2w1CmFqYjJmVDdYTUI4R0ExVWRJd1FZTUJhQUZDTnNMVDArS1YxNHVHdytxdUs3TGg1c2gvSlRNQTBHQ1NxR1NJYjMKRFFFQkJRVUFBNElDQVFBVDV3SkZQcWVydmJqYTUrOTBpS3hpMWQwUVZ0VkdCK3o2YW9BTXVXSytxZ2kwdmd2cgptdTlvdDJsdlRTQ1NuUmhqZWlQMFNJZHFGTU9SbUJ0T0NGay9rWURwOU0vOTFiK3ZTK1M5ZUFseHJOQ0I1Vk9mClBxeEVQcC93djFyQmNFNEdCTy9jNkhjRm9uM0Yrb0JZQ3NVUWJaREtTU1p4aERtM21qN3BiNjdGTmJaYkpJekoKNzBIRHNSZTJPMDRvaVR4K2g2ZzZwVzNjT1FNZ0lBdkZnS041RXg3MjdLNDIzMEIwTklkR2t6dWo0S1NNTDBOTQpzbFNBY1haNDFPb1NLTmp5NDRCVkVadjBaZHhURHJSTTRFd0p0TnlnZ0Z6bXRUdVYwMm5rVWoxYllZWUM1ZjBMCkFEcjZzMFhNeWFOazh0d2xXWWxZRFo1dUtEcFZSVkJmaUdjcTB1Skl6SXZlbWh1VHJvZmg4cEJRUU5rUFJERlQKUnExaVRvMUloaGwzL0ZsMWtYazFXUjNqVGpOYjRqSFg3bElvWHdwd3A3NjdIQVBLR2hqUTljRmJuSE1FdGtybwpSbEpZZHRScTVtY2NEdHdUMEdGeW9KTExCWmRISE1ISnowRjlIN0ZOazJ0VFFRTWhLNU1WWXdnK0xJYWVlNTg2CkNRVnFmYnNjcDdldmxnakxXOThIKzV6eWxSSEFnb0gyRzc5YUhsak5LTXA5Qk91cTZTbkVnbEVzaVdHVnR1MmwKaG54OFNCM3NWSlpIZWVyOGYvVVFRd3FiQU8rS2R5NzBObWJTYXFhVnRwOGpPeExpaWRXa3dTeVJUc3VVNkQ4aQpEaUg1dUVxQlhFeGpyajBGc2x4Y1ZLZFZqNWdsVmNTbWtMd1pLYkVVMU9Ld2xlVC9pWEZodm9vV2hRPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  tls.key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlKS1FJQkFBS0NBZ0VBM1cyOStJRDYxOTRiSDZlakxySUM0aGIyVWdvOHY2WkMrTXJjazJkTllNTlBqY09LCkFCdnh4RXRCYW1uU2FlVS9JWTdGQy9naU42MjJMRXRWLzNvRGNydWEwK3lXdVZhZnl4bVp5VEtVYjQvR1VnYWYKUlFQZi9laVg5dXJXdXJ0SUs3WGdOR0ZOVWpZUHE0ZFNKUVBQaHdDSEUvTEtBeWtXblpCWFJyWDBEcTRYeUFwTgprdTBJcGpJakVYSCs4aXhFMTJ3SDh3dDdERXZkTzdUM04zQ2ZVYmFJVGwxcUJYK05tMlo2cTRBZy91NXJsOE5KCmZYZzcxWm1YQTNYT2o3ekZ2cHlhcFJJWmNQbWt2WlluN1NNQ3A4ZFh5WEhQZHBTaUlXTDJ1QjNLaU80SnJVWXYKdDJHekxCVVRocCtsTlNaYVovUTN5T2FBQVVrT3grMWgwODI4NVBpK1A4bE8rSDJYaWM0U3ZNcTF4dExnMmJObwpQQzVLbmJSZnVGUHVVRDIvM2RTaWlyYWdKNnVZRExPeVdKRGl2S0d0LzcyT1ZURVBBTDlvNlQycEdacndiUXVpCkZHckdUTVpPdldNU3BRdE5sK3RDQ1hsVDRtV3FKRFJ3dU1Hckk0RG5uR3p0M0lLcU53UzRReW85S3FqTUlQd24KWFpBbVdQbTNGT0tlNHNGd2M1ZnBhd0tPMDFKWmV3RHNZVER4VmorY3dYd0Z4YkUyeUJpRnoyRkFId2ZvcHdhSAozNXAzQzZsa2NnUDJrL3pnQWxuQmx1ekFDVUkrTUtKL0cwZ3YvdUFoajFPSEpRM0w2a24xU3B2UTQxL3VlQmpsCnVuRXhxUVNZRDdHdFoxS2c4dU9jcTJyK1dJU0UzUWM5TXBRRkZrVVZsbG1nV0d3WUR1TjNac2V6OTVrQ0F3RUEKQVFLQ0FnQnltRUh4b3VhdTR6Nk1VbGlzYU9uL0VqMG1WaS84UzFKcnFha2dEQjFLajZuVFJ6aGJPQnNXS0pCUgpQelRySXY1YUlxWXR2SndRenJEeUdZY0hNYUVwTnBnNVJ6NzE2alBHaTVoQVBSSCs3cHlIaE8vV2F0djRidkIrCmxDak8rTyt2MTIrU0RDMVU5NitDYVFVRkxRU3c3SC83dmZINFVzSm1odlgwSFdTU1dGenNaUkNpa2xPZ2wxLzQKdmxOZ0I3TVUvYzdiWkx5b3IzWnVXUWg4UTZmZ1JTUWowa3AxVC83OFJyd0RsOHI3eEc0Z1c2dmo2RjZtKzliZwpybzVaYXl1M3F4cUpoV1Z2UjNPUHZtOHBWYTRoSUpSNUo1SmozeVpOT3dkT1gvU2FpdjZ0RXg3TXZCNWJHUWxDCjZjbzVTSUVQUFovRk5DMVkvUE5PV3JiL1E0R1cxQVNjZElDWnU3d0lrS3pXQUpDbzU5QThMdXY1RlY4dm00UjIKNEpreUI2a1hjVmZvd3JqWVhxREYvVVgwZGRETExHRjk2WlN0dGUzUFhYOFBRV1k4OUZadUJrR3c2TlJaSW5IaQp4aW5OMlY4Y203Q3c4NWQ5RXoyekVHQjRLQzdMSStKZ0xRdGRnM1h2YmRmaE9pMDZlR2pnSzJtd2ZPcVQ4U3ErCnY5UE9JSlhUTkVJM2ZpM2RCODZhZi84T1hSdE9yQWExbWlrMm1zREkxR29pN2NLUWJDM2Z6L3AxSVNRQ3B0dnMKWXZOd3N0RER1dGtBOW85YXJhUXk1YjBMQzZ3NWsrQ1NkVk5iZDhPMkVVZDBPQk9VamJsSEt2ZFozVm96OEVERgp5d1lpbW1OR2plMWxLOG5oMm5kcGphNXEzaXBEczFoS2c1VXVqb0dmZWkyZ24wY2g1UUtDQVFFQThPK0lIT091ClQvbFVnV3Nwb3BoRTBZMWFVSlFQcWdLM0VpS0I4NGFwd0xmejJlQVBTQmZmMmRDTjdYcDZzLy91MGZvNDFMRTUKUDBkcy81ZXU5UERsTkY2SEg1SDNPWXBWLzU3djVPMk9TQlFkQi8rM1RtTm1RR1lKQ1N6b3VJUzNZTk9VUFExegpGRnZSYXRlTjkxQlc3d0tGSHIwK000ekc2ZXpmdXRBUXl3V05vY2U3b0dhWVRUOHoveVdYcW1GaWREcW5nNXc1CjZkOHQ0MFNjb3pJVmFjR3VnK2xSaThsYlRDKzNUcDByK2xhNjZoNDl1cGdlZDNoRk92R1hJT3lidlljRTk4SzIKR3BObDljYzRxNk8xV0xkUjdRQzkxWk5mbEtPS0U4ZkFMTFovc3RFWEwwcDJiaXhiU25iSWR4T0VVY2gvaVFoTQpjaHhsc1JGTGp4VjFkd0tDQVFFQTYwWDZMeWVmSWxYelUzUEErZ0lSWVYwZzhGT3h6eFhmdnF2WWV5T0d3RGFhCnAvRXg1MHo3NmpJSks4d2xXNUVpN1U2eHN4eHczRTlETEg3U2YzSDRLaUdvdUJWSWRjdjkrSVIwTGNkWVBSOVYKb0NRMU1tNWE3ZmpubS9GSndUb2tkZ1dHU3dtRlRINy9qR2NOSFo4bHVtbFJGQ2o2VmNMVC9uUnhNNmRnSVhTbwp3MUQ5UUdDOVYrZTZLT1o2VlI1eEswaDhwT3RrcW9HcmJGTHUyNkdQQlN1Z3VQSlh0MGZ3SnQ5UEFHKzZWdnhKCjg5TkxNTC9uK2cyL2pWS1hoZlRUMU1iYjNGeDRsbmJMbmtQK0pydllJYW9RMVBaTmdnSUxZQ1VHSkpUTHRxT1QKZ2tnMVM0MS9YOEVGZzY3MWtBQjZaWVBiZDVXbkwxNFhwMGE5TU9CL2J3S0NBUUVBNldWQWw2dS9hbDEvalRkQQpSKy8xaW9IQjRaanNhNmJoclVHY1hVb3dHeTZYbkpHK2Uvb1VzUzJrcjA0Y20wM3NEYUMxZU9TTkxrMkV1enczCkViUmlkSTYxbXRHTmlrSUYrUEFBTitZZ0ZKYlhZSzVJNWpqSURzNUpKb2hJa0thUDljNUFKYnhucEdzbHZMZy8KSURyRlhCYzIyWVk5UVRhNFlsZENpL2VPclAwZUxJQU5zOTV1M3pYQXF3UEJuaDFrZ0c5cFlzYnVHeTVGaDRrcApxN1dTcExZbzFrUW82SjhRUUFkaExWaDRCN1FJc1U3R1FZR20wZGpDUjgxTXQybzluQ1cxbkVVVW56MzJZVmF5CkFTTS9RMGVpcDFJMmt6U0dQTGtId3cyWGpqamtEMWNaZkloSG5ZWitrTzNzVjkyaUtvOXRiRk9McW1iejQ4bDcKUm9wbEZRS0NBUUVBNmkrRGNvQ0w1QStOM3Rsdmt1dVFCVXcveHpobjJ1dTVCUC9rd2QyQStiN2dmcDZVdjlsZgpQNlNDZ0hmNkQ0VU9NUXlOME8xVVlkYjcxRVNBbnA4QkdGN2NwQzk3S3RYY2ZRekszKzUzSkpBV0dRc3hjSHRzClEwZm9zczZnVFpma1J4NEVxSmhYZU9kSTA2YVg1WTVPYlpqN1BZZjBkbjB4cXl5WXFZUEhLa1lHM2pPMWdlbEoKVDBDM2lwS3YzaDRwSTU1Smc1ZFRZbTBrQnZVZUVMeGxzZzNWTTRMMlVOZG9jaWtCYUR2T1RWdGUrVGF1dDEydQpPTGFLbnM5QlIvT0ZEMXpKNkRTYlM1bi80QTlwNFlCRkNHMVJ4OGxMS1VlRHJ6WHJRV3Bpdys5YW11bnBNc1VyCnJsSmhmTXdnWGpBN3BPUjFCam1PYXBYTUVaTldLbHFzUFFLQ0FRQnlWRHhJd01RY3pVRndRTVhjdTJJYkEzWjgKQ3poZjY2K3ZRV2graExSelFPWTRoUEJOY2VVaWVrcEhSTHdkSGF4U2xEVHFCN1ZQcSsyZ1NrVnJDWDgvWFRGYgpTZVZIVFlFN2l5MENreW1lKzJ4Y21zbC9EaVVIZkV5K1hOY0RnT3V0UzVNbldYQU5xTVFFb2FMVytOUExJM0x1ClYxc0NNWVRkN0hOOXR3N3docUxnMTh3QjF6b21TTVZHVDREa2ttQXpxNHpTS0kxRk5ZcDhLQTNPRTFFbXdxKzAKd1JzUXVhd1FWTENVRVAzVG82a1lPd1R6SnE3amhpVUs2Rm5qTGplVHJOUVNWZG9xd29KcmxUQUhnWFZWM3E3cQp2M1RHZDN4WEQ5eVFJam11Z05neE5pd0FaemhKcy9aSnkrK2ZQU0oxWFF4YmQ5cVBnaGdHb2UvZmY2RzcKLS0tLS1FTkQgUlNBIFBSSVZBVEUgS0VZLS0tLS0K
""".format(self.tls_crt_base64)

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
ambassador_id: {self.ambassador_id}
name: tls
config:
  server:
    enabled: True
    secret: client-cert-server-secret
  client:
    enabled: True
    secret: client-cert-secret
    cert_required: True
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: {self.target.path.k8s}
""")

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url(self.name + "/"), insecure=True, client_cert = base64.b64decode(self.tls_crt_base64).decode('utf-8'))
        # client_cert = base64.b64decode(self.tls_crt_base64).decode('utf-8')

    def requirements(self):
        return []


class TLSOriginationSecret(AmbassadorTest):

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
ambassador_id: {self.ambassador_id}
name: tls
config:
  upstream:
    secret: test-certs-secret
  upstream-files:
    cert_chain_file: /ambassador/default/secrets/test-certs-secret/tls.crt
    private_key_file: /ambassador/default/secrets/test-certs-secret/tls.key
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: {self.target.path.k8s}
tls: upstream
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}-files
prefix: /{self.name}-files/
service: {self.target.path.k8s}
tls: upstream-files
""")

    def queries(self):
        yield Query(self.url(self.name + "/"))
        yield Query(self.url(self.name + "-files/"))

    def check(self):
        for r in self.results:
            assert r.backend.request.tls.enabled


class TLS(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return super().manifests() + """
---
apiVersion: v1
kind: Secret
metadata:
  name: test-certs-secret
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
---
apiVersion: v1
kind: Secret
metadata:
  name: ambassador-certs
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURwakNDQW82Z0F3SUJBZ0lKQUpxa1Z4Y1RtQ1FITUEwR0NTcUdTSWIzRFFFQkN3VUFNR2d4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4RWpBUUJnTlZCQU1NQ1d4dlkyRnNhRzl6CmREQWVGdzB4T0RFd01UQXhNREk1TURKYUZ3MHlPREV3TURjeE1ESTVNREphTUdneEN6QUpCZ05WQkFZVEFsVlQKTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRS0RBaEVZWFJoZDJseQpaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEVqQVFCZ05WQkFNTUNXeHZZMkZzYUc5emREQ0NBU0l3CkRRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMcTZtdS9FSzlQc1Q0YkR1WWg0aEZPVnZiblAKekV6MGpQcnVzdXcxT05MQk9jT2htbmNSTnE4c1FyTGxBZ3NicDBuTFZmQ1pSZHQ4UnlOcUFGeUJlR29XS3IvZAprQVEybVBucjBQRHlCTzk0UHo4VHdydDBtZEtEU1dGanNxMjlOYVJaT0JqdStLcGV6RytOZ3pLMk04M0ZtSldUCnFYdTI3ME9pOXlqb2VGQ3lPMjdwUkdvcktkQk9TcmIwd3ozdFdWUGk4NFZMdnFKRWprT0JVZjJYNVF3b25XWngKMktxVUJ6OUFSZVVUMzdwUVJZQkJMSUdvSnM4U042cjF4MSt1dTNLdTVxSkN1QmRlSHlJbHpKb2V0aEp2K3pTMgowN0pFc2ZKWkluMWNpdXhNNzNPbmVRTm1LUkpsL2NEb3BLemswSldRSnRSV1NnbktneFNYWkRrZjJMOENBd0VBCkFhTlRNRkV3SFFZRFZSME9CQllFRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1COEdBMVVkSXdRWU1CYUEKRkJoQzdDeVRpNGFkSFVCd0wvTkZlRTZLdnFIRE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBSFJvb0xjcFdEa1IyMEhENEJ5d1BTUGRLV1hjWnN1U2tXYWZyekhoYUJ5MWJZcktIR1o1CmFodFF3L1gwQmRnMWtidlpZUDJSTzdGTFhBSlNTdXVJT0NHTFVwS0pkVHE1NDREUThNb1daWVZKbTc3UWxxam0KbHNIa2VlTlRNamFOVjdMd0MzalBkMERYelczbGVnWFRoYWpmZ2dtLzBJZXNGRzBVWjFEOTJHNURmc0hLekpSagpNSHZyVDNtVmJGZjkrSGJhRE4yT2g5VjIxUWhWSzF2M0F2dWNXczhUWCswZHZFZ1dtWHBRcndEd2pTMU04QkRYCldoWjVsZTZjVzhNYjhnZmRseG1JckpnQStuVVZzMU9EbkJKS1F3MUY4MVdkc25tWXdweVUrT2xVais4UGt1TVoKSU4rUlhQVnZMSWJ3czBmamJ4UXRzbTArZVBpRnN2d0NsUFk9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2Z0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktnd2dnU2tBZ0VBQW9JQkFRQzZ1cHJ2eEN2VDdFK0cKdzdtSWVJUlRsYjI1ejh4TTlJejY3ckxzTlRqU3dUbkRvWnAzRVRhdkxFS3k1UUlMRzZkSnkxWHdtVVhiZkVjagphZ0JjZ1hocUZpcS8zWkFFTnBqNTY5RHc4Z1R2ZUQ4L0U4SzdkSm5TZzBsaFk3S3R2VFdrV1RnWTd2aXFYc3h2CmpZTXl0alBOeFppVms2bDd0dTlEb3ZjbzZIaFFzanR1NlVScUt5blFUa3EyOU1NOTdWbFQ0dk9GUzc2aVJJNUQKZ1ZIOWwrVU1LSjFtY2RpcWxBYy9RRVhsRTkrNlVFV0FRU3lCcUNiUEVqZXE5Y2RmcnJ0eXJ1YWlRcmdYWGg4aQpKY3lhSHJZU2IvczB0dE95UkxIeVdTSjlYSXJzVE85enAza0RaaWtTWmYzQTZLU3M1TkNWa0NiVVZrb0p5b01VCmwyUTVIOWkvQWdNQkFBRUNnZ0VBSVFsZzNpamNCRHViK21Eb2syK1hJZDZ0V1pHZE9NUlBxUm5RU0NCR2RHdEIKV0E1Z2NNNTMyVmhBV0x4UnR6dG1ScFVXR0dKVnpMWlpNN2ZPWm85MWlYZHdpcytkYWxGcWtWVWFlM2FtVHVQOApkS0YvWTRFR3Nnc09VWSs5RGlZYXRvQWVmN0xRQmZ5TnVQTFZrb1JQK0FrTXJQSWFHMHhMV3JFYmYzNVp3eFRuCnd5TTF3YVpQb1oxWjZFdmhHQkxNNzlXYmY2VFY0WXVzSTRNOEVQdU1GcWlYcDNlRmZ4L0tnNHhtYnZtN1JhYzcKOEJ3Z3pnVmljNXlSbkVXYjhpWUh5WGtyazNTL0VCYUNEMlQwUjM5VmlVM1I0VjBmMUtyV3NjRHowVmNiVWNhKwpzeVdyaVhKMHBnR1N0Q3FWK0dRYy9aNmJjOGt4VWpTTWxOUWtudVJRZ1FLQmdRRHpwM1ZaVmFzMTA3NThVT00rCnZUeTFNL0V6azg4cWhGb21kYVFiSFRlbStpeGpCNlg3RU9sRlkya3JwUkwvbURDSEpwR0MzYlJtUHNFaHVGSUwKRHhSQ2hUcEtTVmNsSytaaUNPaWE1ektTVUpxZnBOcW15RnNaQlhJNnRkNW9mWk42aFpJVTlJR2RUaGlYMjBONwppUW01UnZlSUx2UHVwMWZRMmRqd2F6Ykgvd0tCZ1FERU1MN21Mb2RqSjBNTXh6ZnM3MW1FNmZOUFhBMVY2ZEgrCllCVG4xS2txaHJpampRWmFNbXZ6dEZmL1F3Wkhmd3FKQUVuNGx2em5ncUNzZTMvUElZMy8zRERxd1p2NE1vdy8KRGdBeTBLQmpQYVJGNjhYT1B1d0VuSFN1UjhyZFg2UzI3TXQ2cEZIeFZ2YjlRRFJuSXc4a3grSFVreml4U0h5Ugo2NWxESklEdlFRS0JnUURpQTF3ZldoQlBCZk9VYlpQZUJydmhlaVVycXRob29BemYwQkJCOW9CQks1OHczVTloCjdQWDFuNWxYR3ZEY2x0ZXRCbUhEK3RQMFpCSFNyWit0RW5mQW5NVE5VK3E2V0ZhRWFhOGF3WXR2bmNWUWdTTXgKd25oK1pVYm9udnVJQWJSajJyTC9MUzl1TTVzc2dmKy9BQWM5RGs5ZXkrOEtXY0Jqd3pBeEU4TGxFUUtCZ0IzNwoxVEVZcTFoY0I4Tk1MeC9tOUtkN21kUG5IYUtqdVpSRzJ1c1RkVWNxajgxdklDbG95MWJUbVI5Si93dXVQczN4ClhWekF0cVlyTUtNcnZMekxSQWgyZm9OaVU1UDdKYlA5VDhwMFdBN1N2T2h5d0NobE5XeisvRlltWXJxeWcxbngKbHFlSHRYNU03REtJUFhvRndhcTlZYVk3V2M2K1pVdG4xbVNNajZnQkFvR0JBSTgwdU9iTkdhRndQTVYrUWhiZApBelkrSFNGQjBkWWZxRytzcTBmRVdIWTNHTXFmNFh0aVRqUEFjWlg3RmdtT3Q5Uit3TlFQK0dFNjZoV0JpKzBWCmVLV3prV0lXeS9sTVZCSW0zVWtlSlRCT3NudTFVaGhXbm5WVDhFeWhEY1FxcndPSGlhaUo3bFZSZmRoRWFyQysKSnpaU0czOHVZUVlyc0lITnRVZFgySmdPCi0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
"""

    def config(self):
        # Use self here, not self.target, because we want the TLS module to
        # be annotated on the Ambassador itself.
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: Module
name: tls
ambassador_id: {self.ambassador_id}
config:
  server:
    enabled: True
    secret: test-certs-secret
""")

        # Use self.target _here_, because we want the httpbin mapping to
        # be annotated on the service, not the Ambassador. Also, you don't
        # need to include the ambassador_id unless you need some special
        # ambassador_id that isn't something that kat already knows about.
        #
        # If the test were more complex, we'd probably need to do some sort
        # of mangling for the mapping name and prefix. For this simple test,
        # it's not necessary.
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.k8s}
""")

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("tls-target/"), insecure=True)


class TLSInvalidSecret(TLS):

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: Module
name: tls
ambassador_id: {self.ambassador_id}
config:
  server:
    enabled: True
    secret: test-certs-secret-invalid
  missing-secret-key:
    cert_chain_file: /nonesuch
  bad-path-info:
    cert_chain_file: /nonesuch 
    private_key_file: /nonesuch
  validation-without-termination:  
    enabled: True
    secret: test-certs-secret-invalid
    ca_secret: ambassador-certs
""")

        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.k8s}
""")

    def scheme(self) -> str:
        return "http"

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)

    def check(self):
        errors = self.results[0].backend.response

        assert(len(errors) == 5)

        # I'm a little concerned about relying on specific text but hmm.
        found = 0

        wanted = {
            "TLSContext server found no certificate in secret test-certs-secret-invalid in namespace default",
            "TLSContext bad-path-info found no cert_chain_file '/nonesuch'",
            "TLSContext bad-path-info found no private_key_file '/nonesuch'",
            "TLSContext validation-without-termination found no certificate in secret test-certs-secret-invalid in namespace default",
            "TLSContext missing-secret-key: 'cert_chain_file' requires 'private_key_file' as well"
        }

        for errsvc, errtext in errors:
            if errtext in wanted:
                found += 1

        assert found == len(errors), "unexpected errors in list"

class RedirectTests(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def requirements(self):
        # only check https urls since test readiness will only end up barfing on redirect
        yield from (r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("https"))

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
    redirect_cleartext_from: 80
""")

        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.k8s}
""")

    def queries(self):
        yield Query(self.url("tls-target/"), expected=301)

class Plain(AmbassadorTest):

    @classmethod
    def variants(cls):
        yield cls(variants(MappingTest))

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield self, """
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config: {}
"""


def unique(options):
    added = set()
    result = []
    for o in options:
        if o.__class__ not in added:
            added.add(o.__class__)
            result.append(o)
    return tuple(result)


class SimpleMapping(MappingTest):

    parent: AmbassadorTest
    target: ServiceType

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

            for mot in variants(OptionTest):
                yield cls(st, (mot,), name="{self.target.name}-{self.options[0].name}")

            yield cls(st, unique(v for v in variants(OptionTest)
                                 if not getattr(v, "isolated", False)), name="{self.target.name}-all")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.k8s}
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)


class AddRequestHeaders(OptionTest):

    parent: Test

    VALUES: ClassVar[Sequence[Dict[str, str]]] = (
        { "foo": "bar" },
        { "moo": "arf" }
    )

    def config(self):
        yield "add_request_headers: %s" % json.dumps(self.value)

    def check(self):
        for r in self.parent.results:
            for k, v in self.value.items():
                actual = r.backend.request.headers.get(k.lower())
                assert actual == [v], (actual, [v])

class AddResponseHeaders(OptionTest):

    parent: Test

    VALUES: ClassVar[Sequence[Dict[str, str]]] = (
        { "foo": "bar" },
        { "moo": "arf" }
    )

    def config(self):
        yield "add_response_headers: %s" % json.dumps(self.value)

    def check(self):
        for r in self.parent.results:
            # Why do we end up with capitalized headers anyway??
            lowercased_headers = { k.lower(): v for k, v in r.headers.items() }

            for k, v in self.value.items():
                actual = lowercased_headers.get(k.lower())
                assert actual == [v], "expected %s: %s but got %s" % (k, v, lowercased_headers)


class HostHeaderMapping(MappingTest):

    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.k8s}
host: inspector.external
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), expected=404)
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.internal"}, expected=404)
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.external"})


class TLSContext(AmbassadorTest):
    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return super().manifests() + """
---
apiVersion: v1
kind: Namespace
metadata:
  name: secret-namespace
---
apiVersion: v1
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURnRENDQW1pZ0F3SUJBZ0lKQUpycUl0ekY2MTBpTUEwR0NTcUdTSWIzRFFFQkN3VUFNRlV4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVFzd0NRWURWUVFLREFKRQpWekViTUJrR0ExVUVBd3dTZEd4ekxXTnZiblJsZUhRdGFHOXpkQzB4TUI0WERURTRNVEV3TVRFek5UTXhPRm9YCkRUSTRNVEF5T1RFek5UTXhPRm93VlRFTE1Ba0dBMVVFQmhNQ1ZWTXhDekFKQmdOVkJBZ01BazFCTVE4d0RRWUQKVlFRSERBWkNiM04wYjI0eEN6QUpCZ05WQkFvTUFrUlhNUnN3R1FZRFZRUUREQkowYkhNdFkyOXVkR1Y0ZEMxbwpiM04wTFRFd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUUM5T2dDOHd4eUlyUHpvCkdYc0xwUEt0NzJERXgyd2p3VzhuWFcyd1dieWEzYzk2bjJuU0NLUEJuODVoYnFzaHpqNWloU1RBTURJb2c5RnYKRzZSS1dVUFhUNEtJa1R2M0NESHFYc0FwSmxKNGxTeW5ReW8yWnYwbytBZjhDTG5nWVpCK3JmenRad3llRGhWcAp3WXpCVjIzNXp6NisycWJWbUNabHZCdVhiVXFUbEVZWXZ1R2xNR3o3cFBmT1dLVXBlWW9kYkcyZmIraEZGcGVvCkN4a1VYclFzT29SNUpkSEc1aldyWnVCTzQ1NVNzcnpCTDhSbGU1VUhvMDVXY0s3YkJiaVF6MTA2cEhDSllaK3AKdmxQSWNOU1g1S2gzNEZnOTZVUHg5bFFpQTN6RFRLQmZ5V2NMUStxMWNabExjV2RnUkZjTkJpckdCLzdyYTFWVApnRUplR2tQekFnTUJBQUdqVXpCUk1CMEdBMVVkRGdRV0JCUkRWVUtYWWJsRFdNTzE3MUJuWWZhYlkzM0NFVEFmCkJnTlZIU01FR0RBV2dCUkRWVUtYWWJsRFdNTzE3MUJuWWZhYlkzM0NFVEFQQmdOVkhSTUJBZjhFQlRBREFRSC8KTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFBUE8vRDRUdDUyWHJsQ0NmUzZnVUVkRU5DcnBBV05YRHJvR2M2dApTVGx3aC8rUUxRYk5hZEtlaEtiZjg5clhLaituVXF0cS9OUlpQSXNBSytXVWtHOVpQb1FPOFBRaVY0V1g1clE3CjI5dUtjSmZhQlhrZHpVVzdxTlFoRTRjOEJhc0JySWVzcmtqcFQ5OVF4SktuWFFhTitTdzdvRlBVSUFOMzhHcWEKV2wvS1BNVHRicWt3eWFjS01CbXExVkx6dldKb0g1Q2l6Skp3aG5rWHh0V0tzLzY3clROblBWTXorbWVHdHZTaQpkcVg2V1NTbUdMRkVFcjJoZ1VjQVpqazNWdVFoLzc1aFh1K1UySXRzQys1cXBsaEc3Q1hzb1huS0t5MVhsT0FFCmI4a3IyZFdXRWs2STVZNm5USnpXSWxTVGtXODl4d1hyY3RtTjlzYjlxNFNuaVZsegotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2UUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktjd2dnU2pBZ0VBQW9JQkFRQzlPZ0M4d3h5SXJQem8KR1hzTHBQS3Q3MkRFeDJ3andXOG5YVzJ3V2J5YTNjOTZuMm5TQ0tQQm44NWhicXNoemo1aWhTVEFNRElvZzlGdgpHNlJLV1VQWFQ0S0lrVHYzQ0RIcVhzQXBKbEo0bFN5blF5bzJadjBvK0FmOENMbmdZWkIrcmZ6dFp3eWVEaFZwCndZekJWMjM1eno2KzJxYlZtQ1psdkJ1WGJVcVRsRVlZdnVHbE1HejdwUGZPV0tVcGVZb2RiRzJmYitoRkZwZW8KQ3hrVVhyUXNPb1I1SmRIRzVqV3JadUJPNDU1U3NyekJMOFJsZTVVSG8wNVdjSzdiQmJpUXoxMDZwSENKWVorcAp2bFBJY05TWDVLaDM0Rmc5NlVQeDlsUWlBM3pEVEtCZnlXY0xRK3ExY1psTGNXZGdSRmNOQmlyR0IvN3JhMVZUCmdFSmVHa1B6QWdNQkFBRUNnZ0VBQmFsN3BpcE1hMGFKMXNRVWEzZkhEeTlQZlBQZXAzODlQVGROZGU1cGQxVFYKeFh5SnBSQS9IaWNTL05WYjU0b05VZE5jRXlnZUNCcFJwUHAxd3dmQ3dPbVBKVmo3SzF3aWFqbmxsQldpZUJzMgpsOWFwcDdFVE9DdWJ5WTNWU2dLQldWa0piVzBjOG9uSFdEL0RYM0duUjhkTXdGYzRrTUdadkllUlo4bU1acmdHCjZPdDNKOHI2eVZsZWI2OGF1WmtneXMwR2VGc3pNdVRubHJCOEw5djI1UUtjVGtESjIvRWx1Y1p5aER0eGF0OEIKTzZOUnNubmNyOHhwUVdPci9sV3M5VVFuZEdCdHFzbXMrdGNUN1ZUNU9UanQ4WHY5NVhNSHB5Z29pTHk3czhvYwpJMGprNDJabzRKZW5JT3c2Rm0weUFEZ0E3eWlXcks0bEkzWGhqaTVSb1FLQmdRRGRqaWNkTUpYVUZWc28rNTJkCkUwT2EwcEpVMFNSaC9JQmdvRzdNakhrVWxiaXlpR1pNanA5MEo5VHFaL1ErM1pWZVdqMmxPSWF0OG5nUzB6MDAKVzA3T1ZxYXprMVNYaFZlY2tGNWFEcm5PRDNhU2VWMSthV3JUdDFXRWdqOVFxYnJZYVA5emd4UkpkRzV3WENCUApGNDNFeXE5ZEhXOWF6SSt3UHlJQ0JqNnZBd0tCZ1FEYXBTelhPR2ViMi9SMWhlWXdWV240czNGZEtYVjgzemtTCnFSWDd6d1pLdkk5OGMybDU1Y1ZNUzBoTGM0bTVPMXZCaUd5SG80eTB2SVAvR0k0Rzl4T1FhMXdpVnNmUVBiSU4KLzJPSDFnNXJLSFdCWVJUaHZGcERqdHJRU2xyRHVjWUNSRExCd1hUcDFrbVBkL09mY2FybG42MjZEamthZllieAp3dWUydlhCTVVRS0JnQm4vTmlPOHNiZ0RFWUZMbFFEN1k3RmxCL3FmMTg4UG05aTZ1b1dSN2hzMlBrZmtyV3hLClIvZVBQUEtNWkNLRVNhU2FuaVVtN3RhMlh0U0dxT1hkMk85cFI0Skd4V1JLSnkrZDJSUmtLZlU5NTBIa3I4M0gKZk50KzVhLzR3SWtzZ1ZvblorSWIvV05wSUJSYkd3ZHMwaHZIVkxCdVpjU1h3RHlFQysrRTRCSVZBb0dCQUoxUQp6eXlqWnRqYnI4NkhZeEpQd29teEF0WVhLSE9LWVJRdUdLVXZWY1djV2xrZTZUdE51V0dsb1FTNHd0VkdBa1VECmxhTWFaL2o2MHJaT3dwSDhZRlUvQ2ZHakl1MlFGbmEvMUtzOXR1NGZGRHpjenh1RVhDWFR1Vmk0eHdtZ3R2bVcKZkRhd3JTQTZrSDdydlp4eE9wY3hCdHloc3pCK05RUHFTckpQSjJlaEFvR0FkdFJKam9vU0lpYURVU25lZUcyZgpUTml1T01uazJkeFV3RVF2S1E4eWNuUnpyN0QwaEtZVWIycThHKzE2bThQUjNCcFMzZDFLbkpMVnI3TUhaWHpSCitzZHNaWGtTMWVEcEZhV0RFREFEWWI0ckRCb2RBdk8xYm03ZXdTMzhSbk1UaTlhdFZzNVNTODNpZG5HbFZiSmsKYkZKWG0rWWxJNHFkaXowTFdjWGJyREE9Ci0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
kind: Secret
metadata:
  name: same-secret-1
  namespace: secret-namespace
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURnRENDQW1pZ0F3SUJBZ0lKQUlIWTY3cFNoZ3NyTUEwR0NTcUdTSWIzRFFFQkN3VUFNRlV4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVFzd0NRWURWUVFLREFKRQpWekViTUJrR0ExVUVBd3dTZEd4ekxXTnZiblJsZUhRdGFHOXpkQzB5TUI0WERURTRNVEV3TVRFME1EUXhObG9YCkRUSTRNVEF5T1RFME1EUXhObG93VlRFTE1Ba0dBMVVFQmhNQ1ZWTXhDekFKQmdOVkJBZ01BazFCTVE4d0RRWUQKVlFRSERBWkNiM04wYjI0eEN6QUpCZ05WQkFvTUFrUlhNUnN3R1FZRFZRUUREQkowYkhNdFkyOXVkR1Y0ZEMxbwpiM04wTFRJd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUURjQThZdGgvUFdhT0dTCm9ObXZFSFoyNGpRN1BLTitENG93TEhXZWl1UmRtaEEwWU92VTN3cUczVnFZNFpwbFpBVjBQS2xELysyWlNGMTQKejh3MWVGNFFUelphWXh3eTkrd2ZITmtUREVwTWpQOEpNMk9FYnlrVVJ4VVJ2VzQrN0QzMEUyRXo1T1BseG1jMApNWU0vL0pINUVEUWhjaURybFlxZTFTUk1SQUxaZVZta2FBeXU2TkhKVEJ1ajBTSVB1ZExUY2grOTBxK3Jkd255CmZrVDF4M09UYW5iV2pub21FSmU3TXZ5NG12dnFxSUh1NDhTOUM4WmQxQkdWUGJ1OFYvVURyU1dROXpZQ1g0U0cKT2FzbDhDMFhtSDZrZW1oUERsRC9UdjB4dnlINXE1TVVjSGk0bUp0Titnem9iNTREd3pWR0VqZWY1TGVTMVY1RgowVEFQMGQrWEFnTUJBQUdqVXpCUk1CMEdBMVVkRGdRV0JCUWRGMEdRSGRxbHRoZG5RWXFWaXVtRXJsUk9mREFmCkJnTlZIU01FR0RBV2dCUWRGMEdRSGRxbHRoZG5RWXFWaXVtRXJsUk9mREFQQmdOVkhSTUJBZjhFQlRBREFRSC8KTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFBbUFLYkNsdUhFZS9JRmJ1QWJneDBNenV6aTkwd2xtQVBiOGdtTwpxdmJwMjl1T1ZzVlNtUUFkZFBuZEZhTVhWcDFaaG1UVjVDU1F0ZFgyQ1ZNVyswVzQ3Qy9DT0Jkb1NFUTl5akJmCmlGRGNseG04QU4yUG1hR1FhK3hvT1hnWkxYZXJDaE5LV0JTWlIrWktYTEpTTTlVYUVTbEhmNXVuQkxFcENqK2oKZEJpSXFGY2E3eElGUGtyKzBSRW9BVmMveFBubnNhS2pMMlV5Z0dqUWZGTnhjT042Y3VjYjZMS0pYT1pFSVRiNQpINjhKdWFSQ0tyZWZZK0l5aFFWVk5taWk3dE1wY1UyS2pXNXBrVktxVTNkS0l0RXEyVmtTZHpNVUtqTnhZd3FGCll6YnozNFQ1MENXbm9HbU5SQVdKc0xlVmlPWVUyNmR3YkFXZDlVYitWMDFRam43OAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2d0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktrd2dnU2xBZ0VBQW9JQkFRRGNBOFl0aC9QV2FPR1MKb05tdkVIWjI0alE3UEtOK0Q0b3dMSFdlaXVSZG1oQTBZT3ZVM3dxRzNWcVk0WnBsWkFWMFBLbEQvKzJaU0YxNAp6OHcxZUY0UVR6WmFZeHd5OSt3ZkhOa1RERXBNalA4Sk0yT0VieWtVUnhVUnZXNCs3RDMwRTJFejVPUGx4bWMwCk1ZTS8vSkg1RURRaGNpRHJsWXFlMVNSTVJBTFplVm1rYUF5dTZOSEpUQnVqMFNJUHVkTFRjaCs5MHErcmR3bnkKZmtUMXgzT1RhbmJXam5vbUVKZTdNdnk0bXZ2cXFJSHU0OFM5QzhaZDFCR1ZQYnU4Vi9VRHJTV1E5ellDWDRTRwpPYXNsOEMwWG1INmtlbWhQRGxEL1R2MHh2eUg1cTVNVWNIaTRtSnROK2d6b2I1NER3elZHRWplZjVMZVMxVjVGCjBUQVAwZCtYQWdNQkFBRUNnZ0VCQUk2U3I0anYwZForanJhN0gzVnZ3S1RYZnl0bjV6YVlrVjhZWUh3RjIyakEKbm9HaTBSQllIUFU2V2l3NS9oaDRFWVM2anFHdkptUXZYY3NkTldMdEJsK2hSVUtiZVRtYUtWd2NFSnRrV24xeQozUTQwUytnVk5OU2NINDRvYUZuRU0zMklWWFFRZnBKMjJJZ2RFY1dVUVcvWnpUNWpPK3dPTXc4c1plSTZMSEtLCkdoOENsVDkrRGUvdXFqbjNCRnQwelZ3cnFLbllKSU1DSWFrb2lDRmtIcGhVTURFNVkyU1NLaGFGWndxMWtLd0sKdHFvWFpKQnlzYXhnUTFRa21mS1RnRkx5WlpXT01mRzVzb1VrU1RTeURFRzFsYnVYcHpUbTlVSTlKU2lsK01yaAp1LzVTeXBLOHBCSHhBdFg5VXdiTjFiRGw3Sng1SWJyMnNoM0F1UDF4OUpFQ2dZRUE4dGNTM09URXNOUFpQZlptCk9jaUduOW9STTdHVmVGdjMrL05iL3JodHp1L1RQUWJBSzhWZ3FrS0dPazNGN1krY2txS1NTWjFnUkF2SHBsZEIKaTY0Y0daT1dpK01jMWZVcEdVV2sxdnZXbG1nTUlQVjVtbFpvOHowMlNTdXhLZTI1Y2VNb09oenFlay9vRmFtdgoyTmxFeTh0dEhOMUxMS3grZllhMkpGcWVycThDZ1lFQTUvQUxHSXVrU3J0K0dkektJLzV5cjdSREpTVzIzUTJ4CkM5ZklUTUFSL1Q4dzNsWGhyUnRXcmlHL3l0QkVPNXdTMVIwdDkydW1nVkhIRTA5eFFXbzZ0Tm16QVBNb1RSekMKd08yYnJqQktBdUJkQ0RISjZsMlFnOEhPQWovUncrK2x4bEN0VEI2YS8xWEZIZnNHUGhqMEQrWlJiWVZzaE00UgpnSVVmdmpmQ1Y1a0NnWUVBMzdzL2FieHJhdThEaTQ3a0NBQ3o1N3FsZHBiNk92V2d0OFF5MGE5aG0vSmhFQ3lVCkNML0VtNWpHeWhpMWJuV05yNXVRWTdwVzR0cG5pdDJCU2d1VFlBMFYrck8zOFhmNThZcTBvRTFPR3l5cFlBUkoKa09SanRSYUVXVTJqNEJsaGJZZjNtL0xnSk9oUnp3T1RPNXFSUTZHY1dhZVlod1ExVmJrelByTXUxNGtDZ1lCbwp4dEhjWnNqelVidm5wd3hTTWxKUStaZ1RvZlAzN0lWOG1pQk1POEJrclRWQVczKzFtZElRbkFKdWRxTThZb2RICmF3VW03cVNyYXV3SjF5dU1wNWFadUhiYkNQMjl5QzVheFh3OHRtZlk0TTVtTTBmSjdqYW9ydGFId1pqYmNObHMKdTJsdUo2MVJoOGVpZ1pJU1gyZHgvMVB0ckFhWUFCZDcvYWVYWU0wVWtRS0JnUUNVbkFIdmRQUGhIVnJDWU1rTgpOOFBEK0t0YmhPRks2S3MvdlgyUkcyRnFmQkJPQWV3bEo1d0xWeFBLT1RpdytKS2FSeHhYMkcvREZVNzduOEQvCkR5V2RjM2ZCQWQ0a1lJamZVaGRGa1hHNEFMUDZBNVFIZVN4NzNScTFLNWxMVWhPbEZqc3VPZ0NKS28wVlFmRC8KT05paDB6SzN5Wmc3aDVQamZ1TUdGb09OQWc9PQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==
kind: Secret
metadata:
  name: same-secret-2
type: kubernetes.io/tls
"""

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-same-prefix-1
prefix: /tls-context-same/
service: http://{self.target.path.k8s}
host: tls-context-host-1
""")
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: {self.name}-same-context-1
hosts:
- tls-context-host-1
secret: same-secret-1.secret-namespace
""")
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-same-prefix-2
prefix: /tls-context-same/
service: http://{self.target.path.k8s}
host: tls-context-host-2
""")
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: {self.name}-same-context-2
hosts:
- tls-context-host-2
secret: same-secret-2
alpn_protocols: h2,http/1.1
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-other-mapping
prefix: /{self.name}/
service: https://{self.target.path.k8s}
""")

    def scheme(self) -> str:
        return "https"

    @staticmethod
    def _go_close_connection_error(url):
        """
        :param url: url passed to the query
        :return: error message string that Go's net/http package throws when server closes connection
        """
        return "Get {}: EOF".format(url)

    def queries(self):
        # Correct host
        yield Query(self.url("tls-context-same/"),
                    headers={"Host": "tls-context-host-1"},
                    expected=200,
                    insecure=True,
                    sni=True)
        yield Query(self.url("tls-context-same/"),
                    headers={"Host": "tls-context-host-2"},
                    expected=200,
                    insecure=True,
                    sni=True)

        # Incorrect host
        yield Query(self.url("tls-context-same/"),
                    headers={"Host": "tls-context-host-3"},
                    error=self._go_close_connection_error(self.url("tls-context-same/")),
                    insecure=True,
                    sni=True)

        # Incorrect path, correct host
        yield Query(self.url("tls-context-different/"),
                    headers={"Host": "tls-context-host-1"},
                    expected=404,
                    insecure=True,
                    sni=True)

        # Other mappings with no host will fail
        yield Query(self.url(self.name + "/"),
                    error=self._go_close_connection_error(self.url(self.name + "/")),
                    insecure=True)

        # Other mappings with non-existent host will fail
        yield Query(self.url(self.name + "/"),
                    error=self._go_close_connection_error(self.url(self.name + "/")),
                    sni=True,
                    headers={"Host": "tls-context-host-3"},
                    insecure=True)

        # Other mappings should get all TLS if existing host specified
        yield Query(self.url(self.name + "/"),
                    headers={"Host": "tls-context-host-1"},
                    expected=200,
                    insecure=True,
                    sni=True)
        yield Query(self.url(self.name + "/"),
                    headers={"Host": "tls-context-host-2"},
                    expected=200,
                    insecure=True,
                    sni=True)

    def check(self):
        for result in self.results:
            if result.status == 200:
                host_header = result.query.headers['Host']
                tls_common_name = result.tls[0]['Issuer']['CommonName']
                assert host_header == tls_common_name


    def url(self, prefix, scheme=None) -> str:
        if scheme is None:
            scheme = self.scheme()
        if DEV:
            port = 8443
            return "%s://%s/%s" % (scheme, "localhost:%s" % (port + self.index), prefix)
        else:
            return "%s://%s/%s" % (scheme, self.path.k8s, prefix)

    def requirements(self):
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))


class UseWebsocket(OptionTest):
    # TODO: add a check with a websocket client as soon as we have backend support for it

    def config(self):
        yield 'use_websocket: true'


class WebSocketMapping(MappingTest):

    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: echo.websocket.org:80
host_rewrite: echo.websocket.org
use_websocket: true
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), expected=404)

        yield Query(self.parent.url(self.name + "/"), expected=101, headers={
            "Connection": "Upgrade",
            "Upgrade": "websocket",
            "sec-websocket-key": "DcndnpZl13bMQDh7HOcz0g==",
            "sec-websocket-version": "13"
        })

        yield Query(self.parent.url(self.name + "/", scheme="ws"), messages=["one", "two", "three"])

    def check(self):
        assert self.results[-1].messages == ["one", "two", "three"]


class CORS(OptionTest):
    # isolated = True
    # debug = True

    parent: MappingTest

    def config(self):
        yield 'cors: { origins: "*" }'

    def queries(self):
        for q in self.parent.queries():
            yield Query(q.url)  # redundant with parent
            yield Query(q.url, headers={ "Origin": "https://www.test-cors.org" })

    def check(self):
        # can assert about self.parent.results too
        assert self.results[0].backend.name == self.parent.target.path.k8s
        # Uh. Is it OK that this is case-sensitive?
        assert "Access-Control-Allow-Origin" not in self.results[0].headers

        assert self.results[1].backend.name == self.parent.target.path.k8s
        # Uh. Is it OK that this is case-sensitive?
        assert self.results[1].headers["Access-Control-Allow-Origin"] == [ "https://www.test-cors.org" ]


class CaseSensitive(OptionTest):

    parent: MappingTest

    def config(self):
        yield "case_sensitive: false"

    def queries(self):
        for q in self.parent.queries():
            idx = q.url.find("/", q.url.find("://") + 3)
            upped = q.url[:idx] + q.url[idx:].upper()
            assert upped != q.url
            yield Query(upped)


class AutoHostRewrite(OptionTest):

    parent: MappingTest

    def config(self):
        yield "auto_host_rewrite: true"

    def check(self):
        for r in self.parent.results:
            host = r.backend.request.host
            assert r.backend.name == host, (r.backend.name, host)


class Rewrite(OptionTest):

    parent: MappingTest

    VALUES = ("/foo", "foo")

    def config(self):
        yield self.format("rewrite: {self.value}")

    def queries(self):
        if self.value[0] != "/":
            for q in self.parent.pending:
                q.xfail = "rewrite option is broken for values not beginning in slash"

        return super(OptionTest, self).queries()

    def check(self):
        if self.value[0] != "/":
            pytest.xfail("this is broken")

        for r in self.parent.results:
            assert r.backend.request.url.path == self.value


class TLSOrigination(MappingTest):

    parent: AmbassadorTest
    definition: str

    IMPLICIT = """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: https://{self.target.path.k8s}
"""

    EXPLICIT = """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: {self.target.path.k8s}
tls: true
"""

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for name, dfn in ("IMPLICIT", cls.IMPLICIT), ("EXPLICIT", cls.EXPLICIT):
                yield cls(v, dfn, name="{self.target.name}-%s" % name)

    def init(self, target, definition):
        MappingTest.init(self, target)
        self.definition = definition

    def config(self):
        yield self.target, self.format(self.definition)

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))

    def check(self):
        for r in self.results:
            assert r.backend.request.tls.enabled


class HostRedirectMapping(MappingTest):
    parent: AmbassadorTest
    target: ServiceType

    def init(self):
        MappingTest.init(self, HTTP())

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: foobar.com
host_redirect: true
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/anything?itworked=true"), expected=301)

    def check(self):
        assert self.results[0].headers['Location'] == [
            self.format("http://foobar.com/{self.name}/anything?itworked=true")
        ]


class CanaryMapping(MappingTest):

    parent: AmbassadorTest
    target: ServiceType
    canary: ServiceType
    weight: int

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for w in (10, 50):
                yield cls(v, v.clone("canary"), w, name="{self.target.name}-{self.weight}")

    def init(self, target: ServiceType, canary: ServiceType, weight):
        MappingTest.init(self, target)
        self.canary = canary
        self.weight = weight

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.k8s}
""")
        yield self.canary, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-canary
prefix: /{self.name}/
service: http://{self.canary.path.k8s}
weight: {self.weight}
""")

    def queries(self):
        for i in range(100):
            yield Query(self.parent.url(self.name + "/"))

    def check(self):
        hist = {}

        for r in self.results:
            hist[r.backend.name] = hist.get(r.backend.name, 0) + 1

        canary = 100*hist.get(self.canary.path.k8s, 0)/len(self.results)
        # main = 100*hist.get(self.target.path.k8s, 0)/len(self.results)

        assert abs(self.weight - canary) < 25, (self.weight, canary)


class AmbassadorIDTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield self, """
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config: {}
"""
        for prefix, amb_id in (("findme", "{self.ambassador_id}"),
                               ("findme-array", "[{self.ambassador_id}, missme]"),
                               ("findme-array2", "[missme, {self.ambassador_id}]"),
                               ("missme", "missme"),
                               ("missme-array", "[missme1, missme2]")):
            yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.path.k8s}-{prefix}
prefix: /{prefix}/
service: {self.target.path.k8s}
ambassador_id: {amb_id}
            """, prefix=self.format(prefix), amb_id=self.format(amb_id))

    def queries(self):
        yield Query(self.url("findme/"))
        yield Query(self.url("findme-array/"))
        yield Query(self.url("findme-array2/"))
        yield Query(self.url("missme/"), expected=404)
        yield Query(self.url("missme-array/"), expected=404)


class StatsdTest(AmbassadorTest):
    def init(self):
        self.target = HTTP()
        if DEV:
            self.skip_node = True

    def manifests(self) -> str:
        envs = """
    - name: STATSD_ENABLED
      value: 'true'
"""

        return self.format(AMBASSADOR, image=os.environ["AMBASSADOR_DOCKER_IMAGE"], envs=envs) + GRAPHITE_CONFIG.format('statsd-sink')

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.k8s}
""")

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query("http://statsd-sink/render?format=json&target=summarize(stats_counts.envoy.cluster.cluster_http___statsdtest_http.upstream_rq_200,'1hour','sum',true)&from=-1hour", phase=2)

    def check(self):
        assert 0 < self.results[-1].json[0]['datapoints'][0][0] <= 1000


# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
main = Runner(AmbassadorTest)

