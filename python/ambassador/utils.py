#!/usr/bin/env python

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import binascii
import hashlib
import io
import logging
import os
import re
import socket
import tempfile
import threading
import time
from builtins import bytes
from distutils.util import strtobool
from typing import TYPE_CHECKING, Any, Dict, List, Optional, TextIO, Union
from urllib.parse import urlparse

import orjson
import requests
import yaml
from prometheus_client import Gauge

from .VERSION import Version

if TYPE_CHECKING:
    from .config.acresource import ACResource  # pragma: no cover
    from .ir import IRResource  # pragma: no cover
    from .ir.irtlscontext import IRTLSContext  # pragma: no cover

logger = logging.getLogger("utils")
logger.setLevel(logging.INFO)

# XXX What a hack. There doesn't seem to be a way to convince mypy that SafeLoader
# and CSafeLoader share a base class, even though they do. Sigh.

yaml_loader: Any = yaml.SafeLoader
yaml_dumper: Any = yaml.SafeDumper

try:
    yaml_loader = yaml.CSafeLoader
except AttributeError:
    pass

try:
    yaml_dumper = yaml.CSafeDumper
except AttributeError:
    pass

yaml_logged_loader = False
yaml_logged_dumper = False


def parse_yaml(serialization: str) -> Any:
    global yaml_logged_loader

    if not yaml_logged_loader:
        yaml_logged_loader = True

        # logger.info("YAML: using %s parser" % ("Python" if (yaml_loader == yaml.SafeLoader) else "C"))

    return list(yaml.load_all(serialization, Loader=yaml_loader))


def dump_yaml(obj: Any, **kwargs) -> str:
    global yaml_logged_dumper

    if not yaml_logged_dumper:
        yaml_logged_dumper = True

        # logger.info("YAML: using %s dumper" % ("Python" if (yaml_dumper == yaml.SafeDumper) else "C"))

    return yaml.dump(obj, Dumper=yaml_dumper, **kwargs)


def parse_json(serialization: str) -> Any:
    return orjson.loads(serialization)


def dump_json(obj: Any, pretty=False) -> str:
    # There's a nicer way to do this in python, I'm sure.
    if pretty:
        return bytes.decode(
            orjson.dumps(
                obj, option=orjson.OPT_NON_STR_KEYS | orjson.OPT_SORT_KEYS | orjson.OPT_INDENT_2
            )
        )
    else:
        return bytes.decode(orjson.dumps(obj, option=orjson.OPT_NON_STR_KEYS))


def _load_url_contents(
    logger: logging.Logger, url: str, stream1: TextIO, stream2: Optional[TextIO] = None
) -> bool:
    saved = False

    try:
        with requests.get(url) as r:
            if r.status_code == 200:

                # All's well, pull the config down.
                encoded = b""

                try:
                    for chunk in r.iter_content(chunk_size=65536):
                        # We do this by hand instead of with 'decode_unicode=True'
                        # above because setting decode_unicode only decodes text,
                        # and WATT hands us application/json...
                        encoded += chunk

                    decoded = encoded.decode("utf-8")
                    stream1.write(decoded)

                    if stream2:
                        stream2.write(decoded)

                    saved = True
                except IOError as e:
                    logger.error("couldn't save Kubernetes resources: %s" % e)
                except Exception as e:
                    logger.error("couldn't read Kubernetes resources: %s" % e)
    except requests.exceptions.RequestException as e:
        logger.error("could not load new snapshot: %s" % e)

    return saved


def save_url_contents(
    logger: logging.Logger, url: str, path: str, stream2: Optional[TextIO] = None
) -> bool:
    with open(path, "w", encoding="utf-8") as stream:
        return _load_url_contents(logger, url, stream, stream2=stream2)


def load_url_contents(
    logger: logging.Logger, url: str, stream2: Optional[TextIO] = None
) -> Optional[str]:
    stream = io.StringIO()

    saved = _load_url_contents(logger, url, stream, stream2=stream2)

    if saved:
        return stream.getvalue()
    else:
        return None


