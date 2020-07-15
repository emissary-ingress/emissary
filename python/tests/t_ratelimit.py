from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, ServiceType

class RateLimitV0Test(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: rate-limit-v0
spec:
  selector:
    app: rate-limit-v0
  ports:
  - port: 5000
    name: grpc
    targetPort: grpc
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rate-limit-v0
spec:
  selector:
    matchLabels:
      app: rate-limit-v0
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: rate-limit-v0
    spec:
      containers:
      - name: rate-limit
        image: {self.test_image[ratelimit]}
        ports:
        - name: grpc
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
""" + super().manifests()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        # ambassador_id: [ {self.with_tracing.ambassador_id}, {self.no_tracing.ambassador_id} ]
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
rate_limits:
- descriptor: A test case
  headers:
  - "x-ambassador-test-allow"
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_label_mapping
prefix: /labels/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - host_and_user:
      - custom-label:
          header: ":authority"
          omit_if_not_present: true
      - user:
          header: "x-user"
          omit_if_not_present: true

    - omg_header:
      - custom-label:
          header: "x-omg"
          default: "OMFG!"
""")

        # For self.with_tracing, we want to configure the TracingService.
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: RateLimitService
name: ratelimit-v0
service: rate-limit-v0:5000
timeout_ms: 500
""")

    def queries(self):
        # Speak through each Ambassador to the traced service...
        # yield Query(self.with_tracing.url("target/"))
        # yield Query(self.no_tracing.url("target/"))

        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true'
        })

        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'over my dead body'
        })

class RateLimitV1Test(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: rate-limit-v1
spec:
  selector:
    app: rate-limit-v1
  ports:
  - port: 5000
    name: grpc
    targetPort: grpc
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rate-limit-v1
spec:
  selector:
    matchLabels:
      app: rate-limit-v1
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: rate-limit-v1
    spec:
      containers:
      - name: rate-limit
        image: {self.test_image[ratelimit]}
        ports:
        - name: grpc
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
""" + super().manifests()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: RateLimitService
name: ratelimit-v1
service: rate-limit-v1:5000
timeout_ms: 500
""")

    def queries(self):
        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true'
        })

        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'over my dead body'
        })

class RateLimitV1WithTLSTest(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: rate-limit-tls
spec:
  selector:
    app: rate-limit-tls
  ports:
  - port: 5000
    name: grpc
    targetPort: grpc
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rate-limit-tls
spec:
  selector:
    matchLabels:
      app: rate-limit-tls
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: rate-limit-tls
    spec:
      containers:
      - name: rate-limit
        image: {self.test_image[ratelimit]}
        env:
        - name: "USE_TLS"
          value: "true"
        ports:
        - name: grpc
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
---
apiVersion: v1
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURDakNDQWZJQ0NRRGdYUjZ3V1Z6Wk9EQU5CZ2txaGtpRzl3MEJBUVVGQURCSE1SNHdIQVlEVlFRRERCVnkKWVhSbGJHbHRhWFF1WkdGMFlYZHBjbVV1YVc4eEpUQWpCZ2txaGtpRzl3MEJDUUVXRm1odmMzUnRZWE4wWlhKQQpaR0YwWVhkcGNtVXVhVzh3SGhjTk1Ua3dPVEU1TVRnek16QXlXaGNOTWpFd09ERTVNVGd6TXpBeVdqQkhNUjR3CkhBWURWUVFEREJWeVlYUmxiR2x0YVhRdVpHRjBZWGRwY21VdWFXOHhKVEFqQmdrcWhraUc5dzBCQ1FFV0ZtaHYKYzNSdFlYTjBaWEpBWkdGMFlYZHBjbVV1YVc4d2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFSwpBb0lCQVFDeWw5VkJtVjVCcFYxOHZzclNqUktyZGlEVnZZS0dxNlZVaGFTRTlZSWNRODhiSTFaeHlhUE9zUVlRCmMycmY4Q0RKdUp4M1hoUjUzMENwN3pQNmVSMjNwMkZBOSsxSWs5SHZhWUx0WDRtQTJOdjh4V1kxaEhua1BURnMKVFRwazBMREdYYjVZWnh2QkczNTNKL3NFKzFrSUUxenpKZldpQUpUMzZzMk5PRzRVQWhoVmFPS2p1K3grYXJwbQoyZnNhTldKTTFEMS9CUXVVN1Vid0p0QmIyZFo2WUtUNHE4M2doQWgybDhad1hQcFdJQmtpWGpNNXJ0WkQ3QmN4CkRxdFNtVE1ZejVjZWNwbmhiNEw4Z3hFVUJyWlRxQ3g4RVkvcDArY05mN2hScmFWbTd6Q3BaRDhvSUtLS0IvUDQKM1dHZHRHNlpHS0VFSmtXNjB5QWtxRmI3czNWdEFnTUJBQUV3RFFZSktvWklodmNOQVFFRkJRQURnZ0VCQUFJUwp2Znh1K3dQcUxicVRZV1NLTUt6S3JmNUxxWlpBZFpZZXNIR0oxNmFyVmt6eGhzcElGRGZpblZ1UmI1eERxWVVSCnFta0U1K0dCNDh5UGoxaUQvdXdPOEI0ckFnNW1Pekw1MEpUMDVQT1dyc0hjK1loLzAvUXpGNmlqNHlVNWZ6dloKRHpJdFBiNE5qWTMxUE44WkJMYTFGSHk5d2JGSzdyS2lWK2MxTzd0UHVjQTVUWnoxS0h5VUVYaWxHdmNoczB4OApzazRJR2xrVTNoSDFVWHBzTmw0NTI4VjR0L3dXOTdiYmhFeHYvYnBwR3RLbjRKei9hYkNScjA3Q3kzUytjUzc5CnlQNTZiQVZxa1R2TTVkUkhiNG9zOGYzNUJuQTdPci9YRTQ2K2VRMXNVY21IVG5OUWdFVkpWZGVwR3V4Sk5pNFMKejJ0WHhHSStTdTFlUnlQcVNGWT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2UUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktjd2dnU2pBZ0VBQW9JQkFRQ3lsOVZCbVY1QnBWMTgKdnNyU2pSS3JkaURWdllLR3E2VlVoYVNFOVlJY1E4OGJJMVp4eWFQT3NRWVFjMnJmOENESnVKeDNYaFI1MzBDcAo3elA2ZVIyM3AyRkE5KzFJazlIdmFZTHRYNG1BMk52OHhXWTFoSG5rUFRGc1RUcGswTERHWGI1WVp4dkJHMzUzCkovc0UrMWtJRTF6ekpmV2lBSlQzNnMyTk9HNFVBaGhWYU9LanUreCthcnBtMmZzYU5XSk0xRDEvQlF1VTdVYncKSnRCYjJkWjZZS1Q0cTgzZ2hBaDJsOFp3WFBwV0lCa2lYak01cnRaRDdCY3hEcXRTbVRNWXo1Y2VjcG5oYjRMOApneEVVQnJaVHFDeDhFWS9wMCtjTmY3aFJyYVZtN3pDcFpEOG9JS0tLQi9QNDNXR2R0RzZaR0tFRUprVzYweUFrCnFGYjdzM1Z0QWdNQkFBRUNnZ0VBS3hvNzdOWWdDb1hubHpqUTZKb0ZuSDRwRkl6bFdLMUtmS2k0ZVNKcm9YaTQKSGx1Yi9HQm0rWFo5K1RCeDVkUWxoYW5aa1hHU1RZdVZKcTVGaERrQTlCY2dnTGFWZlFPNEVpa0w0VkJDZG1kZwpTSlEzdzhqU1JrU0NqaG5oY3YxdS9LRVpWR3FtSnlnRWtLdUVpTUpFelk4bXlzUXBrVXpFcDBUekVSZENjZTljClNKT2Q2V2dGYUdzN3ZmTEpMajBmTk1SYytwcjFORnBmcTk0R2lyZFVyQUIvOHhlL2lpT3hnZXJoM1JPR0I4Z1MKZWpMeTZmZEhZK0Y1d2cyREtFMjVjeHcrdWFUeEo4MElucktWYTdhVjlqeGl1L0YvWW42a0FGT201cUFaTEdJUQo3bDVDY0dMWCtKdS96UGNpSEJRRWx2R2dSTGdZTCtxWHNPYStyZ3VvZ1FLQmdRRFpVU25xQUdvd2Zma3h1QTB2CnVxc042M3NaMHFVVUMraDB1aGZCanV1ZGxEZU8rM2U5WC9ncW1vTHFnYmpLbUNxcHZ0eFRKT2RZbWlkTG42QysKU3ZIMkw5V3RBNzVFMDB1bDdWb0VEOWs5S1c5aEU4M1ZoM0hJcnowQ3RtLzAvMEtJaVlXb2VkaWw2YktlTndUSApXci9MdlI5M2F2dHo2NUkwVWFBVjVyMjhqUUtCZ1FEU1loS2R1YzB1YUQ2VW45NW0yTmZJelRaNWxaNmZZdFFyCkZHbzByWWlTelZCRlZrNnR3akI5dmhtdmtjZjBNU21wQ1NPUUZGcjJ3Zk9UZGtBTnlxTWxwNXdHK3ZpRi9LeGMKcXNMQ1RROGhwNmFnazMzRE9MVEl2OXpKdkhnU2NsWW9pOTZqNk9Zc28rWUtITkxmTU1jVWw3dThqYmZ3YWx2dwp4Tno3WkdRVVlRS0JnSFdsZWRKamJSbFphVWxnUVV0QWZBL3FGbGR4Y01xOGM1aVZrZnpJT1lleVVLMklOMWQvCkYrTkFpSFVaeXdkcWYxWXJyQzBhd2w5MS9LWDFBZGxpeTBDaXZzT09UamdHUjJMSmJyemFNNW5uejVNM1hHd24KaWhMQnczNnZjMGFuMWNZQzVTZkM1dVZTOGM2ekxGUWNMYzdIVUx5ZVh3aHZWRlFjaUZTeStLNlZBb0dBUkFObQpwMDBBOHliS1RId2VoenRGRDJxZ1dOQXc5ckFaalUvTlFmaHo5Wm1nZ0xuMU42RlcwZC9hSi9OR0pFQ2Npa1FsCkZoZ3VqQ1dKbkR1WFc1NE4va2RnWHJWV0VPTHR5Z3QrYVJoR2N3ZmpDM2lES05DMVNVMFZrTFo0VHVaZHlqL2wKbXpIWTc4ZVF2K1l2bWU0SC9qVkxnUnFEdzVwdTNMaVlCRUdoUlNFQ2dZRUExdmNGWmsxOXVLS0JSWDhFRlgzbQphTlk2QWVhWTVJV2xVbDFJbEpHUHBFNnBnejh1aVRYNU9paG5KbGh4NDIzdjBIRU1QV0ZxNXQzcCtJdGRKZ24zCnlTQ0RNcXhyZUo4cytCS3BqeUcwVHFjN0pkbkpUcm1NN1FUMCtVUXExYmlBOENiU0FvMWVybHJpcEZ1blVMblEKSXp3SURyM1VzN0ROQkxNZmlOaDZuVWs9Ci0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0K
kind: Secret
metadata:
  name: ratelimit-tls-secret
type: kubernetes.io/tls
""" + super().manifests()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: ratelimit-tls-context
secret: ratelimit-tls-secret
alpn_protocols: h2
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: RateLimitService
name: ratelimit-tls
service: rate-limit-tls:5000
timeout_ms: 500
tls: ratelimit-tls-context
""")

    def queries(self):
        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true'
        })

        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'nope'
        })
