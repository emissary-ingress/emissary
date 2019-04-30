from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, ServiceType


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
service: {self.target.path.fqdn}
""")

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url(self.name + "/"), error=['connection reset by peer', 'EOF'])

    def requirements(self):
        yield from (r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("http://"))


class ClientCertificateAuthentication(AmbassadorTest):
    presto_crt = """
-----BEGIN CERTIFICATE-----
MIIDYTCCAkkCCQCrK74a3GFhijANBgkqhkiG9w0BAQsFADBxMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCTUExDzANBgNVBAcMBkJvc3RvbjERMA8GA1UECgwIRGF0YXdp
cmUxFDASBgNVBAsMC0VuZ2luZWVyaW5nMRswGQYDVQQDDBJtYXN0ZXIuZGF0YXdp
cmUuaW8wIBcNMTkwMTEwMTkxOTUyWhgPMjExODEyMTcxOTE5NTJaMHIxCzAJBgNV
BAYTAklOMQswCQYDVQQIDAJLQTESMBAGA1UEBwwJQmFuZ2Fsb3JlMQ8wDQYDVQQK
DAZQcmVzdG8xFDASBgNVBAsMC0VuZ2luZWVyaW5nMRswGQYDVQQDDBJwcmVzdG8u
ZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCvPcFp
hw5Ja67z23L4YCYTgNdw4eVh7EHyzOpmf3VGhvx/UtNMVOH7Dcf+I7QEyxtQeBiZ
HOcThgr/k/wrAbMjdThRS8yJxRZgj79Li92pKkJbhLGsBeTuw8lBhtwyn85vEZrt
TOWEjlXHHLlz1OHiSAfYChIGjenPu5sT++O1AAs15b/0STBxkrZHGVimCU6qEWqB
PYVcGYqXdb90mbsuY5GAdAzUBCGQH/RLZAl8ledT+uzkcgHcF30gUT5Ik5Ks4l/V
t+C6I52Y0S4aCkT38XMYKMiBh7XzpjJUnR0pW5TYS37wq6nnVFsNReaMKmbOWp1X
5wEjoRJqDrHtVvjDAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAI3LR5fS6D6yFa6b
yl6+U/i44R3VYJP1rkee0s4C4WbyXHURTqQ/0z9wLU+0Hk57HI+7f5HO/Sr0q3B3
wuZih+TUbbsx5jZW5e++FKydFWpx7KY4MUJmePydEMoUaSQjHWnlAuv9PGp5ZZ30
t0lP/mVGNAeiXsILV8gRHnP6aV5XywK8c+828BQDRfizJ+uKYvnAJmqpn4aOOJh9
csjrK52+RNebMT0VxZF4JYGd0k00au9CaciWpPk69C+A/7K/xtV4ZFtddVP9SldF
ahmIu2g3fI5G+/2Oz8J+qX2B+QqT21/pOPKnMQU54BQ6bmI3fBM9B+2zm92FfgYH
9wgA5+Y=
-----END CERTIFICATE-----        
"""

    presto_key = """