def parse_bool(s: Optional[Union[str, bool]]) -> bool:
    """
    Parse a boolean value from a string. T, True, Y, y, 1 return True;
    other things return False.
    """

    # If `s` is already a bool, return its value.
    #
    # This allows a caller to not know or care whether their value is already
    # a boolean, or if it is a string that needs to be parsed below.
    if isinstance(s, bool):
        return s

    # If we didn't get anything at all, return False.
    if not s:
        return False

    # OK, we got _something_, so try strtobool.
    try:
        return bool(strtobool(s))  # the linter does not like a Literal[0, 1] being returned here
    except ValueError:
        return False


class SystemInfo:
    MyHostName = os.environ.get("HOSTNAME", None)

    if not MyHostName:
        MyHostName = "localhost"

        try:
            MyHostName = socket.gethostname()
        except:
            pass


class RichStatus:
    def __init__(self, ok, **kwargs):
        self.ok = ok
        self.info = kwargs
        self.info["hostname"] = SystemInfo.MyHostName
        self.info["version"] = Version

    # Remember that __getattr__ is called only as a last resort if the key
    # isn't a normal attr.
    def __getattr__(self, key):
        return self.info.get(key)

    def __bool__(self):
        return self.ok

    def __nonzero__(self):
        return bool(self)

    def __contains__(self, key):
        return key in self.info

    def __str__(self):
        attrs = ["%s=%s" % (key, repr(self.info[key])) for key in sorted(self.info.keys())]
        astr = " ".join(attrs)

        if astr:
            astr = " " + astr

        return "<RichStatus %s%s>" % ("OK" if self else "BAD", astr)

    def as_dict(self):
        d = {"ok": self.ok}

        for key in self.info.keys():
            d[key] = self.info[key]

        return d

    @classmethod
    def fromError(self, error, **kwargs):
        kwargs["error"] = error
        return RichStatus(False, **kwargs)

    @classmethod
    def OK(self, **kwargs):
        return RichStatus(True, **kwargs)


class Timer:
    """
    Timer is a simple class to measure time. When a Timer is created,
    it is given a name, and is stopped.

    t = Timer("test timer")

    The simplest way to use the Timer is as a context manager:

    with t:
        something_to_be_timed()

    You can also use the start method to start the timer:

    t.start()

    ...and the .stop method to stop the timer and update the timer's
    records.

    t.stop()

    Timers record the accumulated time and the number of start/stop
    cycles (in .accumulated and .cycles respectively). They can also
    return the average time per cycle (.average) and minimum and
    maximum times per cycle (.minimum and .maximum).
    """

    name: str
    _cycles: int
    _starttime: float
    _accumulated: float
    _minimum: float
    _maximum: float
    _running: bool
    _faketime: float
    _gauge: Optional[Gauge] = None

    def __init__(self, name: str, prom_metrics_registry: Optional[Any] = None) -> None:
        """
        Create a Timer, given a name. The Timer is initially stopped.
        """

        self.name = name

        if prom_metrics_registry:
            metric_prefix = re.sub(r"\s+", "_", name).lower()
            self._gauge = Gauge(
                f"{metric_prefix}_time_seconds",
                f"Elapsed time on {name} operations",
                namespace="ambassador",
                registry=prom_metrics_registry,
            )

        self.reset()

    def reset(self) -> None:
        self._cycles = 0
        self._starttime = 0
        self._accumulated = 0.0
        self._minimum = 999999999999
        self._maximum = -999999999999
        self._running = False
        self._faketime = 0.0

    def __enter__(self):
        self.start()
        return self

    def __exit__(self, type, value, traceback):
        self.stop()

    def __bool__(self) -> bool:
        """
        Timers test True in a boolean context if they have timed at least one
        cycle.
        """
        return self._cycles > 0

    def start(self, when: Optional[float] = None) -> None:
        """
        Start a Timer running.

        :param when: Optional start time. If not supplied,
        the current time is used.
        """

        # If we're already running, this method silently discards the
        # currently-running cycle. Why? Because otherwise, it's a little
        # too easy to forget to stop a Timer, cause an Exception, and
        # crash the world.
        #
        # Not that I ever got bitten by this. Of course. [ :P ]

        self._starttime = when or time.perf_counter()
        self._running = True

    def stop(self, when: Optional[float] = None) -> float:
        """
        Stop a Timer, increment the cycle count, and update the
        accumulated time with the amount of time since the Timer
        was started.

        :param when: Optional stop time. If not supplied,
        the current time is used.
        :return: The amount of time the Timer has accumulated
        """

        # If we're already stopped, just return the same thing as the
        # previous call to stop. See comments in start() for why this
        # isn't an Exception...

        if self._running:
            if not when:
                when = time.perf_counter()

            self._running = False
            self._cycles += 1

            this_cycle = (when - self._starttime) + self._faketime
            if self._gauge:
                self._gauge.set(this_cycle)

            self._faketime = 0

            self._accumulated += this_cycle

            if this_cycle < self._minimum:
                self._minimum = this_cycle

            if this_cycle > self._maximum:
                self._maximum = this_cycle

        return self._accumulated

    def faketime(self, faketime: float) -> None:
        """
        Add fake time to a Timer. This is intended solely for
        testing.
        """

        if not self._running:
            raise Exception(f"Timer {self.name}.faketime: not running")

        self._faketime = faketime

    @property
    def cycles(self):
        """
        The number of timing cycles this Timer has recorded.
        """
        return self._cycles

    @property
    def starttime(self):
        """
        The time this Timer was last started, or 0 if it has
        never been started.
        """
        return self._starttime

    @property
    def accumulated(self):
        """
        The amount of time this Timer has accumulated.
        """
        return self._accumulated

    @property
    def minimum(self):
        """
        The minimum single-cycle time this Timer has recorded.
        """
        return self._minimum

    @property
    def maximum(self):
        """
        The maximum single-cycle time this Timer has recorded.
        """
        return self._maximum

    @property
    def average(self):
        """
        The average cycle time for this Timer.
        """
        if self._cycles > 0:
            return self._accumulated / self._cycles

        raise Exception(f"Timer {self.name}.average: no cycles to average")

    @property
    def running(self):
        """
        Whether or not this Timer is running.
        """
        return self._running

    def __str__(self) -> str:
        s = "Timer %s: " % self.name

        if self._running:
            s += "running, "

        s += "%.6f sec" % self._accumulated

        return s

    def summary(self) -> str:
        """
        Return a summary of this Timer.
        """

        return "TIMER %s: %d, %.3f/%.3f/%.3f" % (
            self.name,
            self.cycles,
            self.minimum,
            self.average,
            self.maximum,
        )


