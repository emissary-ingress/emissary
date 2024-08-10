import json
import os
import platform
from base64 import b64encode
from typing import Dict, Optional

# from ambassador.VERSION import Version, Commit


def _get_images() -> Dict[str, str]:
    ret: Dict[str, str] = {}

    arch = platform.machine()

    if arch == "x86_64":
        arch = "amd64"

    # These should really be overridable by environment variables or a file
    # or something.
    return {
        "test-auth": f"ghcr.io/emissary-ingress/test-auth:latest-{arch}",
        "test-shadow": f"ghcr.io/emissary-ingress/test-shadow:latest-{arch}",
        "test-stats": f"ghcr.io/emissary-ingress/test-stats:latest-{arch}",
        "kat-client": f"ghcr.io/emissary-ingress/kat-client:latest-{arch}",
        "kat-server": f"ghcr.io/emissary-ingress/kat-server:latest-{arch}",
    }


_image_cache: Optional[Dict[str, str]] = None


def get_images() -> Dict[str, str]:
    global _image_cache
    if not _image_cache:
        _image_cache = _get_images()
    return _image_cache


_file_cache: Dict[str, str] = {}


def load(manifest_name: str) -> str:
    if manifest_name in _file_cache:
        return _file_cache[manifest_name]
    manifest_dir = __file__[: -len(".py")]
    manifest_file = os.path.join(manifest_dir, manifest_name + ".yaml")
    manifest_content = open(manifest_file, "r").read()
    _file_cache[manifest_name] = manifest_content
    return manifest_content


def format(st: str, /, **kwargs):
    # These replace statments ensure that these fields can be formatted properly
    st = st.replace("'{.status.replicas}'", "'{{.status.replicas}}'")
    st = st.replace("'{.spec.replicas}'", "'{{.spec.replicas}}'")
    serviceAccountExtra = ""
    if os.environ.get("DEV_USE_IMAGEPULLSECRET", False):
        serviceAccountExtra = """
imagePullSecrets:
- name: dev-image-pull-secret
"""
    return st.format(serviceAccountExtra=serviceAccountExtra, images=get_images(), **kwargs)


def namespace_manifest(namespace: str) -> str:
    ret = f"""
---
apiVersion: v1
kind: Namespace
metadata:
  name: {namespace}
"""

    if os.environ.get("DEV_USE_IMAGEPULLSECRET", None):
        dockercfg = {
            "auths": {
                os.path.dirname(os.environ["DEV_REGISTRY"]): {
                    "auth": b64encode(
                        (
                            os.environ["DOCKER_BUILD_USERNAME"]
                            + ":"
                            + os.environ["DOCKER_BUILD_PASSWORD"]
                        ).encode("utf-8")
                    ).decode("utf-8")
                }
            }
        }
        ret += f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: dev-image-pull-secret
  namespace: {namespace}
type: kubernetes.io/dockerconfigjson
data:
  ".dockerconfigjson": "{b64encode(json.dumps(dockercfg).encode("utf-8")).decode("utf-8")}"
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: default
  namespace: {namespace}
imagePullSecrets:
- name: dev-image-pull-secret
"""

    return ret


def crd_manifests() -> str:
    ret = ""

    ret += namespace_manifest("emissary-system")

    # Use .replace instead of .format because there are other '{word}' things in 'description' fields
    # that would cause KeyErrors when .format erroneously tries to evaluate them.
    ret += (
        load("crds")
        .replace("{images[emissary]}", get_images()["emissary"])
        .replace("{serviceAccountExtra}", format("{serviceAccountExtra}"))
    )

    return ret
