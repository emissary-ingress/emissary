from tests.utils import (
    econf_compile,
    econf_foreach_listener,
    econf_foreach_listener_chain,
    EnvoyHCMInfo,
    EnvoyTCPInfo,
)

import pytest
import json

# This manifest set is from a test setup Flynn used when testing this by hand.
manifests = """
---
apiVersion: v1
kind: Secret
metadata:
  name: tls-cert
  namespace: default
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNzRENDQVpnQ0NRRFd2TnRjRzNpelZEQU5CZ2txaGtpRzl3MEJBUXNGQURBYU1SZ3dGZ1lEVlFRRERBOWgKYldKaGMzTmhaRzl5TFdObGNuUXdIaGNOTWpFd056QTRNakF5T0RNd1doY05Nakl3TnpBNE1qQXlPRE13V2pBYQpNUmd3RmdZRFZRUUREQTloYldKaGMzTmhaRzl5TFdObGNuUXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCCkR3QXdnZ0VLQW9JQkFRQ1pVbXhqT1lrTWlKRm0yZSttZDlMelNwd0oxSWlic1lUWHp5a1NiMExZYlNqcG5jMGoKV0dWMEppOXdlU3FSSFFPMHM4NUZreENzT2s1K2ZCWDFJOTYra1Z2V3NyeWgwcDlsdjI3ZUpHZFp1Q1ZsSmR3cApuYnBaWFF6R3JjWVVaeTA2WEVWOGxkaFdOSVhMazc1bmxsWmE5M2xjajRXRzNTRHpzT2MrdEtWaEtNaG9QSkVaClVGbXNxZ080dm8yZkJxYk0zNXhBT3lFSHhodXgvVlNLeVIxbHN0S0dsd25icGliZDc2UUZCdWYwbHN2bEJRTFAKV2xiRW8zZzI0NWxMNFhMWjg2UURoaTJseTdSNFN5em4yZ2E2TjZYQWNxMjFYTzNQUzhPaFp6d2J1cGpEMHRadApxL0JjY01kTElXbm9zVmlpc0FVdElLUHpCbjVkNFhBaGRtVnhBZ01CQUFFd0RRWUpLb1pJaHZjTkFRRUxCUUFECmdnRUJBSmFONUVxcTlqMi9IVnJWWk9wT3BuWVRSZlU0OU1pNDlvbkF1ZjlmQk9tR3dlMHBWSmpqTVlyQW9kZ1IKYWVyUHVWUlBEWGRzZXczejJkMjliRzBMVTJTdEpBMEY0Z05vWTY0bGVZUTN0RjFDUmxsczdKaWVWelN1RVVyUwpLZjZiaWJ0aUlLSU4waEdTV3R2YU04ZXhqb2Y3ZGUyeWFLNEVPeE1pQmJyZkFPNnJ6MXgzc1ovOENGTnp3OXNRClhCNWpZSWhNZWhsb2xhR0U5RGNydUdrbStFQ3ZCNjZkajFNcm5UamVJcWc4QnN4Wm5WYlZ4cDlUZTJRZ2hyTmkKckVySndjV1NSU3lUZzBEZXdUektYQUx2aW5iRTliZ3pNdFhNSEhkUmZQYUMvWmFCTUd1QXExeWJTOUV3M2MvWgo1dk00aFdOaHU5MS9DSmN5UVJHdlJRWXFiZTA9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBbVZKc1l6bUpESWlSWnRudnBuZlM4MHFjQ2RTSW03R0UxODhwRW05QzJHMG82WjNOCkkxaGxkQ1l2Y0hrcWtSMER0TFBPUlpNUXJEcE9mbndWOVNQZXZwRmIxcks4b2RLZlpiOXUzaVJuV2JnbFpTWGMKS1oyNldWME14cTNHRkdjdE9seEZmSlhZVmpTRnk1TytaNVpXV3ZkNVhJK0ZodDBnODdEblByU2xZU2pJYUR5UgpHVkJacktvRHVMNk5ud2Ftek4rY1FEc2hCOFlic2YxVWlza2RaYkxTaHBjSjI2WW0zZStrQlFibjlKYkw1UVVDCnoxcFd4S040TnVPWlMrRnkyZk9rQTRZdHBjdTBlRXNzNTlvR3VqZWx3SEt0dFZ6dHowdkRvV2M4RzdxWXc5TFcKYmF2d1hIREhTeUZwNkxGWW9yQUZMU0NqOHdaK1hlRndJWFpsY1FJREFRQUJBb0lCQUdlaVNOVUE3TnZsNjdKRAptVE5DUnZwZjhmekxCZE9IT0MzUFB3blEzclAvaE9uejJkY01SdmN0WUY5NzV3UFRRcy8vd1d0UnJyRmJiL2NhCjFKU3dQRDAvYjM0OXJqY0xjT2FMY05zQ2JFRStzVGdmVVNOb0U2K1hyNjBUaEpJQjg1WkJERTdiMGpEaXE1VWgKTmxBNlZBQ0V5aW1BY1ZicFhQNmJFcE5WODNzcDFBUEUrc2xpUWVrMHBWK2VJcFNuWGNkMWRNbjdhcHNuYmR3MgpBbDErRDBiTkJweUNSd1dCMm81dmh0ZzIrcndaQUNOdTFQdmJGc0g5bURGUit2elJBT1oycFRDMzRwWDBhcktECnUyMGFMTU1PT2NETWN0bWp2OHJrcVJVRWt6aTNuV0ljWVVVYXFKVG1Ub2RLZlRobXhsbGx5aDg5UVAzUG8raEwKYWk0b0VJa0NnWUVBeUcvQ2xaa3g4WHF3M0NrZXJrZlZoTm83OHIxWFVFRFE3U3dMUUllaHo1Ri9iRU9WRS8yUgpJeGFZQkx2alMwRkRzY0s3TkpPeU9kTVBpU1VNcHdMSC9kNnNrWjA2QWRTVllPbUJpNUFCMUJNZXk1b0cvSmtXClpzSm42Q3g5aEJUZTVzQnRCUWQ1K1phUXU4aDBhUFcwcFh3b1h5aW1JejNpZ3Vxdk1Dc3plNU1DZ1lFQXc5TWQKY2ZmK1FhbmxqMnVpcmcxbnljMFVkN1h6ekFZRXVMUERPdHltcm1SSU9BNGl4c3JzMzRmbVE4STc4cXpDMnhmNQpEdlJPNTNzMW9XSHNzbXdxcmgzQ0RVaDF2UEVEcHVqR3dLd2E4bE1yQ2piWDhtYk1ibVNyelBuczVWeVhXaEhFCkN3VHNPV3RleUZ3OVFkZTR1K011SEYzSHB0SHFvZlRFTGZJRXBXc0NnWUVBdVBPM3dFZGVlSTlZUjYrQjZodkwKQVE1SHB4UGtUOStmYWxyci94Mm95RnBnRkV6QWNYUFh5Mkw3MzlKb1NIYnV1a2NRYTlHbDhnbTZHamtmMWJTUgpTc2VBd2RVdFE2Y2dPQThBUlFJYlRkQmU2RTAzQ1R0U0dueGxXUzVFbSs2T1NLdGpiZkthTVI4b2FyN3IvRFpOCi9TMzJLdWpkZFVPVGttNXdQYWgvbHhVQ2dZQmh3N0dNcDZJQmlGKzZaYU5YUUF3VC9OWCtHaEg0UnZ6dWRaaS8KZDArait4N3ZGV2VaVmRCQ25PZUI1cVBsT1Frak51bTU1SkRNRW9BbzdPbXQva0Nrb3VpeGx2NW84TzdBMHEvLwpteXpzMUViRmw3SGlMQjVkOHRhdXhBdllTb3lwZy9zYkFUOHFQNGVYZ2kxM0JNc095cEhIeWE0V2cvQ2ZJTU1jCnFScFd0d0tCZ0hYRjVSWUo4alpLTnE2bHI5TVZhZFRXdkpRZ01VbHR0UWlaM3ZrMmc0S09Kc1NWdWtEbjFpZysKQ0NKZUU2VS9OS0N3ejJSMXBjSWVET3dwek9IYzJWNkt4Z0RYZUYyVWsvMjMydlB3aXRjVExhS2hsTTlDOGNLcwp6RGlJcVFkZDRLdFhDajc4S040TlhHZ1hJdVdXOHZERFY4Q05wQm45eUlUUXFST3NRSHRrCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-https-listener
  namespace: default
spec:
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-http-listener
  namespace: default
spec:
  port: 8080
  protocol: HTTP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-alternate-listener
  namespace: default
spec:
  port: 8888
  protocol: HTTPS
  securityModel: XFP
  statsPrefix: alternate
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-tls-listener
  namespace: default
spec:
  port: 9999
  protocol: TLS
  securityModel: SECURE
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: quote-backend
  namespace: default
spec:
  prefix: /backend/
  service: quote
  hostname: '*'
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
metadata:
  name: tls-backend
  namespace: default
spec:
  port: 9999
  service: quote
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
metadata:
  name: tcp-backend
  namespace: default
spec:
  port: 9998
  service: quote
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: quote-backend
  namespace: default
spec:
  prefix: /backend/
  service: quote
  hostname: '*'
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: wildcard-host
  namespace: default
spec:
  hostname: "*"
  acmeProvider:
    authority: none
  tlsSecret:
    name: tls-cert
"""