class DelayTrigger(threading.Thread):
    def __init__(self, onfired, timeout=5, name=None):
        super().__init__()

        if name:
            self.name = name

        self.trigger_source, self.trigger_dest = socket.socketpair()

        self.onfired = onfired
        self.timeout = timeout

        self.setDaemon(True)
        self.start()

    def trigger(self):
        self.trigger_source.sendall(b"X")

    def run(self):
        while True:
            self.trigger_dest.settimeout(None)
            x = self.trigger_dest.recv(128)

            self.trigger_dest.settimeout(self.timeout)

            while True:
                try:
                    x = self.trigger_dest.recv(128)
                except socket.timeout:
                    self.onfired()
                    break


class PeriodicTrigger(threading.Thread):
    def __init__(self, onfired, period=5, name=None):
        super().__init__()

        if name:
            self.name = name

        self.onfired = onfired
        self.period = period

        self.daemon = True
        self.start()

    def trigger(self):
        pass

    def run(self):
        while True:
            time.sleep(self.period)
            self.onfired()


class SecretInfo:
    """
    SecretInfo encapsulates a secret, including its name, its namespace, and all of its
    ciphertext elements. Pretty much everything in Ambassador that worries about secrets
    uses a SecretInfo.
    """

    def __init__(
        self,
        name: str,
        namespace: str,
        secret_type: str,
        tls_crt: Optional[str] = None,
        tls_key: Optional[str] = None,
        user_key: Optional[str] = None,
        root_crt: Optional[str] = None,
        decode_b64=True,
    ) -> None:
        self.name = name
        self.namespace = namespace
        self.secret_type = secret_type

        if decode_b64:
            if self.is_decodable(tls_crt):
                assert tls_crt
                tls_crt = self.decode(tls_crt)

            if self.is_decodable(tls_key):
                assert tls_key
                tls_key = self.decode(tls_key)

            if self.is_decodable(user_key):
                assert user_key
                user_key = self.decode(user_key)

            if self.is_decodable(root_crt):
                assert root_crt
                root_crt = self.decode(root_crt)

        self.tls_crt = tls_crt
        self.tls_key = tls_key
        self.user_key = user_key
        self.root_crt = root_crt

    @staticmethod
    def is_decodable(b64_pem: Optional[str]) -> bool:
        if not b64_pem:
            return False

        return not (b64_pem.startswith("-----BEGIN") or b64_pem.startswith("-sanitized-"))

    @staticmethod
    def decode(b64_pem: str) -> Optional[str]:
        """
        Do base64 decoding of a cryptographic element.

        :param b64_pem: Base64-encoded PEM element
        :return: Decoded PEM element
        """
        utf8_pem = None
        pem = None

        try:
            utf8_pem = binascii.a2b_base64(b64_pem)
        except binascii.Error:
            return None

        try:
            pem = utf8_pem.decode("utf-8")
        except UnicodeDecodeError:
            return None

        return pem

    @staticmethod
    def fingerprint(pem: Optional[str]) -> str:
        """
        Generate and return a cryptographic fingerprint of a PEM element.

        The fingerprint is the uppercase hex SHA-1 signature of the element's UTF-8
        representation.

        :param pem: PEM element
        :return: fingerprint string
        """
        if not pem:
            return "<none>"

        h = hashlib.new("sha1")
        h.update(pem.encode("utf-8"))
        hd = h.hexdigest()[0:16].upper()

        keytype = "PEM" if pem.startswith("-----BEGIN") else "RAW"

        return f"{keytype}: {hd}"

    def to_dict(self) -> Dict[str, Any]:
        """
        Return the dictionary representation of this SecretInfo.

        :return: dict
        """
        return {
            "name": self.name,
            "namespace": self.namespace,
            "secret_type": self.secret_type,
            "tls_crt": self.fingerprint(self.tls_crt),
            "tls_key": self.fingerprint(self.tls_key),
            "user_key": self.fingerprint(self.user_key),
            "root_crt": self.fingerprint(self.root_crt),
        }

    @classmethod
    def from_aconf_secret(cls, aconf_object: "ACResource") -> "SecretInfo":
        """
        Convert an ACResource containing a secret into a SecretInfo. This is used by the IR.save_secret_info()
        to convert saved secrets into SecretInfos.

        :param aconf_object: a ACResource containing a secret
        :return: SecretInfo
        """

        tls_crt = aconf_object.get("tls_crt", None)
        if not tls_crt:
            tls_crt = aconf_object.get("cert-chain_pem")

        tls_key = aconf_object.get("tls_key", None)
        if not tls_key:
            tls_key = aconf_object.get("key_pem")

        user_key = aconf_object.get("user_key", None)
        if not user_key:
            # We didn't have a 'user_key', do we have a `crl_pem` instead?
            user_key = aconf_object.get("crl_pem", None)

        return SecretInfo(
            aconf_object.name,
            aconf_object.namespace,
            aconf_object.secret_type,
            tls_crt,
            tls_key,
            user_key,
            aconf_object.get("root-cert_pem", None),
        )

    @classmethod
    def from_dict(
        cls,
        resource: "IRResource",
        secret_name: str,
        namespace: str,
        source: str,
        cert_data: Optional[Dict[str, Any]],
        secret_type="kubernetes.io/tls",
    ) -> Optional["SecretInfo"]:
        """
        Given a secret's name and namespace, and a dictionary of configuration elements, return
        a SecretInfo for the secret.

        The "source" parameter needs some explanation. When working with secrets in most environments
        where Ambassador runs, secrets will be loaded from some external system (e.g. Kubernetes),
        and serialized to disk, and the disk serialization is the thing we can actually read the
        dictionary of secret data from. The "source" parameter is the thing we read to get the actual
        dictionary -- in our example above, "source" would be the pathname of the serialization on
        disk, rather than the Kubernetes resource name.

        :param resource: owning IRResource
        :param secret_name: name of secret
        :param namespace: namespace of secret
        :param source: source of data
        :param cert_data: dictionary of secret info (public and private key, etc.)
        :param secret_type: Kubernetes-style secret type
        :return:
        """
        tls_crt = None
        tls_key = None
        user_key = None

        if not cert_data:
            resource.ir.logger.error(
                f"{resource.kind} {resource.name}: found no certificate in {source}?"
            )
            return None

        if secret_type == "kubernetes.io/tls":
            # OK, we have something to work with. Hopefully.
            tls_crt = cert_data.get("tls.crt", None)

            if not tls_crt:
                # Having no public half is definitely an error. Having no private half given a public half
                # might be OK, though -- that's up to our caller to decide.
                resource.ir.logger.error(
                    f"{resource.kind} {resource.name}: found data but no certificate in {source}?"
                )
                return None

            tls_key = cert_data.get("tls.key", None)
        elif secret_type == "Opaque":
            user_key = cert_data.get("user.key", None)

            if not user_key:
                # The opaque keys we support must have user.key, but will likely have nothing else.
                resource.ir.logger.error(
                    f"{resource.kind} {resource.name}: found data but no user.key in {source}?"
                )
                return None

            cert = None
        elif secret_type == "istio.io/key-and-cert":
            resource.ir.logger.error(
                f"{resource.kind} {resource.name}: found data but handler for istio key not finished yet"
            )

        return SecretInfo(
            secret_name, namespace, secret_type, tls_crt=tls_crt, tls_key=tls_key, user_key=user_key
        )