-----BEGIN RSA PRIVATE KEY-----
MIIEoQIBAAKCAQEArz3BaYcOSWuu89ty+GAmE4DXcOHlYexB8szqZn91Rob8f1LT
TFTh+w3H/iO0BMsbUHgYmRznE4YK/5P8KwGzI3U4UUvMicUWYI+/S4vdqSpCW4Sx
rAXk7sPJQYbcMp/ObxGa7UzlhI5Vxxy5c9Th4kgH2AoSBo3pz7ubE/vjtQALNeW/
9EkwcZK2RxlYpglOqhFqgT2FXBmKl3W/dJm7LmORgHQM1AQhkB/0S2QJfJXnU/rs
5HIB3Bd9IFE+SJOSrOJf1bfguiOdmNEuGgpE9/FzGCjIgYe186YyVJ0dKVuU2Et+
8Kup51RbDUXmjCpmzlqdV+cBI6ESag6x7Vb4wwIDAQABAoIBAHfXwPS9Mw0NAoms
kzS+9Gs0GqINKoTMQNGeR9Mu6XIBEJ62cuBp0F2TsCjiG9OHXzep2hCkDndwnQbq
GnMC55KhMJGQR+IUEdiZldZBYaa1ysmxtpwRL94FsRYJ9377gP6+SHhutSvw90KD
J2TKumu4nPym7mrjFHpHL6f8BF6b9dJftE2o27TX04+39kPiX4d+4CLfG7YFteYR
98qYHwAk58+s3jJxk7gaDehb0PvOIma02eLF7dNA7h0BtB2h2rfPLNlgKv2MN7k3
NxRHwXEzSCfK8rL8yxQLo4gOy3up+LU7LRERBIkpOyS5tkKcIGoG1w5zEB4sqJZC
Me2ZbUkCgYEA4RGHtfYkecTIBwSCgdCqJYa1zEr35xbgqxOWF7DfjjMwfxeitdh+
U487SpDpoH68Rl/pnqQcHToQWRfLGXv0NZxsQDH5UulK2dLy2JfQSlFMWc0rQ210
v8F35GXohB3vi4Tfrl8wrkEBbCBoZDmp7MPZEGVGb0KVl+gU2u19CwUCgYEAx1Mt
w6M8+bj3ZQ9Va9tcHSk9IVRKx0fklWY0/cmoGw5P2q/Yudd3CGupINGEA/lHqqW3
boxfdneYijOmTQO9/od3/NQRDdTrCRKOautts5zeJw7fUvls5/Iip5ZryR5mYqEz
Q/yMffzZPYVPXR0E/HEnCjf8Vs+0dDa2QwAhDycCf0j4ZgeYxjq0kiW0UJvGC2Qf
SNHzfGxv/md48jC8J77y2cZa42YRyuNMjOygDx75+BDZB+VnT7YqHSLFlBOvHH5F
ONOXYD6BZMM6oYGXtvBha1+yJVS3KCMDltt2LuymyAN0ERF3y1CzwsJLv4y/JVie
JsIqE6v+6oFVvW09kk0CgYEAuazRL7ILJfDYfAqJnxxLNVrp9/cmZXaiB02bRWIp
N3Lgji1KbOu6lVx8wvaIzI7U5LDUK6WVc6y6qtqsKoe237hf3GPLsx/JBb2EbzL6
ENuq0aV4AToZ6gLTp1tm8oVgCLZzI/zI/r+fukBJispyj5n0LP+0D0YSqkMhC06+
fPcCgYB85vDLHorvbb8CYcIOvJxogMjXVasOfSLqtCkzICg4i6qCmLkXbs0qmDIz
bIpIFzUdXu3tu+gPV6ab9dPmpj1M77yu7+QLL7zRy/1/EJaY/tFjWzcuF5tP7jKT
UZCMWuBXFwTbeSQHESs5IWpSDxBGJbSNFmCeyo52Dw/fSYxUEg==
-----END RSA PRIVATE KEY-----        
"""

    ca_cert = """