def check_filter(abbrev, typed_config, expected_stat_prefix):
    stat_prefix = typed_config.get("stat_prefix", "UNKNOWN")

    print(f"------ {abbrev}, stat_prefix {stat_prefix}")

    assert (
        stat_prefix == expected_stat_prefix
    ), f"wanted stat_prefix {expected_stat_prefix}, got {stat_prefix}"


def check_listener(listener):
    port = listener["address"]["socket_address"]["port_value"]

    got_count = len(listener["filter_chains"])
    got_plural = "" if (got_count == 1) else "s"

    print(f"---- Listener @ {port}, {got_count} chain{got_plural}:")

    # In this test, we can derive chain counts, protocols, and expected stat_prefix from
    # the port number. Ports 8443 and 8888 do HTTP and HTTPS, so they need two chains.
    # Ports < 9000 use HTTP, not TCP.

    check_info = {
        8080: ("HCM", 1, EnvoyHCMInfo, "ingress_http"),
        8443: ("HCM", 2, EnvoyHCMInfo, "ingress_https"),
        8888: ("HCM", 2, EnvoyHCMInfo, "alternate"),
        9998: ("TCP", 1, EnvoyTCPInfo, "ingress_tcp_9998"),
        9999: ("TCP", 1, EnvoyTCPInfo, "ingress_tls_9999"),
    }

    abbrev, chain_count, filter_info, expected_stat_prefix = check_info[port]

    # Here's wishing Python had really good anonymous functions...
    def checker(typed_config):
        return check_filter(abbrev, typed_config, expected_stat_prefix)

    econf_foreach_listener_chain(
        listener,
        checker,
        chain_count=chain_count,
        need_name=filter_info.name,
        need_type=filter_info.type,
    )


@pytest.mark.compilertest
def listener_stats_prefix():
    # ...compile an Envoy config...
    econf = econf_compile(manifests)

    # ...and make sure everything looks good.
    econf_foreach_listener(econf, check_listener, listener_count=5)