class SavedSecret:
    """
    SavedSecret collects information about a secret saved locally, including its name, namespace,
    paths to its elements on disk, and a copy of its cert data dictionary.

    It's legal for a SavedSecret to have paths, etc, of None, representing a secret for which we
    found no information. SavedSecret will evaluate True as a boolean if - and only if - it has
    the minimal information needed to represent a real secret.
    """

    def __init__(
        self,
        secret_name: str,
        namespace: str,
        cert_path: Optional[str],
        key_path: Optional[str],
        user_path: Optional[str],
        root_cert_path: Optional[str],
        cert_data: Optional[Dict],
    ) -> None:
        self.secret_name = secret_name
        self.namespace = namespace
        self.cert_path = cert_path
        self.key_path = key_path
        self.user_path = user_path
        self.root_cert_path = root_cert_path
        self.cert_data = cert_data

    @property
    def name(self) -> str:
        return "secret %s in namespace %s" % (self.secret_name, self.namespace)

    def __bool__(self) -> bool:
        return bool((bool(self.cert_path) or bool(self.user_path)) and (self.cert_data is not None))

    def __str__(self) -> str:
        return (
            "<SavedSecret %s.%s -- cert_path %s, key_path %s, user_path %s, root_cert_path %s, cert_data %s>"
            % (
                self.secret_name,
                self.namespace,
                self.cert_path,
                self.key_path,
                self.user_path,
                self.root_cert_path,
                "present" if self.cert_data else "absent",
            )
        )