-----BEGIN CERTIFICATE-----
MIIDuDCCAqCgAwIBAgIJAJ0X57eypBNTMA0GCSqGSIb3DQEBCwUAMHExCzAJBgNV
BAYTAlVTMQswCQYDVQQIDAJNQTEPMA0GA1UEBwwGQm9zdG9uMREwDwYDVQQKDAhE
YXRhd2lyZTEUMBIGA1UECwwLRW5naW5lZXJpbmcxGzAZBgNVBAMMEm1hc3Rlci5k
YXRhd2lyZS5pbzAeFw0xOTAxMTAxOTAzMzBaFw0yNDAxMDkxOTAzMzBaMHExCzAJ
BgNVBAYTAlVTMQswCQYDVQQIDAJNQTEPMA0GA1UEBwwGQm9zdG9uMREwDwYDVQQK
DAhEYXRhd2lyZTEUMBIGA1UECwwLRW5naW5lZXJpbmcxGzAZBgNVBAMMEm1hc3Rl
ci5kYXRhd2lyZS5pbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAOvQ
V5ZwSfrd5VwmzZ9Jch97rQn49p6oQb6EHZ1yOa2evA7165jd0qjKPO2X2FO41X8B
pAaKdLg2imh/p/cW7bgr3G6tGTFU1VGjyeLMDWD50evM62vzX8TnaUzdTGN1Nu36
rZ3bg+EKr8Eb25odZlJr2mf6KRx7Sr6sOSx6Q5TxRosrrftwKcz29pve0d8oCbdi
DROVVc5zAim3scfwupEBkC61vZJ38fiv0DCX9ZgkpLtFJQ9eLEPHGJPjyfewjSSy
/nNv/mRsbziCmCtwgpflTm89c+q3IhomA5axYAQcCCj9po5HUdrmIBJGLAMVy9by
FgdNthWAxvB4vfAyx9sCAwEAAaNTMFEwHQYDVR0OBBYEFGT9P/8pPxb7QRUxW/Wh
izd2sglKMB8GA1UdIwQYMBaAFGT9P/8pPxb7QRUxW/Whizd2sglKMA8GA1UdEwEB
/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAKsVOarsMZIxK9JKS0GTsgEsca8j
YaL85balnwAnpq2YR0cH2XowgKb3r3ufmTB4DsY/Q0iehCJy339Br65P1PJ0h/zf
dFNrvJ4ioX5LZw9bJ0AQND+YQ0E+MttZilOClsO9PBvmmPJuuaeaWoKjVfsN/Tc0
2qLU3ZU0z9nhXx6e9bqaFKIMcbqbVOgKjwWFil9dDn/CoJlaTS4IZ9NhqcS8X1wt
T2md/IKZhKJsp7VPFx59ehngEOjFhphswm1t8gAeq/P7JHZQyAPfXl3rd1RARnER
AJfULDOksXSEodSf+mGCkUhuod/h8LMGWLXzCgtHpJ2wZTp9kVVUkJvJjIU=
-----END CERTIFICATE-----
"""

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return super().manifests() + """
---
apiVersion: v1
metadata:
  name: client-cert-secret
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUR1RENDQXFDZ0F3SUJBZ0lKQUowWDU3ZXlwQk5UTUEwR0NTcUdTSWIzRFFFQkN3VUFNSEV4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4R3pBWkJnTlZCQU1NRW0xaGMzUmxjaTVrCllYUmhkMmx5WlM1cGJ6QWVGdzB4T1RBeE1UQXhPVEF6TXpCYUZ3MHlOREF4TURreE9UQXpNekJhTUhFeEN6QUoKQmdOVkJBWVRBbFZUTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRSwpEQWhFWVhSaGQybHlaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEd6QVpCZ05WQkFNTUVtMWhjM1JsCmNpNWtZWFJoZDJseVpTNXBiekNDQVNJd0RRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFPdlEKVjVad1NmcmQ1Vndtelo5SmNoOTdyUW40OXA2b1FiNkVIWjF5T2EyZXZBNzE2NWpkMHFqS1BPMlgyRk80MVg4QgpwQWFLZExnMmltaC9wL2NXN2JncjNHNnRHVEZVMVZHanllTE1EV0Q1MGV2TTYydnpYOFRuYVV6ZFRHTjFOdTM2CnJaM2JnK0VLcjhFYjI1b2RabEpyMm1mNktSeDdTcjZzT1N4NlE1VHhSb3NycmZ0d0tjejI5cHZlMGQ4b0NiZGkKRFJPVlZjNXpBaW0zc2Nmd3VwRUJrQzYxdlpKMzhmaXYwRENYOVpna3BMdEZKUTllTEVQSEdKUGp5ZmV3alNTeQovbk52L21Sc2J6aUNtQ3R3Z3BmbFRtODljK3EzSWhvbUE1YXhZQVFjQ0NqOXBvNUhVZHJtSUJKR0xBTVZ5OWJ5CkZnZE50aFdBeHZCNHZmQXl4OXNDQXdFQUFhTlRNRkV3SFFZRFZSME9CQllFRkdUOVAvOHBQeGI3UVJVeFcvV2gKaXpkMnNnbEtNQjhHQTFVZEl3UVlNQmFBRkdUOVAvOHBQeGI3UVJVeFcvV2hpemQyc2dsS01BOEdBMVVkRXdFQgovd1FGTUFNQkFmOHdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBS3NWT2Fyc01aSXhLOUpLUzBHVHNnRXNjYThqCllhTDg1YmFsbndBbnBxMllSMGNIMlhvd2dLYjNyM3VmbVRCNERzWS9RMGllaENKeTMzOUJyNjVQMVBKMGgvemYKZEZOcnZKNGlvWDVMWnc5YkowQVFORCtZUTBFK010dFppbE9DbHNPOVBCdm1tUEp1dWFlYVdvS2pWZnNOL1RjMAoycUxVM1pVMHo5bmhYeDZlOWJxYUZLSU1jYnFiVk9nS2p3V0ZpbDlkRG4vQ29KbGFUUzRJWjlOaHFjUzhYMXd0ClQybWQvSUtaaEtKc3A3VlBGeDU5ZWhuZ0VPakZocGhzd20xdDhnQWVxL1A3SkhaUXlBUGZYbDNyZDFSQVJuRVIKQUpmVUxET2tzWFNFb2RTZittR0NrVWh1b2QvaDhMTUdXTFh6Q2d0SHBKMndaVHA5a1ZWVWtKdkpqSVU9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
kind: Secret
type: Opaque
---
apiVersion: v1
kind: Secret
metadata:
  name: client-cert-server-secret
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURaekNDQWs4Q0NRQ3JLNzRhM0dGaGlUQU5CZ2txaGtpRzl3MEJBUXNGQURCeE1Rc3dDUVlEVlFRR0V3SlYKVXpFTE1Ba0dBMVVFQ0F3Q1RVRXhEekFOQmdOVkJBY01Ca0p2YzNSdmJqRVJNQThHQTFVRUNnd0lSR0YwWVhkcApjbVV4RkRBU0JnTlZCQXNNQzBWdVoybHVaV1Z5YVc1bk1Sc3dHUVlEVlFRRERCSnRZWE4wWlhJdVpHRjBZWGRwCmNtVXVhVzh3SGhjTk1Ua3dNVEV3TVRrd056TTRXaGNOTWprd01UQTNNVGt3TnpNNFdqQjZNUXN3Q1FZRFZRUUcKRXdKSlRqRUxNQWtHQTFVRUNBd0NTMEV4RWpBUUJnTlZCQWNNQ1VKaGJtZGhiRzl5WlRFVE1CRUdBMVVFQ2d3SwpRVzFpWVhOellXUnZjakVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEh6QWRCZ05WQkFNTUZtRnRZbUZ6CmMyRmtiM0l1WlhoaGJYQnNaUzVqYjIwd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUIKQVFDN1liY3o5SkZOSHVYY3pvZERrTURvUXd0M1pmQnpjaElwTFlkeHNDZnB1UUYybGNmOGxXMEJKNnZlNU0xTAovMjNZalFYeEFsV25VZ3FZdFlEL1hiZGh3RCtyRWx3RXZWUzR1US9IT2EyUTUwVkF6SXNYa0lxWm00dVA1QzNECk8rQ0NncXJ3UUgzYS8vdlBERldYWkUyeTJvcUdZdE1Xd20zVXQrYnFWSFEzOThqcTNoaGt3MmNXL0pLTjJkR2UKRjk0OWxJWG15NHMrbGE3b21RWldWY0JFcWdQVzJDL1VrZktSbVdsVkRwK0duSk8vZHFobDlMN3d2a2hhc2JETAphbVkweXdiOG9LSjFRdmlvV1JxcjhZZnQ5NzVwaGgzazRlRVdMMUNFTmxFK09vUWNTNVRPUEdndko3WlMyaU43CllVTDRBK0gydCt1WWdUdnFSYVNqcTdnckFnTUJBQUV3RFFZSktvWklodmNOQVFFTEJRQURnZ0VCQUJURGJ4MzkKUGpoT2JpVW1Rdm9vbVhOVjJ1TG1FZkxJcGlKQUhWOTM0VTlmMnhVUS93eExkcElhVXM0WTlRSzhOR2h2U3dSSAp4Y2w4R2hGYzBXRDRoNEJTdmNhdUdVS21LRzh5ZVFhdGhGVjBzcGFHYjUvaFBqUVdDWnNYK3crbjU4WDROOHBrCmx5YkE4akZGdUZlb3R3Z1l6UUhzQUppU29DbW9OQ0ZkaE4xT05FS1FMY1gxT2NRSUFUd3JVYzRBRkw2Y0hXZ1MKb1FOc3BTMlZIbENsVkpVN0E3Mkh4R3E5RFVJOWlaMmYxVnc1Rmpod0dxalBQMDJVZms1Tk9RNFgzNWlrcjlDcApyQWtJSnh1NkZPUUgwbDBmZ3VNUDlsUFhJZndlMUowQnNLZHRtd2wvcHp0TVV5dW5TbURVWEgyR1l5YmdQTlQyCnNMVFF1RFZaR0xmbFJUdz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  tls.key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBdTJHM00vU1JUUjdsM002SFE1REE2RU1MZDJYd2MzSVNLUzJIY2JBbjZia0JkcFhICi9KVnRBU2VyM3VUTlMvOXQySTBGOFFKVnAxSUttTFdBLzEyM1ljQS9xeEpjQkwxVXVMa1B4em10a09kRlFNeUwKRjVDS21adUxqK1F0d3p2Z2dvS3E4RUI5MnYvN3p3eFZsMlJOc3RxS2htTFRGc0p0MUxmbTZsUjBOL2ZJNnQ0WQpaTU5uRnZ5U2pkblJuaGZlUFpTRjVzdUxQcFd1NkprR1ZsWEFSS29EMXRndjFKSHlrWmxwVlE2ZmhweVR2M2FvClpmUys4TDVJV3JHd3kycG1OTXNHL0tDaWRVTDRxRmthcS9HSDdmZSthWVlkNU9IaEZpOVFoRFpSUGpxRUhFdVUKemp4b0x5ZTJVdG9qZTJGQytBUGg5cmZybUlFNzZrV2tvNnU0S3dJREFRQUJBb0lCQVFDbmZrZjViQko1Z2pYcgpzcnliKzRkRDFiSXBMdmpJNk4wczY2S1hUK1BOZW03QlprOVdDdWRkMGUxQ2x2aWZoeG5VS1BKM3BTT1ZKYk9OCkh5aklteWV4ZTl3dGVZTEJSYysyTXMzVXdrelFLcm52bXlaMWtPRWpQek40RW5tSmV6dEt6YXdvaHkwNGxmcXEKNzVhT2RiMHlNMEVCc05LSkZKQ0NSVVJtajhrMndJQXIwbHFhV0ZNcGlYT3FzTXBvWTZMY3plaGlMZHU0bUFaSQpRRHhCM3dLVGpmdGNIdzcxTmFKZlg5V2t2OFI4ZWlqeWpNOUl2Y1cwZmRQem9YVTBPZEFTa09ZRlFIZHlCUFNiCjllNWhDSGFJczZia1hBOEs4YmZRazBSL0d6STcyVXArd0JrbnJnTlhZTXFudHJSa0ljNURER1g0b3VOc2lqUkoKSWtrWER2TjVBb0dCQU8veFQrNTYyQ2hwc3R2NUpvMi9ycFdGb05tZ3ZJT0RMRGxiamhHZEpqKytwNk1BdjFQWgo2d042WnozMmppUG1OYzdCK2hrQm40RFQvVkFpU3NLRG1SK09tUkg1TVNzQXh6aWRxU3lNcldxdG1lMDNBVzd6Cklja0FNTGdwWHhDdW1HMzRCM2Jxb3VUdGVRdm5WcmRlR2hvdUJ5OUJSMVpXbnRtWHVscVhyNUFmQW9HQkFNZnIKN29NVGwzdUVVeml5a0IzYmkxb0RYdUNjN01Qc3h0c1IwdElqZXc3RStwTGoyaUxXZUZuMGVhdnJYaHQ1ODRJbwpDZG90a1ZMMHhrZ1g3M2ZremxEd1hobTJVTXBaQmxzSzBnR09SaUYzd0ZMU0hJNmxRUmJkaXRIb0JqcDRGTEZzCitlanZKUDZ1ZitBekZ5cjBLTnc3TnpyaCthbFhFQ09RS2NqUXJlWjFBb0dBQXRLZzhScEszcmJYbnRUZ2lqeGUKRG01REJTeHA2MVlvdUFnR3ROaFhjZHFKV0ZhUzZhYWZxQ3ZSZVI0a2IvR3VZbDlQMU9sNitlWUVqZVBKWTE1dQo5N3NTdSs1bGtLN3lxUXpaeDZka0J1UkI4bE42VmRiUVorL3pvc2NCMGsxcmg2ZXFWdEROMThtZmFlOXZ5cnAxCnJpY3FlSGpaSVAvbDRJTnpjc3RrQ2xzQ2dZQmh5TVZkZVZ5emZuS1NIY3lkdmY5MzVJUW9pcmpIeiswbnc1MEIKU1hkc0x1NThvRlBXakY1TGFXZUZybGJXUzV6T1FiVW44UGZPd29pbFJJZk5kYTF3SzFGcmRDQXFDTWN5Q3FYVApPdnFVYmhVMHJTNW9tdTJ1T0dnbzZUcjZxRGMrM1JXVFdEMFpFTkxkSDBBcXMwZTFDSVdvR0ZWYi9ZaVlUSEFUCmwvWW03UUtCZ1FEcFYvSjRMakY5VzBlUlNXenFBaDN1TStCdzNNN2NEMUxnUlZ6ZWxGS2w2ZzRBMWNvdU8wbHAKalpkMkVMZDlzTHhBVENVeFhQZ0dDTjY0RVNZSi92ZUozUmJzMTMrU2xqdjRleTVKck1ieEhNRC9CU1ovY2VjaAp4aFNWNkJsMHVKb2tlMTRPMEJ3OHJzSUlxZTVZSUxqSlMwL2E2eTllSlJtaGZJVG9PZU5PTUE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
"""

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
service: {self.target.path.fqdn}
""")

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url(self.name + "/"), insecure=True, client_crt=self.presto_crt, client_key=self.presto_key, client_cert_required=True, ca_cert=self.ca_cert)

        yield Query(self.url(self.name + "/"), insecure=True, error="handshake failure")

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.insecure = True
            query.client_cert = self.presto_crt
            query.client_key = self.presto_key
            query.client_cert_required = True
            query.ca_cert = self.ca_cert
            yield (r[0], query)


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
    cert_chain_file: /ambassador/snapshots/default/secrets-decoded/test-certs-secret/F94E4DCF30ABC50DEF240AA8024599B67CC03991.crt
    private_key_file: /ambassador/snapshots/default/secrets-decoded/test-certs-secret/F94E4DCF30ABC50DEF240AA8024599B67CC03991.key
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: {self.target.path.fqdn}
tls: upstream
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}-files
prefix: /{self.name}-files/
service: {self.target.path.fqdn}
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
service: {self.target.path.fqdn}
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
service: {self.target.path.fqdn}
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
            "TLSContext server found no certificate in secret test-certs-secret-invalid in namespace default, ignoring...",
            "TLSContext bad-path-info found no cert_chain_file '/nonesuch'",
            "TLSContext bad-path-info found no private_key_file '/nonesuch'",
            "TLSContext validation-without-termination found no certificate in secret test-certs-secret-invalid in namespace default, ignoring...",
            "TLSContext missing-secret-key: 'cert_chain_file' requires 'private_key_file' as well",
        }

        for errsvc, errtext in errors:
            if errtext in wanted:
                found += 1

        assert found == len(errors), "unexpected errors in list"


