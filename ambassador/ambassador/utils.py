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

from typing import Any, Dict, List, Optional, TextIO, TYPE_CHECKING

import binascii
import hashlib
import io
import socket
import threading
import time
import os
import logging
import requests
import tempfile
import yaml

from .VERSION import Version
from urllib.parse import urlparse

if TYPE_CHECKING:
    from .ir.irtlscontext import IRTLSContext
    from .config.acresource import ACResource

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


def parse_yaml(serialization: str, **kwargs) -> Any:
    global yaml_logged_loader

    if not yaml_logged_loader:
        yaml_logged_loader = True

        logger.info("YAML: using %s parser" % ("Python" if (yaml_loader == yaml.SafeLoader) else "C"))

    return list(yaml.load_all(serialization, Loader=yaml_loader))


def dump_yaml(obj: Any, **kwargs) -> str:
    global yaml_logged_dumper

    if not yaml_logged_dumper:
        yaml_logged_dumper = True

        logger.info("YAML: using %s dumper" % ("Python" if (yaml_dumper == yaml.SafeDumper) else "C"))

    return yaml.dump(obj, Dumper=yaml_dumper, **kwargs)


def _load_url_contents(logger: logging.Logger, url: str, stream1: TextIO, stream2: Optional[TextIO]=None) -> bool:
    saved = False

    try:
        with requests.get(url, stream=True) as r:
            if r.status_code == 200:

                # All's well, pull the config down.
                try:
                    for chunk in r.iter_content(chunk_size=65536):
                        # We do this by hand instead of with 'decode_unicode=True'
                        # above because setting decode_unicode only decodes text,
                        # and WATT hands us application/json...
                        chunk = chunk.decode('utf-8')
                        stream1.write(chunk)

                        if stream2:
                            stream2.write(chunk)

                    saved = True
                except IOError as e:
                    logger.error("couldn't save Kubernetes resources: %s" % e)
                except Exception as e:
                    logger.error("couldn't read Kubernetes resources: %s" % e)
    except requests.exceptions.RequestException as e:
        logger.error("could not load new snapshot: %s" % e)

    return saved


def save_url_contents(logger: logging.Logger, url: str, path: str, stream2: Optional[TextIO]=None) -> bool:
    with open(path, 'w', encoding='utf-8') as stream:
        return _load_url_contents(logger, url, stream, stream2=stream2)


def load_url_contents(logger: logging.Logger, url: str, stream2: Optional[TextIO]=None) -> Optional[str]:
    stream = io.StringIO()

    saved = _load_url_contents(logger, url, stream, stream2=stream2)

    if saved:
        return stream.getvalue()
    else:
        return None


class SystemInfo:
    MyHostName = os.environ.get('HOSTNAME', None)

    if not MyHostName:
        MyHostName = 'localhost'

        try:
            MyHostName = socket.gethostname()
        except:
            pass


class RichStatus:
    def __init__(self, ok, **kwargs):
        self.ok = ok
        self.info = kwargs
        self.info['hostname'] = SystemInfo.MyHostName
        self.info['version'] = Version

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
        attrs = ["%s=%s" % (key, self.info[key]) for key in sorted(self.info.keys())]
        astr = " ".join(attrs)

        if astr:
            astr = " " + astr

        return "<RichStatus %s%s>" % ("OK" if self else "BAD", astr)

    def as_dict(self):
        d = { 'ok': self.ok }

        for key in self.info.keys():
            d[key] = self.info[key]

        return d

    @classmethod
    def fromError(self, error, **kwargs):
        kwargs['error'] = error
        return RichStatus(False, **kwargs)

    @classmethod
    def OK(self, **kwargs):
        return RichStatus(True, **kwargs)