class SecretHandler:
    """
    SecretHandler: manage secrets for Ambassador. There are two fundamental rules at work here:

    - The Python part of Ambassador doesn’t get to talk directly to Kubernetes. Part of this is
      because the Python K8s client isn’t maintained all that well. Part is because, for testing,
      we need to be able to separate secrets from Kubernetes.
    - Most of the handling of secrets (e.g. saving the actual bits of the certs) need to be
      common code paths, so that testing them outside of Kube gives results that are valid inside
      Kube.

    To work within these rules, you’re required to pass a SecretHandler when instantiating an IR.
    The SecretHandler mediates access to secrets outside Ambassador, and to the cache of secrets
    we've already loaded.

    SecretHandler subclasses will typically only need to override load_secret: the other methods
    of SecretHandler generally won't need to change, and arguably should not be considered part
    of the public interface of SecretHandler.

    Finally, note that SecretHandler itself is deliberately written to work correctly with
    secrets as they're handed over from watt, which means that it can be instantiated directly
    and handed to the IR when we're running "for real" in Kubernetes with watt. Other things
    (like mockery and the watch_hook) use subclasses to manage specific needs that they have.
    """

    logger: logging.Logger
    source_root: str
    cache_dir: str

    def __init__(
        self, logger: logging.Logger, source_root: str, cache_dir: str, version: str
    ) -> None:
        self.logger = logger
        self.source_root = source_root
        self.cache_dir = cache_dir
        self.version = version

    def load_secret(
        self, resource: "IRResource", secret_name: str, namespace: str
    ) -> Optional[SecretInfo]:
        """
        load_secret: given a secret’s name and namespace, pull it from wherever it really lives,
        write it to disk, and return a SecretInfo telling the rest of Ambassador where it got written.

        This is the fallback load_secret implementation, which doesn't do anything: it is written
        assuming that ir.save_secret_info has already filled ir.saved_secrets with any secrets handed in
        from watt, so that load_secrets will never be called for those secrets. Therefore, if load_secrets
        gets called at all, it's for a secret that wasn't found, and it should just return None.

        :param resource: referencing resource (so that we can correctly default the namespace)
        :param secret_name: name of the secret
        :param namespace: namespace, if any specific namespace was given
        :return: Optional[SecretInfo]
        """

        self.logger.debug(
            "SecretHandler (%s %s): load secret %s in namespace %s"
            % (resource.kind, resource.name, secret_name, namespace)
        )

        return None

    def still_needed(self, resource: "IRResource", secret_name: str, namespace: str) -> None:
        """
        still_needed: remember that a given secret is still needed, so that we can tell watt to
        keep paying attention to it.

        The default implementation doesn't do much of anything, because it assumes that we're
        not running in the watch_hook, so watt has already been told everything it needs to be
        told. This should be OK for everything that's not the watch_hook.

        :param resource: referencing resource
        :param secret_name: name of the secret
        :param namespace: namespace of the secret
        :return: None
        """

        self.logger.debug(
            "SecretHandler (%s %s): secret %s in namespace %s is still needed"
            % (resource.kind, resource.name, secret_name, namespace)
        )

    def cache_secret(self, resource: "IRResource", secret_info: SecretInfo) -> SavedSecret:
        """
        cache_secret: stash the SecretInfo from load_secret into Ambassador’s internal cache,
        so that we don’t have to call load_secret again if we need it again.

        The default implementation should be usable by everything that's not the watch_hook.

        :param resource: referencing resource
        :param secret_info: SecretInfo returned from load_secret
        :return: SavedSecret
        """

        name = secret_info.name
        namespace = secret_info.namespace
        tls_crt = secret_info.tls_crt
        tls_key = secret_info.tls_key
        user_key = secret_info.user_key
        root_crt = secret_info.root_crt

        return self.cache_internal(name, namespace, tls_crt, tls_key, user_key, root_crt)

    def cache_internal(
        self,
        name: str,
        namespace: str,
        tls_crt: Optional[str],
        tls_key: Optional[str],
        user_key: Optional[str],
        root_crt: Optional[str],
    ) -> SavedSecret:
        h = hashlib.new("sha1")

        tls_crt_path = None
        tls_key_path = None
        user_key_path = None
        root_crt_path = None
        cert_data = None

        # Don't save if it has neither a tls_crt or a user_key or the root_crt
        if tls_crt or user_key or root_crt:
            for el in [tls_crt, tls_key, user_key]:
                if el:
                    h.update(el.encode("utf-8"))

            hd = h.hexdigest().upper()

            secret_dir = os.path.join(self.cache_dir, namespace, "secrets-decoded", name)

            try:
                os.makedirs(secret_dir)
            except FileExistsError:
                pass

            if tls_crt:
                tls_crt_path = os.path.join(secret_dir, f"{hd}.crt")
                open(tls_crt_path, "w").write(tls_crt)

            if tls_key:
                tls_key_path = os.path.join(secret_dir, f"{hd}.key")
                open(tls_key_path, "w").write(tls_key)

            if user_key:
                user_key_path = os.path.join(secret_dir, f"{hd}.user")
                open(user_key_path, "w").write(user_key)

            if root_crt:
                root_crt_path = os.path.join(secret_dir, f"{hd}.root.crt")
                open(root_crt_path, "w").write(root_crt)

            cert_data = {
                "tls_crt": tls_crt,
                "tls_key": tls_key,
                "user_key": user_key,
                "root_crt": root_crt,
            }

            self.logger.debug(
                f"saved secret {name}.{namespace}: {tls_crt_path}, {tls_key_path}, {root_crt_path}"
            )

        return SavedSecret(
            name, namespace, tls_crt_path, tls_key_path, user_key_path, root_crt_path, cert_data
        )

    def secret_info_from_k8s(
        self,
        resource: "IRResource",
        secret_name: str,
        namespace: str,
        source: str,
        serialization: Optional[str],
    ) -> Optional[SecretInfo]:
        """
        secret_info_from_k8s is NO LONGER USED.
        """

        objects: Optional[List[Any]] = None

        self.logger.debug(f"getting secret info for secret {secret_name} from k8s")

        # If serialization is None or empty, we'll just return None.

        if serialization:
            try:
                objects = parse_yaml(serialization)
            except yaml.error.YAMLError as e:
                self.logger.error(f"{resource.kind} {resource.name}: could not parse {source}: {e}")

        if not objects:
            # Nothing in the serialization, we're done.
            return None

        secret_type = None
        cert_data = None
        ocount = 0
        errors = 0

        for obj in objects:
            ocount += 1
            kind = obj.get("kind", None)

            if kind != "Secret":
                self.logger.error(
                    "%s %s: found K8s %s at %s.%d?"
                    % (resource.kind, resource.name, kind, source, ocount)
                )
                errors += 1
                continue

            metadata = obj.get("metadata", None)

            if not metadata:
                self.logger.error(
                    "%s %s: found K8s Secret with no metadata at %s.%d?"
                    % (resource.kind, resource.name, source, ocount)
                )
                errors += 1
                continue

            secret_type = metadata.get("type", "kubernetes.io/tls")

            if "data" in obj:
                if cert_data:
                    self.logger.error(
                        "%s %s: found multiple Secrets in %s?"
                        % (resource.kind, resource.name, source)
                    )
                    errors += 1
                    continue

                cert_data = obj["data"]

        if errors:
            # Bzzt.
            return None

        return SecretInfo.from_dict(
            resource, secret_name, namespace, source, cert_data=cert_data, secret_type=secret_type
        )