class TLSContext(AmbassadorTest):
    # debug = True

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
service: http://{self.target.path.fqdn}
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
service: http://{self.target.path.fqdn}
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
kind: Module
name: tls
config:
  server:
    enabled: True
    secret: test-certs-secret 
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-other-mapping
prefix: /{self.name}/
service: https://{self.target.path.fqdn}
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
        # 0
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"),
                    headers={"Host": "tls-context-host-2"},
                    insecure=True,
                    sni=True)

        # 1 - Correct host #1
        yield Query(self.url("tls-context-same/"),
                    headers={"Host": "tls-context-host-1"},
                    expected=200,
                    insecure=True,
                    sni=True)
        # 2 - Correct host #2
        yield Query(self.url("tls-context-same/"),
                    headers={"Host": "tls-context-host-2"},
                    expected=200,
                    insecure=True,
                    sni=True)

        # 3 - Incorrect host
        yield Query(self.url("tls-context-same/"),
                    headers={"Host": "tls-context-host-3"},
                    # error=self._go_close_connection_error(self.url("tls-context-same/")),
                    expected=404,
                    insecure=True)

        # 4 - Incorrect path, correct host
        yield Query(self.url("tls-context-different/"),
                    headers={"Host": "tls-context-host-1"},
                    expected=404,
                    insecure=True,
                    sni=True)

        # Other mappings with no host will respond with the fallbock cert.
        # 5 - no Host header, fallback cert from the TLS module
        yield Query(self.url(self.name + "/"),
                    # error=self._go_close_connection_error(self.url(self.name + "/")),
                    insecure=True)

        # 6 - explicit Host header, fallback cert
        yield Query(self.url(self.name + "/"),
                    # error=self._go_close_connection_error(self.url(self.name + "/")),
                    # sni=True,
                    headers={"Host": "tls-context-host-3"},
                    insecure=True)

        # 7 - explicit Host header 1 wins, we'll get the SNI cert for this overlapping path
        yield Query(self.url(self.name + "/"),
                    headers={"Host": "tls-context-host-1"},
                    expected=200,
                    insecure=True,
                    sni=True)

        # 7 - explicit Host header 2 wins, we'll get the SNI cert for this overlapping path
        yield Query(self.url(self.name + "/"),
                    headers={"Host": "tls-context-host-2"},
                    expected=200,
                    insecure=True,
                    sni=True)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json
        assert (len(errors) == 0)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers:
                host_header = result.query.headers['Host']
                tls_common_name = result.tls[0]['Issuer']['CommonName']

                # XXX Weirdness with the fallback cert here! You see, if we use host
                # tls-context-host-3 (or, really, anything except -1 or -2), then the
                # fallback cert actually has CN 'localhost'. We should replace this with
                # a real fallback cert, but for now, just hack the host_header.
                #
                # Ew.

                if host_header == 'tls-context-host-3':
                    host_header = 'localhost'

                assert host_header == tls_common_name, "test %d wanted CN %s, but got %s" % (idx, host_header, tls_common_name)

            idx += 1

    def requirements(self):
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))