class DelayTrigger (threading.Thread):
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
        self.trigger_source.sendall(b'X')

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
    def __init__(self, name: str, namespace: str,
                 tls_crt: Optional[str], tls_key: Optional[str]=None, decode_b64=True) -> None:
        self.name = name
        self.namespace = namespace

        if decode_b64:
            if tls_crt and not tls_crt.startswith('-----BEGIN'):
                tls_crt = self.decode(tls_crt)

            if tls_key and not tls_key.startswith('-----BEGIN'):
                tls_key = self.decode(tls_key)

        self.tls_crt = tls_crt
        self.tls_key = tls_key

    @staticmethod
    def decode(b64_pem: str) -> Optional[str]:
        utf8_pem = None
        pem = None

        try:
            utf8_pem = binascii.a2b_base64(b64_pem)
        except binascii.Error:
            return None

        try:
            pem = utf8_pem.decode('utf-8')
        except UnicodeDecodeError:
            return None

        return pem

    @staticmethod
    def fingerprint(pem: Optional[str]) -> str:
        if not pem:
            return '<none>'

        h = hashlib.new('sha1')
        h.update(pem.encode('utf-8'))
        hd = h.hexdigest()[0:16].upper()

        keytype = 'PEM' if pem.startswith('-----BEGIN') else 'RAW'

        return f'{keytype}: {hd}'

    def to_dict(self) -> Dict[str, Any]:
        return {
            'name': self.name,
            'namespace': self.namespace,
            'tls_crt': self.fingerprint(self.tls_crt),
            'tls_key': self.fingerprint(self.tls_key)
        }

    @classmethod
    def from_aconf_secret(cls, aconf_object: 'ACResource') -> 'SecretInfo':
        return SecretInfo(
            aconf_object.name,
            aconf_object.namespace,
            aconf_object.tls_crt,
            aconf_object.get('tls_key', None)
        )

    @classmethod
    def from_dict(cls, context: 'IRTLSContext',
                  secret_name: str, namespace: str, source: str,
                  cert_data: Optional[Dict[str, Any]]) -> Optional['SecretInfo']:
        logger = context.ir.logger

        if not cert_data:
            logger.error("TLSContext %s: found no certificate in %s?" % (context.name, source))
            return None

        # OK, we have something to work with. Hopefully.
        cert = cert_data.get('tls.crt', None)

        if not cert:
            # Having no public half is definitely an error. Having no private half given a public half
            # might be OK, though -- that's up to our caller to decide.
            logger.error("TLSContext %s: found data but no cert in %s?" % (context.name, source))
            return None

        key = cert_data.get('tls.key', None)

        return SecretInfo(secret_name, namespace, cert, key)


class SavedSecret:
    def __init__(self, secret_name: str, namespace: str,
                 cert_path: Optional[str], key_path: Optional[str], cert_data: Optional[Dict]) -> None:
        self.secret_name = secret_name
        self.namespace = namespace
        self.cert_path = cert_path
        self.key_path = key_path
        self.cert_data = cert_data

    @property
    def name(self) -> str:
        return "secret %s in namespace %s" % (self.secret_name, self.namespace)

    def __bool__(self) -> bool:
        return bool(bool(self.cert_path) and (self.cert_data is not None))

    def __str__(self) -> str:
        return "<SavedSecret %s.%s -- cert_path %s, key_path %s, cert_data %s>" % (
                  self.secret_name, self.namespace, self.cert_path, self.key_path,
                  "present" if self.cert_data else "absent"
                )


