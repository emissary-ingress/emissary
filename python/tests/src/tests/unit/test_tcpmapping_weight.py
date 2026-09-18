from typing import Dict, List

from tests.utils import econf_compile

# A TCPMapping group's weights reach Envoy through tcp_proxy's weighted_clusters,
# where each weight is a share of the sum. The IR carries weights as cumulative
# thresholds because that is what HTTP routes need from runtime_fraction, so the
# listener has to turn them back into per-cluster shares on the way out.

TCP_WEIGHTED = """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: tcp-listener
  namespace: default
spec:
  port: 6789
  protocol: TCP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
metadata:
  name: tgt70
  namespace: default
spec:
  port: 6789
  service: tgt70:8080
  weight: 70
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
metadata:
  name: tgt30
  namespace: default
spec:
  port: 6789
  service: tgt30:8080
  weight: 30
"""


def _tcp_cluster_weights(econf) -> Dict[str, int]:
    """Map cluster name to the weight tcp_proxy hands Envoy."""
    weights: Dict[str, int] = {}
    for listener in econf["static_resources"]["listeners"]:
        for chain in listener.get("filter_chains", []):
            for f in chain.get("filters", []):
                if f.get("name") != "envoy.filters.network.tcp_proxy":
                    continue
                clusters = f["typed_config"].get("weighted_clusters", {}).get("clusters", [])
                for c in clusters:
                    weights[c["name"]] = c["weight"]
    return weights


def _weight_for(weights: Dict[str, int], substring: str) -> int:
    matches: List[int] = [w for name, w in weights.items() if substring in name]
    assert len(matches) == 1, f"expected one cluster matching {substring!r}, got {weights}"
    return matches[0]


def test_tcpmapping_weights_are_per_cluster_shares():
    weights = _tcp_cluster_weights(econf_compile(TCP_WEIGHTED))
    assert len(weights) == 2, f"expected two weighted clusters, got {weights}"

    w70 = _weight_for(weights, "tgt70")
    w30 = _weight_for(weights, "tgt30")

    # Envoy divides by the sum, so the pair has to read 70 and 30 rather than
    # any other pair sharing that ratio.
    assert (w70, w30) == (70, 30), f"got tgt70={w70} tgt30={w30}"
    assert w70 + w30 == 100


def test_tcpmapping_weights_sum_to_one_hundred():
    weights = _tcp_cluster_weights(econf_compile(TCP_WEIGHTED))
    assert sum(weights.values()) == 100, f"weights should sum to 100, got {weights}"