class NullSecretHandler(SecretHandler):
    def __init__(
        self,
        logger: logging.Logger,
        source_root: Optional[str],
        cache_dir: Optional[str],
        version: str,
    ) -> None:
        """
        Returns a valid SecretInfo (with fake keys) for any requested secret. Also, you can pass
        None for source_root and cache_dir to use random temporary directories for them.
        """

        if not source_root:
            self.tempdir_source = tempfile.TemporaryDirectory(
                prefix="null-secret-", suffix="-source"
            )
            source_root = self.tempdir_source.name

        if not cache_dir:
            self.tempdir_cache = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-cache")
            cache_dir = self.tempdir_cache.name

        logger.info(f"NullSecretHandler using source_root {source_root}, cache_dir {cache_dir}")

        super().__init__(logger, source_root, cache_dir, version)

    def load_secret(
        self, resource: "IRResource", secret_name: str, namespace: str
    ) -> Optional[SecretInfo]:
        # In the Real World, the secret loader should, y'know, load secrets..
        # Here we're just gonna fake it.
        self.logger.debug(
            "NullSecretHandler (%s %s): load secret %s in namespace %s"
            % (resource.kind, resource.name, secret_name, namespace)
        )

        return SecretInfo(
            secret_name,
            namespace,
            "fake-secret",
            "fake-tls-crt",
            "fake-tls-key",
            "fake-user-key",
            decode_b64=False,
        )