class SecretHandler:
    logger: logging.Logger
    source_root: str
    cache_dir: str

    def __init__(self, logger: logging.Logger, source_root: str, cache_dir: str, version: str) -> None:
        self.logger = logger
        self.source_root = source_root
        self.cache_dir = cache_dir
        self.version = version

    def load_secret(self, context: 'IRTLSContext',
                    secret_name: str, namespace: str) -> Optional[SecretInfo]:
        # This is the fallback load_secret implementation; it is expected that subclasses
        # will override this.
        #
        # All this one does is return None, meaning that it couldn't find the requested
        # secret (because, well, it doesn't really look).
        self.logger.debug(
            f"SecretHandler: Trying to load secret {secret_name} in namespace {namespace} from TLSContext {context}")
        return None

    def cache_secret(self, context: 'IRTLSContext', secret_info: SecretInfo) -> SavedSecret:
        name = secret_info.name
        namespace = secret_info.namespace
        cert = secret_info.tls_crt
        key = secret_info.tls_key

        cert_path = None
        key_path = None
        cert_data = None

        h = hashlib.new('sha1')

        if cert:
            h.update(cert.encode('utf-8'))

            if key:
                h.update(key.encode('utf-8'))

            hd = h.hexdigest().upper()

            secret_dir = os.path.join(self.cache_dir, namespace, "secrets-decoded", name)

            try:
                os.makedirs(secret_dir)
            except FileExistsError:
                pass

            cert_path = os.path.join(secret_dir, f'{hd}.crt')
            open(cert_path, "w").write(cert)

            if key:
                key_path = os.path.join(secret_dir, f'{hd}.key')
                open(key_path, "w").write(key)

            cert_data = {
                'tls_crt': cert,
                'tls_key': key
            }

        return SavedSecret(name, namespace, cert_path, key_path, cert_data)

    # secret_info_from_k8s takes a K8s Secret and returns a SecretInfo (or None if something
    # is wrong).
    def secret_info_from_k8s(self, context: 'IRTLSContext',
                             secret_name: str, namespace: str, source: str,
                             serialization: Optional[str]) -> Optional[SecretInfo]:
        objects: Optional[List[Any]] = None

        self.logger.debug(f"getting secret info for secret {secret_name} from k8s")

        # If serialization is None or empty, we'll just return None.

        if serialization:
            try:
                objects = parse_yaml(serialization)
            except yaml.error.YAMLError as e:
                self.logger.error("TLSContext %s: could not parse %s: %s" %
                                  (context.name, source, e))

        if not objects:
            # Nothing in the serialization, we're done.
            return None

        cert_data = None
        ocount = 0
        errors = 0

        for obj in objects:
            ocount += 1
            kind = obj.get('kind', None)

            if kind != "Secret":
                self.logger.error("TLSContext %s: found K8s %s at %s.%d?" %
                                  (context.name, kind, source, ocount))
                errors += 1
                continue

            metadata = obj.get('metadata', None)

            if not metadata:
                self.logger.error("TLSContext %s: found K8s Secret with no metadata at %s.%d?" %
                                  (context.name, source, ocount))
                errors += 1
                continue

            if 'data' in obj:
                if cert_data:
                    self.logger.error("TLSContext %s: found multiple Secrets in %s?" %
                                      (context.name, source))
                    errors += 1
                    continue

                cert_data = obj['data']

        if errors:
            # Bzzt.
            return None

        return SecretInfo.from_dict(context, secret_name, source, namespace, cert_data)


class NullSecretHandler(SecretHandler):
    def __init__(self, logger: logging.Logger, source_root: Optional[str], cache_dir: Optional[str], version: str) -> None:
        """
        Returns a valid SecretInfo (with fake keys) for any requested secret. Also, you can pass
        None for source_root and cache_dir to use random temporary directories for them.
        """

        if not source_root:
            self.tempdir_source = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-source")
            source_root = self.tempdir_source.name

        if not cache_dir:
            self.tempdir_cache = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-cache")
            cache_dir = self.tempdir_cache.name

        logger.info(f'NullSecretHandler using source_root {source_root}, cache_dir {cache_dir}')

        super().__init__(logger, source_root, cache_dir, version)

    def load_secret(self, context: 'IRTLSContext', secret_name: str, namespace: str) -> Optional[SecretInfo]:
        # In the Real World, the secret loader should, y'know, load secrets..
        # Here we're just gonna fake it.
        self.logger.debug(f"NullSecretHandler: Trying to load secret {secret_name} in namespace {namespace} from TLSContext {context}")
        return SecretInfo(secret_name, namespace, "fake-tls-crt", "fake-tls-key")


