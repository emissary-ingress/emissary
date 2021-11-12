from base64 import b64encode
from typing import Dict, List, NamedTuple, Optional

class Cert(NamedTuple):
    names: List[str]
    pubcert: str
    privkey: str

    @property
    def k8s_crt(self) -> str:
        return b64encode((self.pubcert+"\n").encode('utf-8')).decode('utf-8')

    @property
    def k8s_key(self) -> str:
        return b64encode((self.privkey+"\n").encode('utf-8')).decode('utf-8')

def strip(s: str) -> str:
    return "\n".join(l.strip() for l in s.split("\n") if l.strip())

_TLSCerts: List[Cert] = [
    Cert(
        names=["master.datawire.io"],
        # Note: This cert is also used to sign several other certs in
        # this file (as the issuer).
        pubcert=strip("""
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
            """),
        privkey=""
    ),

    Cert(
        names=["presto.example.com"],
        # Note: This cert is signed by the "master.datawire.io" cert
        # (rather than being self-signed).
        pubcert=strip("""
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
            """),
        privkey=strip("""
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
        """)
    ),

    Cert(
        names=["ratelimit.datawire.io"],
        pubcert=strip("""
            -----BEGIN CERTIFICATE-----
            MIIDCjCCAfICCQDgXR6wWVzZODANBgkqhkiG9w0BAQUFADBHMR4wHAYDVQQDDBVy
            YXRlbGltaXQuZGF0YXdpcmUuaW8xJTAjBgkqhkiG9w0BCQEWFmhvc3RtYXN0ZXJA
            ZGF0YXdpcmUuaW8wHhcNMTkwOTE5MTgzMzAyWhcNMjEwODE5MTgzMzAyWjBHMR4w
            HAYDVQQDDBVyYXRlbGltaXQuZGF0YXdpcmUuaW8xJTAjBgkqhkiG9w0BCQEWFmhv
            c3RtYXN0ZXJAZGF0YXdpcmUuaW8wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEK
            AoIBAQCyl9VBmV5BpV18vsrSjRKrdiDVvYKGq6VUhaSE9YIcQ88bI1ZxyaPOsQYQ
            c2rf8CDJuJx3XhR530Cp7zP6eR23p2FA9+1Ik9HvaYLtX4mA2Nv8xWY1hHnkPTFs
            TTpk0LDGXb5YZxvBG353J/sE+1kIE1zzJfWiAJT36s2NOG4UAhhVaOKju+x+arpm
            2fsaNWJM1D1/BQuU7UbwJtBb2dZ6YKT4q83ghAh2l8ZwXPpWIBkiXjM5rtZD7Bcx
            DqtSmTMYz5cecpnhb4L8gxEUBrZTqCx8EY/p0+cNf7hRraVm7zCpZD8oIKKKB/P4
            3WGdtG6ZGKEEJkW60yAkqFb7s3VtAgMBAAEwDQYJKoZIhvcNAQEFBQADggEBAAIS
            vfxu+wPqLbqTYWSKMKzKrf5LqZZAdZYesHGJ16arVkzxhspIFDfinVuRb5xDqYUR
            qmkE5+GB48yPj1iD/uwO8B4rAg5mOzL50JT05POWrsHc+Yh/0/QzF6ij4yU5fzvZ
            DzItPb4NjY31PN8ZBLa1FHy9wbFK7rKiV+c1O7tPucA5TZz1KHyUEXilGvchs0x8
            sk4IGlkU3hH1UXpsNl4528V4t/wW97bbhExv/bppGtKn4Jz/abCRr07Cy3S+cS79
            yP56bAVqkTvM5dRHb4os8f35BnA7Or/XE46+eQ1sUcmHTnNQgEVJVdepGuxJNi4S
            z2tXxGI+Su1eRyPqSFY=
            -----END CERTIFICATE-----
            """),
        privkey=strip("""
            -----BEGIN PRIVATE KEY-----
            MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCyl9VBmV5BpV18
            vsrSjRKrdiDVvYKGq6VUhaSE9YIcQ88bI1ZxyaPOsQYQc2rf8CDJuJx3XhR530Cp
            7zP6eR23p2FA9+1Ik9HvaYLtX4mA2Nv8xWY1hHnkPTFsTTpk0LDGXb5YZxvBG353
            J/sE+1kIE1zzJfWiAJT36s2NOG4UAhhVaOKju+x+arpm2fsaNWJM1D1/BQuU7Ubw
            JtBb2dZ6YKT4q83ghAh2l8ZwXPpWIBkiXjM5rtZD7BcxDqtSmTMYz5cecpnhb4L8
            gxEUBrZTqCx8EY/p0+cNf7hRraVm7zCpZD8oIKKKB/P43WGdtG6ZGKEEJkW60yAk
            qFb7s3VtAgMBAAECggEAKxo77NYgCoXnlzjQ6JoFnH4pFIzlWK1KfKi4eSJroXi4
            Hlub/GBm+XZ9+TBx5dQlhanZkXGSTYuVJq5FhDkA9BcggLaVfQO4EikL4VBCdmdg
            SJQ3w8jSRkSCjhnhcv1u/KEZVGqmJygEkKuEiMJEzY8mysQpkUzEp0TzERdCce9c
            SJOd6WgFaGs7vfLJLj0fNMRc+pr1NFpfq94GirdUrAB/8xe/iiOxgerh3ROGB8gS
            ejLy6fdHY+F5wg2DKE25cxw+uaTxJ80InrKVa7aV9jxiu/F/Yn6kAFOm5qAZLGIQ
            7l5CcGLX+Ju/zPciHBQElvGgRLgYL+qXsOa+rguogQKBgQDZUSnqAGowffkxuA0v
            uqsN63sZ0qUUC+h0uhfBjuudlDeO+3e9X/gqmoLqgbjKmCqpvtxTJOdYmidLn6C+
            SvH2L9WtA75E00ul7VoED9k9KW9hE83Vh3HIrz0Ctm/0/0KIiYWoedil6bKeNwTH
            Wr/LvR93avtz65I0UaAV5r28jQKBgQDSYhKduc0uaD6Un95m2NfIzTZ5lZ6fYtQr
            FGo0rYiSzVBFVk6twjB9vhmvkcf0MSmpCSOQFFr2wfOTdkANyqMlp5wG+viF/Kxc
            qsLCTQ8hp6agk33DOLTIv9zJvHgSclYoi96j6OYso+YKHNLfMMcUl7u8jbfwalvw
            xNz7ZGQUYQKBgHWledJjbRlZaUlgQUtAfA/qFldxcMq8c5iVkfzIOYeyUK2IN1d/
            F+NAiHUZywdqf1YrrC0awl91/KX1Adliy0CivsOOTjgGR2LJbrzaM5nnz5M3XGwn
            ihLBw36vc0an1cYC5SfC5uVS8c6zLFQcLc7HULyeXwhvVFQciFSy+K6VAoGARANm
            p00A8ybKTHwehztFD2qgWNAw9rAZjU/NQfhz9ZmggLn1N6FW0d/aJ/NGJECcikQl
            FhgujCWJnDuXW54N/kdgXrVWEOLtygt+aRhGcwfjC3iDKNC1SU0VkLZ4TuZdyj/l
            mzHY78eQv+Yvme4H/jVLgRqDw5pu3LiYBEGhRSECgYEA1vcFZk19uKKBRX8EFX3m
            aNY6AeaY5IWlUl1IlJGPpE6pgz8uiTX5OihnJlhx423v0HEMPWFq5t3p+ItdJgn3
            ySCDMqxreJ8s+BKpjyG0Tqc7JdnJTrmM7QT0+UQq1biA8CbSAo1erlripFunULnQ
            IzwIDr3Us7DNBLMfiNh6nUk=
            -----END PRIVATE KEY-----
            """)
    ),

    Cert(
        names=["ambassador.example.com"],
        # Note: This cert is signed by the "master.datawire.io" cert
        # (rather than being self-signed).
        pubcert=strip("""
            -----BEGIN CERTIFICATE-----
            MIIDZzCCAk8CCQCrK74a3GFhiTANBgkqhkiG9w0BAQsFADBxMQswCQYDVQQGEwJV
            UzELMAkGA1UECAwCTUExDzANBgNVBAcMBkJvc3RvbjERMA8GA1UECgwIRGF0YXdp
            cmUxFDASBgNVBAsMC0VuZ2luZWVyaW5nMRswGQYDVQQDDBJtYXN0ZXIuZGF0YXdp
            cmUuaW8wHhcNMTkwMTEwMTkwNzM4WhcNMjkwMTA3MTkwNzM4WjB6MQswCQYDVQQG
            EwJJTjELMAkGA1UECAwCS0ExEjAQBgNVBAcMCUJhbmdhbG9yZTETMBEGA1UECgwK
            QW1iYXNzYWRvcjEUMBIGA1UECwwLRW5naW5lZXJpbmcxHzAdBgNVBAMMFmFtYmFz
            c2Fkb3IuZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIB
            AQC7Ybcz9JFNHuXczodDkMDoQwt3ZfBzchIpLYdxsCfpuQF2lcf8lW0BJ6ve5M1L
            /23YjQXxAlWnUgqYtYD/XbdhwD+rElwEvVS4uQ/HOa2Q50VAzIsXkIqZm4uP5C3D
            O+CCgqrwQH3a//vPDFWXZE2y2oqGYtMWwm3Ut+bqVHQ398jq3hhkw2cW/JKN2dGe
            F949lIXmy4s+la7omQZWVcBEqgPW2C/UkfKRmWlVDp+GnJO/dqhl9L7wvkhasbDL
            amY0ywb8oKJ1QvioWRqr8Yft975phh3k4eEWL1CENlE+OoQcS5TOPGgvJ7ZS2iN7
            YUL4A+H2t+uYgTvqRaSjq7grAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABTDbx39
            PjhObiUmQvoomXNV2uLmEfLIpiJAHV934U9f2xUQ/wxLdpIaUs4Y9QK8NGhvSwRH
            xcl8GhFc0WD4h4BSvcauGUKmKG8yeQathFV0spaGb5/hPjQWCZsX+w+n58X4N8pk
            lybA8jFFuFeotwgYzQHsAJiSoCmoNCFdhN1ONEKQLcX1OcQIATwrUc4AFL6cHWgS
            oQNspS2VHlClVJU7A72HxGq9DUI9iZ2f1Vw5FjhwGqjPP02Ufk5NOQ4X35ikr9Cp
            rAkIJxu6FOQH0l0fguMP9lPXIfwe1J0BsKdtmwl/pztMUyunSmDUXH2GYybgPNT2
            sLTQuDVZGLflRTw=
            -----END CERTIFICATE-----
            """),
        privkey=strip("""
            -----BEGIN RSA PRIVATE KEY-----
            MIIEpAIBAAKCAQEAu2G3M/SRTR7l3M6HQ5DA6EMLd2Xwc3ISKS2HcbAn6bkBdpXH
            /JVtASer3uTNS/9t2I0F8QJVp1IKmLWA/123YcA/qxJcBL1UuLkPxzmtkOdFQMyL
            F5CKmZuLj+QtwzvggoKq8EB92v/7zwxVl2RNstqKhmLTFsJt1Lfm6lR0N/fI6t4Y
            ZMNnFvySjdnRnhfePZSF5suLPpWu6JkGVlXARKoD1tgv1JHykZlpVQ6fhpyTv3ao
            ZfS+8L5IWrGwy2pmNMsG/KCidUL4qFkaq/GH7fe+aYYd5OHhFi9QhDZRPjqEHEuU
            zjxoLye2Utoje2FC+APh9rfrmIE76kWko6u4KwIDAQABAoIBAQCnfkf5bBJ5gjXr
            sryb+4dD1bIpLvjI6N0s66KXT+PNem7BZk9WCudd0e1ClvifhxnUKPJ3pSOVJbON
            HyjImyexe9wteYLBRc+2Ms3UwkzQKrnvmyZ1kOEjPzN4EnmJeztKzawohy04lfqq
            75aOdb0yM0EBsNKJFJCCRURmj8k2wIAr0lqaWFMpiXOqsMpoY6LczehiLdu4mAZI
            QDxB3wKTjftcHw71NaJfX9Wkv8R8eijyjM9IvcW0fdPzoXU0OdASkOYFQHdyBPSb
            9e5hCHaIs6bkXA8K8bfQk0R/GzI72Up+wBknrgNXYMqntrRkIc5DDGX4ouNsijRJ
            IkkXDvN5AoGBAO/xT+562Chpstv5Jo2/rpWFoNmgvIODLDlbjhGdJj++p6MAv1PZ
            6wN6Zz32jiPmNc7B+hkBn4DT/VAiSsKDmR+OmRH5MSsAxzidqSyMrWqtme03AW7z
            IckAMLgpXxCumG34B3bqouTteQvnVrdeGhouBy9BR1ZWntmXulqXr5AfAoGBAMfr
            7oMTl3uEUziykB3bi1oDXuCc7MPsxtsR0tIjew7E+pLj2iLWeFn0eavrXht584Io
            CdotkVL0xkgX73fkzlDwXhm2UMpZBlsK0gGORiF3wFLSHI6lQRbditHoBjp4FLFs
            +ejvJP6uf+AzFyr0KNw7Nzrh+alXECOQKcjQreZ1AoGAAtKg8RpK3rbXntTgijxe
            Dm5DBSxp61YouAgGtNhXcdqJWFaS6aafqCvReR4kb/GuYl9P1Ol6+eYEjePJY15u
            97sSu+5lkK7yqQzZx6dkBuRB8lN6VdbQZ+/zoscB0k1rh6eqVtDN18mfae9vyrp1
            ricqeHjZIP/l4INzcstkClsCgYBhyMVdeVyzfnKSHcydvf935IQoirjHz+0nw50B
            SXdsLu58oFPWjF5LaWeFrlbWS5zOQbUn8PfOwoilRIfNda1wK1FrdCAqCMcyCqXT
            OvqUbhU0rS5omu2uOGgo6Tr6qDc+3RWTWD0ZENLdH0Aqs0e1CIWoGFVb/YiYTHAT
            l/Ym7QKBgQDpV/J4LjF9W0eRSWzqAh3uM+Bw3M7cD1LgRVzelFKl6g4A1couO0lp
            jZd2ELd9sLxATCUxXPgGCN64ESYJ/veJ3Rbs13+Sljv4ey5JrMbxHMD/BSZ/cech
            xhSV6Bl0uJoke14O0Bw8rsIIqe5YILjJS0/a6y9eJRmhfIToOeNOMA==
            -----END RSA PRIVATE KEY-----
            """)
    ),

    Cert(
        names=["tls-context-host-2"],
        pubcert=strip("""
            -----BEGIN CERTIFICATE-----
            MIIDgDCCAmigAwIBAgIJAIHY67pShgsrMA0GCSqGSIb3DQEBCwUAMFUxCzAJBgNV
            BAYTAlVTMQswCQYDVQQIDAJNQTEPMA0GA1UEBwwGQm9zdG9uMQswCQYDVQQKDAJE
            VzEbMBkGA1UEAwwSdGxzLWNvbnRleHQtaG9zdC0yMB4XDTE4MTEwMTE0MDQxNloX
            DTI4MTAyOTE0MDQxNlowVTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAk1BMQ8wDQYD
            VQQHDAZCb3N0b24xCzAJBgNVBAoMAkRXMRswGQYDVQQDDBJ0bHMtY29udGV4dC1o
            b3N0LTIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDcA8Yth/PWaOGS
            oNmvEHZ24jQ7PKN+D4owLHWeiuRdmhA0YOvU3wqG3VqY4ZplZAV0PKlD/+2ZSF14
            z8w1eF4QTzZaYxwy9+wfHNkTDEpMjP8JM2OEbykURxURvW4+7D30E2Ez5OPlxmc0
            MYM//JH5EDQhciDrlYqe1SRMRALZeVmkaAyu6NHJTBuj0SIPudLTch+90q+rdwny
            fkT1x3OTanbWjnomEJe7Mvy4mvvqqIHu48S9C8Zd1BGVPbu8V/UDrSWQ9zYCX4SG
            Oasl8C0XmH6kemhPDlD/Tv0xvyH5q5MUcHi4mJtN+gzob54DwzVGEjef5LeS1V5F
            0TAP0d+XAgMBAAGjUzBRMB0GA1UdDgQWBBQdF0GQHdqlthdnQYqViumErlROfDAf
            BgNVHSMEGDAWgBQdF0GQHdqlthdnQYqViumErlROfDAPBgNVHRMBAf8EBTADAQH/
            MA0GCSqGSIb3DQEBCwUAA4IBAQAmAKbCluHEe/IFbuAbgx0Mzuzi90wlmAPb8gmO
            qvbp29uOVsVSmQAddPndFaMXVp1ZhmTV5CSQtdX2CVMW+0W47C/COBdoSEQ9yjBf
            iFDclxm8AN2PmaGQa+xoOXgZLXerChNKWBSZR+ZKXLJSM9UaESlHf5unBLEpCj+j
            dBiIqFca7xIFPkr+0REoAVc/xPnnsaKjL2UygGjQfFNxcON6cucb6LKJXOZEITb5
            H68JuaRCKrefY+IyhQVVNmii7tMpcU2KjW5pkVKqU3dKItEq2VkSdzMUKjNxYwqF
            Yzbz34T50CWnoGmNRAWJsLeViOYU26dwbAWd9Ub+V01Qjn78
            -----END CERTIFICATE-----
            """),
        privkey=strip("""
            -----BEGIN PRIVATE KEY-----
            MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDcA8Yth/PWaOGS
            oNmvEHZ24jQ7PKN+D4owLHWeiuRdmhA0YOvU3wqG3VqY4ZplZAV0PKlD/+2ZSF14
            z8w1eF4QTzZaYxwy9+wfHNkTDEpMjP8JM2OEbykURxURvW4+7D30E2Ez5OPlxmc0
            MYM//JH5EDQhciDrlYqe1SRMRALZeVmkaAyu6NHJTBuj0SIPudLTch+90q+rdwny
            fkT1x3OTanbWjnomEJe7Mvy4mvvqqIHu48S9C8Zd1BGVPbu8V/UDrSWQ9zYCX4SG
            Oasl8C0XmH6kemhPDlD/Tv0xvyH5q5MUcHi4mJtN+gzob54DwzVGEjef5LeS1V5F
            0TAP0d+XAgMBAAECggEBAI6Sr4jv0dZ+jra7H3VvwKTXfytn5zaYkV8YYHwF22jA
            noGi0RBYHPU6Wiw5/hh4EYS6jqGvJmQvXcsdNWLtBl+hRUKbeTmaKVwcEJtkWn1y
            3Q40S+gVNNScH44oaFnEM32IVXQQfpJ22IgdEcWUQW/ZzT5jO+wOMw8sZeI6LHKK
            Gh8ClT9+De/uqjn3BFt0zVwrqKnYJIMCIakoiCFkHphUMDE5Y2SSKhaFZwq1kKwK
            tqoXZJBysaxgQ1QkmfKTgFLyZZWOMfG5soUkSTSyDEG1lbuXpzTm9UI9JSil+Mrh
            u/5SypK8pBHxAtX9UwbN1bDl7Jx5Ibr2sh3AuP1x9JECgYEA8tcS3OTEsNPZPfZm
            OciGn9oRM7GVeFv3+/Nb/rhtzu/TPQbAK8VgqkKGOk3F7Y+ckqKSSZ1gRAvHpldB
            i64cGZOWi+Mc1fUpGUWk1vvWlmgMIPV5mlZo8z02SSuxKe25ceMoOhzqek/oFamv
            2NlEy8ttHN1LLKx+fYa2JFqerq8CgYEA5/ALGIukSrt+GdzKI/5yr7RDJSW23Q2x
            C9fITMAR/T8w3lXhrRtWriG/ytBEO5wS1R0t92umgVHHE09xQWo6tNmzAPMoTRzC
            wO2brjBKAuBdCDHJ6l2Qg8HOAj/Rw++lxlCtTB6a/1XFHfsGPhj0D+ZRbYVshM4R
            gIUfvjfCV5kCgYEA37s/abxrau8Di47kCACz57qldpb6OvWgt8Qy0a9hm/JhECyU
            CL/Em5jGyhi1bnWNr5uQY7pW4tpnit2BSguTYA0V+rO38Xf58Yq0oE1OGyypYARJ
            kORjtRaEWU2j4BlhbYf3m/LgJOhRzwOTO5qRQ6GcWaeYhwQ1VbkzPrMu14kCgYBo
            xtHcZsjzUbvnpwxSMlJQ+ZgTofP37IV8miBMO8BkrTVAW3+1mdIQnAJudqM8YodH
            awUm7qSrauwJ1yuMp5aZuHbbCP29yC5axXw8tmfY4M5mM0fJ7jaortaHwZjbcNls
            u2luJ61Rh8eigZISX2dx/1PtrAaYABd7/aeXYM0UkQKBgQCUnAHvdPPhHVrCYMkN
            N8PD+KtbhOFK6Ks/vX2RG2FqfBBOAewlJ5wLVxPKOTiw+JKaRxxX2G/DFU77n8D/
            DyWdc3fBAd4kYIjfUhdFkXG4ALP6A5QHeSx73Rq1K5lLUhOlFjsuOgCJKo0VQfD/
            ONih0zK3yZg7h5PjfuMGFoONAg==
            -----END PRIVATE KEY-----
            """)
    ),

    Cert(
        names=["tls-context-host-1"],
        pubcert=strip("""
            -----BEGIN CERTIFICATE-----
            MIIDgDCCAmigAwIBAgIJAJrqItzF610iMA0GCSqGSIb3DQEBCwUAMFUxCzAJBgNV
            BAYTAlVTMQswCQYDVQQIDAJNQTEPMA0GA1UEBwwGQm9zdG9uMQswCQYDVQQKDAJE
            VzEbMBkGA1UEAwwSdGxzLWNvbnRleHQtaG9zdC0xMB4XDTE4MTEwMTEzNTMxOFoX
            DTI4MTAyOTEzNTMxOFowVTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAk1BMQ8wDQYD
            VQQHDAZCb3N0b24xCzAJBgNVBAoMAkRXMRswGQYDVQQDDBJ0bHMtY29udGV4dC1o
            b3N0LTEwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC9OgC8wxyIrPzo
            GXsLpPKt72DEx2wjwW8nXW2wWbya3c96n2nSCKPBn85hbqshzj5ihSTAMDIog9Fv
            G6RKWUPXT4KIkTv3CDHqXsApJlJ4lSynQyo2Zv0o+Af8CLngYZB+rfztZwyeDhVp
            wYzBV235zz6+2qbVmCZlvBuXbUqTlEYYvuGlMGz7pPfOWKUpeYodbG2fb+hFFpeo
            CxkUXrQsOoR5JdHG5jWrZuBO455SsrzBL8Rle5UHo05WcK7bBbiQz106pHCJYZ+p
            vlPIcNSX5Kh34Fg96UPx9lQiA3zDTKBfyWcLQ+q1cZlLcWdgRFcNBirGB/7ra1VT
            gEJeGkPzAgMBAAGjUzBRMB0GA1UdDgQWBBRDVUKXYblDWMO171BnYfabY33CETAf
            BgNVHSMEGDAWgBRDVUKXYblDWMO171BnYfabY33CETAPBgNVHRMBAf8EBTADAQH/
            MA0GCSqGSIb3DQEBCwUAA4IBAQAPO/D4Tt52XrlCCfS6gUEdENCrpAWNXDroGc6t
            STlwh/+QLQbNadKehKbf89rXKj+nUqtq/NRZPIsAK+WUkG9ZPoQO8PQiV4WX5rQ7
            29uKcJfaBXkdzUW7qNQhE4c8BasBrIesrkjpT99QxJKnXQaN+Sw7oFPUIAN38Gqa
            Wl/KPMTtbqkwyacKMBmq1VLzvWJoH5CizJJwhnkXxtWKs/67rTNnPVMz+meGtvSi
            dqX6WSSmGLFEEr2hgUcAZjk3VuQh/75hXu+U2ItsC+5qplhG7CXsoXnKKy1XlOAE
            b8kr2dWWEk6I5Y6nTJzWIlSTkW89xwXrctmN9sb9q4SniVlz
            -----END CERTIFICATE-----
            """),
        privkey=strip("""
            -----BEGIN PRIVATE KEY-----
            MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC9OgC8wxyIrPzo
            GXsLpPKt72DEx2wjwW8nXW2wWbya3c96n2nSCKPBn85hbqshzj5ihSTAMDIog9Fv
            G6RKWUPXT4KIkTv3CDHqXsApJlJ4lSynQyo2Zv0o+Af8CLngYZB+rfztZwyeDhVp
            wYzBV235zz6+2qbVmCZlvBuXbUqTlEYYvuGlMGz7pPfOWKUpeYodbG2fb+hFFpeo
            CxkUXrQsOoR5JdHG5jWrZuBO455SsrzBL8Rle5UHo05WcK7bBbiQz106pHCJYZ+p
            vlPIcNSX5Kh34Fg96UPx9lQiA3zDTKBfyWcLQ+q1cZlLcWdgRFcNBirGB/7ra1VT
            gEJeGkPzAgMBAAECggEABal7pipMa0aJ1sQUa3fHDy9PfPPep389PTdNde5pd1TV
            xXyJpRA/HicS/NVb54oNUdNcEygeCBpRpPp1wwfCwOmPJVj7K1wiajnllBWieBs2
            l9app7ETOCubyY3VSgKBWVkJbW0c8onHWD/DX3GnR8dMwFc4kMGZvIeRZ8mMZrgG
            6Ot3J8r6yVleb68auZkgys0GeFszMuTnlrB8L9v25QKcTkDJ2/ElucZyhDtxat8B
            O6NRsnncr8xpQWOr/lWs9UQndGBtqsms+tcT7VT5OTjt8Xv95XMHpygoiLy7s8oc
            I0jk42Zo4JenIOw6Fm0yADgA7yiWrK4lI3Xhji5RoQKBgQDdjicdMJXUFVso+52d
            E0Oa0pJU0SRh/IBgoG7MjHkUlbiyiGZMjp90J9TqZ/Q+3ZVeWj2lOIat8ngS0z00
            W07OVqazk1SXhVeckF5aDrnOD3aSeV1+aWrTt1WEgj9QqbrYaP9zgxRJdG5wXCBP
            F43Eyq9dHW9azI+wPyICBj6vAwKBgQDapSzXOGeb2/R1heYwVWn4s3FdKXV83zkS
            qRX7zwZKvI98c2l55cVMS0hLc4m5O1vBiGyHo4y0vIP/GI4G9xOQa1wiVsfQPbIN
            /2OH1g5rKHWBYRThvFpDjtrQSlrDucYCRDLBwXTp1kmPd/Ofcarln626DjkafYbx
            wue2vXBMUQKBgBn/NiO8sbgDEYFLlQD7Y7FlB/qf188Pm9i6uoWR7hs2PkfkrWxK
            R/ePPPKMZCKESaSaniUm7ta2XtSGqOXd2O9pR4JGxWRKJy+d2RRkKfU950Hkr83H
            fNt+5a/4wIksgVonZ+Ib/WNpIBRbGwds0hvHVLBuZcSXwDyEC++E4BIVAoGBAJ1Q
            zyyjZtjbr86HYxJPwomxAtYXKHOKYRQuGKUvVcWcWlke6TtNuWGloQS4wtVGAkUD
            laMaZ/j60rZOwpH8YFU/CfGjIu2QFna/1Ks9tu4fFDzczxuEXCXTuVi4xwmgtvmW
            fDawrSA6kH7rvZxxOpcxBtyhszB+NQPqSrJPJ2ehAoGAdtRJjooSIiaDUSneeG2f
            TNiuOMnk2dxUwEQvKQ8ycnRzr7D0hKYUb2q8G+16m8PR3BpS3d1KnJLVr7MHZXzR
            +sdsZXkS1eDpFaWDEDADYb4rDBodAvO1bm7ewS38RnMTi9atVs5SS83idnGlVbJk
            bFJXm+YlI4qdiz0LWcXbrDA=
            -----END PRIVATE KEY-----
            """)
    ),

    Cert(
        names=["localhost"],
        pubcert=strip("""
            -----BEGIN CERTIFICATE-----
            MIIDpjCCAo6gAwIBAgIJAJqkVxcTmCQHMA0GCSqGSIb3DQEBCwUAMGgxCzAJBgNV
            BAYTAlVTMQswCQYDVQQIDAJNQTEPMA0GA1UEBwwGQm9zdG9uMREwDwYDVQQKDAhE
            YXRhd2lyZTEUMBIGA1UECwwLRW5naW5lZXJpbmcxEjAQBgNVBAMMCWxvY2FsaG9z
            dDAeFw0xODEwMTAxMDI5MDJaFw0yODEwMDcxMDI5MDJaMGgxCzAJBgNVBAYTAlVT
            MQswCQYDVQQIDAJNQTEPMA0GA1UEBwwGQm9zdG9uMREwDwYDVQQKDAhEYXRhd2ly
            ZTEUMBIGA1UECwwLRW5naW5lZXJpbmcxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIw
            DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALq6mu/EK9PsT4bDuYh4hFOVvbnP
            zEz0jPrusuw1ONLBOcOhmncRNq8sQrLlAgsbp0nLVfCZRdt8RyNqAFyBeGoWKr/d
            kAQ2mPnr0PDyBO94Pz8Twrt0mdKDSWFjsq29NaRZOBju+KpezG+NgzK2M83FmJWT
            qXu270Oi9yjoeFCyO27pRGorKdBOSrb0wz3tWVPi84VLvqJEjkOBUf2X5QwonWZx
            2KqUBz9AReUT37pQRYBBLIGoJs8SN6r1x1+uu3Ku5qJCuBdeHyIlzJoethJv+zS2
            07JEsfJZIn1ciuxM73OneQNmKRJl/cDopKzk0JWQJtRWSgnKgxSXZDkf2L8CAwEA
            AaNTMFEwHQYDVR0OBBYEFBhC7CyTi4adHUBwL/NFeE6KvqHDMB8GA1UdIwQYMBaA
            FBhC7CyTi4adHUBwL/NFeE6KvqHDMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcN
            AQELBQADggEBAHRooLcpWDkR20HD4BywPSPdKWXcZsuSkWafrzHhaBy1bYrKHGZ5
            ahtQw/X0Bdg1kbvZYP2RO7FLXAJSSuuIOCGLUpKJdTq544DQ8MoWZYVJm77Qlqjm
            lsHkeeNTMjaNV7LwC3jPd0DXzW3legXThajfggm/0IesFG0UZ1D92G5DfsHKzJRj
            MHvrT3mVbFf9+HbaDN2Oh9V21QhVK1v3AvucWs8TX+0dvEgWmXpQrwDwjS1M8BDX
            WhZ5le6cW8Mb8gfdlxmIrJgA+nUVs1ODnBJKQw1F81WdsnmYwpyU+OlUj+8PkuMZ
            IN+RXPVvLIbws0fjbxQtsm0+ePiFsvwClPY=
            -----END CERTIFICATE-----
            """),
        privkey=strip("""
            -----BEGIN PRIVATE KEY-----
            MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC6uprvxCvT7E+G
            w7mIeIRTlb25z8xM9Iz67rLsNTjSwTnDoZp3ETavLEKy5QILG6dJy1XwmUXbfEcj
            agBcgXhqFiq/3ZAENpj569Dw8gTveD8/E8K7dJnSg0lhY7KtvTWkWTgY7viqXsxv
            jYMytjPNxZiVk6l7tu9Dovco6HhQsjtu6URqKynQTkq29MM97VlT4vOFS76iRI5D
            gVH9l+UMKJ1mcdiqlAc/QEXlE9+6UEWAQSyBqCbPEjeq9cdfrrtyruaiQrgXXh8i
            JcyaHrYSb/s0ttOyRLHyWSJ9XIrsTO9zp3kDZikSZf3A6KSs5NCVkCbUVkoJyoMU
            l2Q5H9i/AgMBAAECggEAIQlg3ijcBDub+mDok2+XId6tWZGdOMRPqRnQSCBGdGtB
            WA5gcM532VhAWLxRtztmRpUWGGJVzLZZM7fOZo91iXdwis+dalFqkVUae3amTuP8
            dKF/Y4EGsgsOUY+9DiYatoAef7LQBfyNuPLVkoRP+AkMrPIaG0xLWrEbf35ZwxTn
            wyM1waZPoZ1Z6EvhGBLM79Wbf6TV4YusI4M8EPuMFqiXp3eFfx/Kg4xmbvm7Rac7
            8BwgzgVic5yRnEWb8iYHyXkrk3S/EBaCD2T0R39ViU3R4V0f1KrWscDz0VcbUca+
            syWriXJ0pgGStCqV+GQc/Z6bc8kxUjSMlNQknuRQgQKBgQDzp3VZVas10758UOM+
            vTy1M/Ezk88qhFomdaQbHTem+ixjB6X7EOlFY2krpRL/mDCHJpGC3bRmPsEhuFIL
            DxRChTpKSVclK+ZiCOia5zKSUJqfpNqmyFsZBXI6td5ofZN6hZIU9IGdThiX20N7
            iQm5RveILvPup1fQ2djwazbH/wKBgQDEML7mLodjJ0MMxzfs71mE6fNPXA1V6dH+
            YBTn1KkqhrijjQZaMmvztFf/QwZHfwqJAEn4lvzngqCse3/PIY3/3DDqwZv4Mow/
            DgAy0KBjPaRF68XOPuwEnHSuR8rdX6S27Mt6pFHxVvb9QDRnIw8kx+HUkzixSHyR
            65lDJIDvQQKBgQDiA1wfWhBPBfOUbZPeBrvheiUrqthooAzf0BBB9oBBK58w3U9h
            7PX1n5lXGvDcltetBmHD+tP0ZBHSrZ+tEnfAnMTNU+q6WFaEaa8awYtvncVQgSMx
            wnh+ZUbonvuIAbRj2rL/LS9uM5ssgf+/AAc9Dk9ey+8KWcBjwzAxE8LlEQKBgB37
            1TEYq1hcB8NMLx/m9Kd7mdPnHaKjuZRG2usTdUcqj81vICloy1bTmR9J/wuuPs3x
            XVzAtqYrMKMrvLzLRAh2foNiU5P7JbP9T8p0WA7SvOhywChlNWz+/FYmYrqyg1nx
            lqeHtX5M7DKIPXoFwaq9YaY7Wc6+ZUtn1mSMj6gBAoGBAI80uObNGaFwPMV+Qhbd
            AzY+HSFB0dYfqG+sq0fEWHY3GMqf4XtiTjPAcZX7FgmOt9R+wNQP+GE66hWBi+0V
            eKWzkWIWy/lMVBIm3UkeJTBOsnu1UhhWnnVT8EyhDcQqrwOHiaiJ7lVRfdhEarC+
            JzZSG38uYQYrsIHNtUdX2JgO
            -----END PRIVATE KEY-----
            """)
    ),

    Cert(
        names=[
            "a.domain.com",
            "b.domain.com",
            "*.domain.com",
            #"localhost",  # don't clash with the other "localhost" cert
            "127.0.0.1",
            "0:0:0:0:0:0:0:1"
        ],
        # Note: This cert is signed by a cert not present in this file
        # (rather than being self-signed).
        pubcert=strip("""
            -----BEGIN CERTIFICATE-----
            MIIEgzCCAuugAwIBAgIRANoyJlZOx3sXGkasn+NQ1GwwDQYJKoZIhvcNAQELBQAw
            dzEeMBwGA1UEChMVbWtjZXJ0IGRldmVsb3BtZW50IENBMSYwJAYDVQQLDB1hbHZh
            cm9AY2FyYm9uIChBbHZhcm8gU2F1cmluKTEtMCsGA1UEAwwkbWtjZXJ0IGFsdmFy
            b0BjYXJib24gKEFsdmFybyBTYXVyaW4pMB4XDTE5MDYwMTAwMDAwMFoXDTMwMDgw
            NDExMTM0N1owUTEnMCUGA1UEChMebWtjZXJ0IGRldmVsb3BtZW50IGNlcnRpZmlj
            YXRlMSYwJAYDVQQLDB1hbHZhcm9AY2FyYm9uIChBbHZhcm8gU2F1cmluKTCCASIw
            DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKEwK+xnnMHnWg/SO+aOo8EN5npV
            tVKZP+4BtTpN9I2RWQJl3cTb5dTQ1BXDoK74E+D4hbowyUji24K95y3W5+5ClY/E
            Y0NmiKBW8wzrcW4rtG8uRyGMDKe/Q2ZdpSKWQA4WYOjdzqbg4/Tk1FXwCR7gGBjj
            g8bxWGUIbnLELr83QV4GuysA09Bq2eUMbuZlErnXEzrLhOgFDGGBE9igex+/vXEz
            4qQWiWspie6edGLDqixswuDyQnL3OTlOqB9lSReasCU2gEYnFnY+98BtPip3UVC0
            Vs79RHA7B4VJ5e21FgjM7MT3osQg8W25wxErFaNvInycw03XBaAc4Us2J6cCAwEA
            AaOBrzCBrDAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDAYD
            VR0TAQH/BAIwADAfBgNVHSMEGDAWgBQF7sHeqTQPVdjftAKZR3Jt8QFCcDBWBgNV
            HREETzBNggxhLmRvbWFpbi5jb22CDGIuZG9tYWluLmNvbYIMKi5kb21haW4uY29t
            gglsb2NhbGhvc3SHBH8AAAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQEL
            BQADggGBAIvTve/xeaixX1qUnfNFHRQ/FrFbL1xhjhpLy1pu6aqMX2ZC5FExvKrQ
            v7VKt5xFLDEHpBpnNEEU5ANgiKRUSP9Db6hzspzXkhDaOoLAkNB/7w5Waq9RsfNT
            TC2OY48cpRPlm0AUyUDtS3iCLPaILv9zYaeWvqdBuDobvIEOFNLpLmz1MLhgQgWb
            wvtnX42W2tYKz75zyO0jjqprpCnu7AyPUxbfmL+hPw9pTyCF670/9afyXMUhj3u8
            ypteAW+5TmfxqsAvN/4UblRfs5Cc/su/dXdPnJ1mBXJNXpZV4dp+5N6ua1Bq5cYv
            NlLCFNqSzQF8UgwSYAgPQUwbvDkmP2wNzSsQdthYX07BAZMTY2a8AX7CT0PpTXsv
            l/SZyrZhWfPgT+ONEnvhRxnqlekdoBTj9uaQA16/GxYXdq98gBJLyf9RazB+emBM
            oIW4hBBG+0Fqj1c5gNdi4g+t7PlGkGus/IN5HEBPFEqccjYSqzNknnkf4LVP7y5q
            jVdXKKFO4A==
            -----END CERTIFICATE-----
            """),
        privkey=strip("""
            -----BEGIN PRIVATE KEY-----
            MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQChMCvsZ5zB51oP
            0jvmjqPBDeZ6VbVSmT/uAbU6TfSNkVkCZd3E2+XU0NQVw6Cu+BPg+IW6MMlI4tuC
            vect1ufuQpWPxGNDZoigVvMM63FuK7RvLkchjAynv0NmXaUilkAOFmDo3c6m4OP0
            5NRV8Ake4BgY44PG8VhlCG5yxC6/N0FeBrsrANPQatnlDG7mZRK51xM6y4ToBQxh
            gRPYoHsfv71xM+KkFolrKYnunnRiw6osbMLg8kJy9zk5TqgfZUkXmrAlNoBGJxZ2
            PvfAbT4qd1FQtFbO/URwOweFSeXttRYIzOzE96LEIPFtucMRKxWjbyJ8nMNN1wWg
            HOFLNienAgMBAAECggEAZ0gfjN4jMpfUPHkASs4xHT2T5eVPRvrNXOsZPZ+/yIpO
            l1vASyh/zup0SvHL3vE0g52aymACScKa1t5p6BRhDmj5vmIfHIvxlZPBLxEZ4Hb+
            qZLknxlG7qF+RXRRoKTXrG8ob23YwVMunbeWWu5wWalLp3747BuvASXy53TPY1O3
            EBcnHQNon/seOQoH9WAR343UnWVX6KBMbgn6BK6Q9ivGvN0wqEIvu6hwLtDUxEn5
            wj3Gxld1hsDi3La0yjiBnnAAlcbtgRGZVcw67MByY+DYgrrHoXEIn+OIP86wBtPk
            8e1z0leuCN6EzTcxQv2qTnviu6Yd4sxTAkiwdvig6QKBgQDFcmTiMT8bVaVlPnqe
            fFumrb9Q5CQpLEfToj3WGBktDrvCAqE7h8HDHs+dRdf8s2+Bmwl1Jd3EziZZllo9
            6QvuHZjQbZ7k1Hft5XZPx2OpiqD6RjIwfzJT8xvdl/6Jz0P24i069oxKQ/hgUbd1
            5EZUyf2lSb2AsR8foa9Pg05NpQKBgQDQ/R79f0xPGOwB06ZbxzfGxiPdghPeqzw+
            LURf8SLxygB7xDBokQOwOYNGvGlFQYS7rnB7z8gQF8OtctJERpgAX9j5V0+aRgvt
            xVee/81P58ioWwBmoDXhJqt33Y9xngTHyupZzuK7gaxNhAkW/BGxblloU/anhJT+
            EHNLnXT2WwKBgC+jPfvk7djmfRVEUclTL7mzSel2YdMdP+cryceR4OEiIOLaR5RZ
            WMJ++JB1fXsWv9yBT3LYQ/1rz4zl3bf6NkqpEWmYSTHkoVrgdf8hmEYbkGNR9GIH
            Dll62kpIlb0iKL+0Kj2Dpq10YMS8cosbHGzwnyX1+KbIFT5IgEeq4oWRAoGBAJUl
            f99j4N62J4AqPxhStaCbOW9U7L9Fr0mkXp6l5c1u3yd03SNTErHKacCqp+owFv0m
            QcpqgBnUC+cWAa+OPd5OiPdxczLjeJHo+15SqoCzJwXXZBLZlXoocciqizuHjVvU
            makcN72fjosHhsErhaj92rrU6TumJ/qlXNMC/TzvAoGBAJzqB1JoprG9vTpMYMKI
            FdRM/cv2zRycqqatOr1sJT8JZSvwHTNl7MN+hX0M+ycIlK5C/FXPmeaFVkifWEDe
            XSp6sYPcCGon3TTjiz5BcH3PzS1FoshejpHwcdsrWugNfUuR+1yHCf2ONilFbdip
            LEjCS8jfhZE+bnKhzNwemsPu
            -----END PRIVATE KEY-----
            """)
       ),

    Cert(
        names=["acook"],
        pubcert=strip("""
            -----BEGIN CERTIFICATE-----
            MIIF7zCCA9egAwIBAgIUBj+Xwyen6cj/bUWUsvYl+kP1m6MwDQYJKoZIhvcNAQEL
            BQAwgYYxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJOWTENMAsGA1UEBwwEY2l0eTER
            MA8GA1UECgwIZGF0YXdpcmUxFDASBgNVBAsMC2VuZ2luZWVyaW5nMQ4wDAYDVQQD
            DAVhY29vazEiMCAGCSqGSIb3DQEJARYTZGV2bnVsbEBkYXRhd2lyZS5pbzAeFw0y
            MTAxMjgxOTQ2NTBaFw0zMTAxMjYxOTQ2NTBaMIGGMQswCQYDVQQGEwJVUzELMAkG
            A1UECAwCTlkxDTALBgNVBAcMBGNpdHkxETAPBgNVBAoMCGRhdGF3aXJlMRQwEgYD
            VQQLDAtlbmdpbmVlcmluZzEOMAwGA1UEAwwFYWNvb2sxIjAgBgkqhkiG9w0BCQEW
            E2Rldm51bGxAZGF0YXdpcmUuaW8wggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIK
            AoICAQDR8KsgN3WrcsLtJ9gzXF4oCeEk920LSdbET0elyak1XAyi/SKDRow4VhBp
            dbrF763j0e6e7d3qoRK48kCyZWoi3RRCfp3o4ZpmAi1sByrMY2SXEAQ2bg8Z2njn
            H6m7zIK9ZNK+ovF9FZk7V7lytMVLROyKTz9tAcTlsWz2bBmpRStEAramHmcjGJc7
            1hSalPY4UKfU7U2J6fGu0AVqxWyf0bJdyCjQcbhO/FfZc0ZDJpdyP1S1UcL77BGy
            JSSrrwS6Xb9oSMaUcl9EEiFGKuEle5VNDRoPPWF9B8Rnj6kn+7eQWA8u7FBcGKAK
            JH7orfLYrzCIDYSgnF9fJpw4AwZkgFiEz/sjj6tNZ0m8LE/uqxAwWHC7LmpaQJrd
            UiW38q0TtMNKOCaUQ3Tn7zNRyEYPXJEJTc00ZmkwIgELLL6aZnNuNdeYXWODVV6H
            KBxI9X0OvYb3eDV023gLXsDyrNgQmXjKEU0rgL6Iw1lH8UyImr2XezqEidvDgCfv
            JUQKRw/oU92I3SFaLmN2uC4hX8+zp7oBJhOAxtp0LHJbeGxsfTDBwxwlY84A4Yqw
            y0dnC/T7mof7ugW9GrYgobiFiI3iOEeVoFVrqurEMj2ek+af+N19ZWxOqiqjVwjG
            qqNP18CmERe0hMWlibeMQ598u5AXw39mKjwGSx33KUBNchglZwIDAQABo1MwUTAd
            BgNVHQ4EFgQUPpghohVuKCxZf3828hfRt01kLyYwHwYDVR0jBBgwFoAUPpghohVu
            KCxZf3828hfRt01kLyYwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOC
            AgEAPCiWBuEgkUCx6+t4pZ/3uCcQ155MYtPRTY+UZdZ8dfyZfYzmdyF7A9x6yDBY
            3yQj1Jyd1BV8zfmsN7O+3aRSOMPhadzmW5Gk3m6Rwcn2R6Cepg/cMw58ODHbePNd
            zsEndFOQ+YA6UJ5G8aTpyMOqcLjD0Uw7wfGV8ZoY55fNT7EzKQAaqNhZHHoI1pNX
            heOPOzUikWc+SYPsfHwSU0FJl4QO6HZaO4xJtOZIbTD/uWStG8ZUq1lE5LbrVbaH
            Lece/A084SI7dBkagHve6xLtLBd4bOxiXDyPgD3oIIJWcHEGZBzp99npX4jGr0Z/
            CbyfixtGVWRRIhnu1AKBZ3TL9FRCujIrYplzaFbEUiLeIO1sUT1AmS4eUjTdLF55
            +HcsxpMU6O2XEng/bw2rbdzQNUKCsgwCEcCfY5GTcrzX9dHQeVeRVXVLBwS08SUg
            73ZEklr62w8XoXsjik5AZ30cDCTe3FGJ+O6ziBIv1NHVM9+TUkfH4mTCxbvnRJsP
            4WmBh2ZqgNdCJfPjJzPB5wLqq6heVH1hwp7o2oNJcj/XirKt2KN1i8Hyl+5qpS5s
            ipaO1QrTqrs3G1dea3L47NK+oRlOYJ01CLVV40xvS7ZT9Tz5dXnmhL4rSB0vYGlA
            TYF3xLsQoUdmx8dQuiKUJ8NsaJkpj6QV9Bi5/LzIVvhB4Rw=
            -----END CERTIFICATE-----
            """),
        privkey=strip("""
            -----BEGIN PRIVATE KEY-----
            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
            NU56R894ytDCPGO6gcVCix8bhdSn/R0=
            -----END PRIVATE KEY-----
            """)
        ),
]

TLSCerts: Dict[str, Cert] = dict([k,v] for v in _TLSCerts for k in v.names)