class EmptySecretHandler(SecretHandler):
    def __init__(
        self,
        logger: logging.Logger,
        source_root: Optional[str],
        cache_dir: Optional[str],
        version: str,
    ) -> None:
        """
        Returns a None to simulate no provided secrets
        """
        super().__init__(logger, "", "", version)

    def load_secret(
        self, resource: "IRResource", secret_name: str, namespace: str
    ) -> Optional[SecretInfo]:
        return None


class FSSecretHandler(SecretHandler):
    # XXX NO LONGER USED
    def load_secret(
        self, resource: "IRResource", secret_name: str, namespace: str
    ) -> Optional[SecretInfo]:
        self.logger.debug(
            "FSSecretHandler (%s %s): load secret %s in namespace %s"
            % (resource.kind, resource.name, secret_name, namespace)
        )

        source = os.path.join(self.source_root, namespace, "secrets", "%s.yaml" % secret_name)

        serialization = None

        try:
            serialization = open(source, "r").read()
        except IOError as e:
            self.logger.error(
                "%s %s: FSSecretHandler could not open %s" % (resource.kind, resource.name, source)
            )

        # Yes, this duplicates part of self.secret_info_from_k8s, but whatever.
        objects: Optional[List[Any]] = None

        # If serialization is None or empty, we'll just return None.
        if serialization:
            try:
                objects = parse_yaml(serialization)
            except yaml.error.YAMLError as e:
                self.logger.error(
                    "%s %s: could not parse %s: %s" % (resource.kind, resource.name, source, e)
                )

        if not objects:
            # Nothing in the serialization, we're done.
            return None

        if len(objects) != 1:
            self.logger.error(
                "%s %s: found %d objects in %s instead of exactly 1"
                % (resource.kind, resource.name, len(objects), source)
            )
            return None

        obj = objects[0]

        version = obj.get("apiVersion", None)
        kind = obj.get("kind", None)

        if (kind == "Secret") and (
            version.startswith("ambassador") or version.startswith("getambassador.io")
        ):
            # It's an Ambassador Secret. It should have a public key and maybe a private key.
            secret_type = obj.get("type", "kubernetes.io/tls")
            return SecretInfo.from_dict(
                resource, secret_name, namespace, source, cert_data=obj, secret_type=secret_type
            )

        # Didn't look like an Ambassador object. Try K8s.
        return self.secret_info_from_k8s(resource, secret_name, namespace, source, serialization)