class FSSecretHandler(SecretHandler):
    def load_secret(self, context: 'IRTLSContext', secret_name: str, namespace: str) -> Optional[SecretInfo]:
        self.logger.debug(f"FSSecretHandler: Trying to load secret {secret_name} in namespace {namespace} from TLSContext {context}")
        source = os.path.join(self.source_root, namespace, "secrets", "%s.yaml" % secret_name)

        serialization = None

        try:
            serialization = open(source, "r").read()
        except IOError as e:
            self.logger.error("TLSContext %s: FSSecretHandler could not open %s" % (context.name, source))

        # Yes, this duplicates part of self.secret_info_from_k8s, but whatever.
        objects: Optional[List[Any]] = None

        # If serialization is None or empty, we'll just return None.
        if serialization:
            try:
                objects = parse_yaml(serialization)
            except yaml.error.YAMLError as e:
                self.logger.error("TLSContext %s: could not parse %s: %s" %
                                  (context.name, source, e))

        if not objects:
            # Nothing in the serialization, we're done.
            return None

        if len(objects) != 1:
            self.logger.error("TLSContext %s: found %d objects in %s instead of exactly 1" %
                              (context.name, len(objects), source))
            return None

        obj = objects[0]

        version = obj.get('apiVersion', None)
        kind = obj.get('kind', None)

        if version.startswith('ambassador') and (kind == 'Secret'):
            # It's an Ambassador Secret. It should have a public key and maybe a private key.
            return SecretInfo.from_dict(context, secret_name, namespace, source, obj)

        # Didn't look like an Ambassador object. Try K8s.
        return self.secret_info_from_k8s(context, secret_name, namespace, source, serialization)


class KubewatchSecretHandler(SecretHandler):
    def load_secret(self, context: 'IRTLSContext', secret_name: str, namespace: str) -> Optional[SecretInfo]:
        self.logger.debug(f"KubewatchSecretHandler: Trying to load secret {secret_name} in namespace {namespace} from TLSContext {context}")
        source = "%s/secrets/%s/%s" % (self.source_root, namespace, secret_name)
        serialization = load_url_contents(self.logger, source)

        if not serialization:
            self.logger.error("TLSContext %s: SCC.url_reader could not load %s" % (context.name, source))

        return self.secret_info_from_k8s(context, secret_name, namespace, source, serialization)

# TODO(gsagula): This duplicates code from ircluster.py.
class ParsedService:
    def __init__(self, logger, service: str, allow_scheme=True, ctx_name: str=None) -> None:
        original_service = service

        originate_tls = False

        self.scheme = 'http'
        self.errors: List[str] = []
        self.name_fields: List[str] = []
        self.ctx_name = ctx_name

        if allow_scheme and service.lower().startswith("https://"):
            service = service[len("https://"):]

            originate_tls = True
            self.name_fields.append('otls')

        elif allow_scheme and service.lower().startswith("http://"):
            service = service[ len("http://"): ]

            if ctx_name:
                self.errors.append(f'Originate-TLS context {ctx_name} being used even though service {service} lists HTTP')
                originate_tls = True
                self.name_fields.append('otls')
            else:
                originate_tls = False

        elif ctx_name:
            # No scheme (or schemes are ignored), but we have a context.
            originate_tls = True
            self.name_fields.append('otls')
            self.name_fields.append(ctx_name)

        if '://' in service:
            idx = service.index('://')
            scheme = service[0:idx]

            if allow_scheme:
                self.errors.append(f'service {service} has unknown scheme {scheme}, assuming {self.scheme}')
            else:
                self.errors.append(f'ignoring scheme {scheme} for service {service}, since it is being used for a non-HTTP mapping')

            service = service[idx + 3:]

        # # XXX Should this be checking originate_tls? Why does it do that?
        # if originate_tls and host_rewrite:
        #     name_fields.append("hr-%s" % host_rewrite)

        # Parse the service as a URL. Note that we have to supply a scheme to urllib's
        # parser, because it's kind of stupid.

        logger.debug(f'Service: {original_service} otls {originate_tls} ctx {ctx_name} -> {self.scheme}, {service}')
        p = urlparse('random://' + service)

        # Is there any junk after the host?

        if p.path or p.params or p.query or p.fragment:
            self.errors.append(f'service {service} has extra URL components; ignoring everything but the host and port')

        # p is read-only, so break stuff out.

        self.hostname = p.hostname
        try:
            self.port = p.port
        except ValueError as e:
            self.errors.append("found invalid port for service {}. Please specify a valid port between 0 and 65535 - {}. Service {} cluster will be ignored, please re-configure".format(service, e, service))
            self.port = 0

        # If the port is unset, fix it up.
        if not self.port:
            self.port = 443 if originate_tls else 80

        self.hostname_port = f'{self.hostname}:{self.port}'