class KubewatchSecretHandler(SecretHandler):
    # XXX NO LONGER USED
    def load_secret(
        self, resource: "IRResource", secret_name: str, namespace: str
    ) -> Optional[SecretInfo]:
        self.logger.debug(
            "FSSecretHandler (%s %s): load secret %s in namespace %s"
            % (resource.kind, resource.name, secret_name, namespace)
        )

        source = "%s/secrets/%s/%s" % (self.source_root, namespace, secret_name)
        serialization = load_url_contents(self.logger, source)

        if not serialization:
            self.logger.error(
                "%s %s: SCC.url_reader could not load %s" % (resource.kind, resource.name, source)
            )

        return self.secret_info_from_k8s(resource, secret_name, namespace, source, serialization)


# TODO(gsagula): This duplicates code from ircluster.py.
class ParsedService:
    def __init__(self, logger, service: str, allow_scheme=True, ctx_name: str = None) -> None:
        original_service = service

        originate_tls = False

        self.scheme = "http"
        self.errors: List[str] = []
        self.name_fields: List[str] = []
        self.ctx_name = ctx_name

        if allow_scheme and service.lower().startswith("https://"):
            service = service[len("https://") :]

            originate_tls = True
            self.name_fields.append("otls")

        elif allow_scheme and service.lower().startswith("http://"):
            service = service[len("http://") :]

            if ctx_name:
                self.errors.append(
                    f"Originate-TLS context {ctx_name} being used even though service {service} lists HTTP"
                )
                originate_tls = True
                self.name_fields.append("otls")
            else:
                originate_tls = False

        elif ctx_name:
            # No scheme (or schemes are ignored), but we have a context.
            originate_tls = True
            self.name_fields.append("otls")
            self.name_fields.append(ctx_name)

        if "://" in service:
            idx = service.index("://")
            scheme = service[0:idx]

            if allow_scheme:
                self.errors.append(
                    f"service {service} has unknown scheme {scheme}, assuming {self.scheme}"
                )
            else:
                self.errors.append(
                    f"ignoring scheme {scheme} for service {service}, since it is being used for a non-HTTP mapping"
                )

            service = service[idx + 3 :]

        # # XXX Should this be checking originate_tls? Why does it do that?
        # if originate_tls and host_rewrite:
        #     name_fields.append("hr-%s" % host_rewrite)

        # Parse the service as a URL. Note that we have to supply a scheme to urllib's
        # parser, because it's kind of stupid.

        logger.debug(
            f"Service: {original_service} otls {originate_tls} ctx {ctx_name} -> {self.scheme}, {service}"
        )
        p = urlparse("random://" + service)

        # Is there any junk after the host?

        if p.path or p.params or p.query or p.fragment:
            self.errors.append(
                f"service {service} has extra URL components; ignoring everything but the host and port"
            )

        # p is read-only, so break stuff out.

        self.hostname = p.hostname
        try:
            self.port = p.port
        except ValueError as e:
            self.errors.append(
                "found invalid port for service {}. Please specify a valid port between 0 and 65535 - {}. Service {} cluster will be ignored, please re-configure".format(
                    service, e, service
                )
            )
            self.port = 0

        # If the port is unset, fix it up.
        if not self.port:
            self.port = 443 if originate_tls else 80

        self.hostname_port = f"{self.hostname}:{self.port}"
