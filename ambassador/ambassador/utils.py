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

from typing import Any, Dict, Optional, TextIO, TYPE_CHECKING

import binascii
import io
import socket
import threading
import time
import os
import logging
import requests
import yaml

from .VERSION import Version

if TYPE_CHECKING:
    from .ir.irtlscontext import IRTLSContext

logger = logging.getLogger("utils")
logger.setLevel(logging.INFO)

yaml_loader = yaml.SafeLoader
yaml_dumper = yaml.SafeDumper

try:
    yaml_loader = yaml.CSafeLoader
except AttributeError:
    pass

try:
    yaml_dumper = yaml.CSafeDumper
except AttributeError:
    pass


def parse_yaml(serialization: str, **kwargs) -> Any:
    if not getattr(parse_yaml, 'logged_info', False):
        parse_yaml.logged_info = True

        logger.info("YAML: using %s parser" % ("Python" if (yaml_loader == yaml.SafeLoader) else "C"))

    return list(yaml.load_all(serialization, Loader=yaml_loader))


def dump_yaml(obj: Any, **kwargs) -> str:
    if not getattr(dump_yaml, 'logged_info', False):
        dump_yaml.logged_info = True

        logger.info("YAML: using %s dumper" % ("Python" if (yaml_dumper == yaml.SafeDumper) else "C"))

    return yaml.dump(obj, Dumper=yaml_dumper, **kwargs)


def _load_url_contents(logger: logging.Logger, url: str, stream1: TextIO, stream2: Optional[TextIO]=None) -> bool:
    saved = False

    try:
        with requests.get(url, stream=True) as r:
            if r.status_code == 200:

                # All's well, pull the config down.
                try:
                    for chunk in r.iter_content(chunk_size=65536, decode_unicode=True):
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


class SecretSaver:
    logger: logging.Logger
    source_root: str
    cache_dir: str
    serialization: Optional[str]

    # These are a little weird because we don't initialize them in __init__.
    # The reason is that they don't actually exist (or need to) until a *reader
    # method gets called, and since all the *reader methods take required values
    # for these things, calling them Optional and setting them to None in
    # __init__ just complicates things.
    context: 'IRTLSContext'
    secret_name: str
    namespace: str

    def __init__(self, logger: logging.Logger, source_root: str, cache_dir: str) -> None:
        self.logger = logger
        self.source_root = source_root
        self.cache_dir = cache_dir

    def null_reader(self, context: 'IRTLSContext', secret_name: str, namespace: str):
        self.context = context
        self.secret_name = secret_name
        self.namespace = namespace

        self.source = os.path.join(self.source_root, namespace, "secrets", "%s.yaml" % secret_name)

        self.serialization = None
        self.logger.error("TLSContext %s: no way to find secret" % context.name)

        return self.secret_parser()

    def file_reader(self, context: 'IRTLSContext', secret_name: str, namespace: str):
        self.context = context
        self.secret_name = secret_name
        self.namespace = namespace

        self.source = os.path.join(self.source_root, namespace, "secrets", "%s.yaml" % secret_name)

        self.serialization = None

        try:
            self.serialization = open(self.source, "r").read()
        except IOError as e:
            self.logger.error("TLSContext %s: SCC.file_reader could not open %s" % (context.name, self.source))

        return self.secret_parser()

    def url_reader(self, context: 'IRTLSContext', secret_name: str, namespace: str):
        self.context = context
        self.secret_name = secret_name
        self.namespace = namespace

        self.source = "%s/secrets/%s/%s" % (self.source_root, namespace, secret_name)
        self.serialization = load_url_contents(self.logger, self.source)

        if not self.serialization:
            self.logger.error("TLSContext %s: SCC.url_reader could not load %s" % (context.name, self.source))

        return self.secret_parser()

    def secret_parser(self) -> SavedSecret:
        objects = []
        cert_data = None
        cert = None
        key = None
        cert_path = None
        key_path = None
        ocount = 0
        errors = 0

        if self.serialization:
            try:
                objects.extend(parse_yaml(self.serialization))
            except yaml.error.YAMLError as e:
                self.logger.error("TLSContext %s: SCC.secret_reader could not parse %s: %s" %
                                  (self.context.name, self.source, e))

        for obj in objects:
            ocount += 1
            kind = obj.get('kind', None)

            if kind != "Secret":
                self.logger.error("TLSContext %s: SCC.secret_reader found K8s %s at %s.%d?" %
                                  (self.context.name, kind, self.source, ocount))
                errors += 1
                continue

            metadata = obj.get('metadata', None)

            if not metadata:
                self.logger.error("TLSContext %s: SCC.secret_reader found K8s Secret with no metadata at %s.%d?" %
                                  (self.context.name, self.source, ocount))
                errors += 1
                continue

            if 'data' in obj:
                if cert_data:
                    self.logger.error("TLSContext %s: SCC.secret_reader found multiple Secrets in %s?" %
                                      (self.context.name, self.source))
                    errors += 1
                    continue

                cert_data = obj['data']

        # if errors:
        #     return None
        #
        # if not cert_data:
        #     self.logger.error("TLSContext %s: SCC.secret_reader found no certificate in %s?" %
        #                       (self.context.name, self.source))
        #     return None

        # OK, we have something to work with. Hopefully.
        if not errors and cert_data:
            cert = cert_data.get('tls.crt', None)

            if cert:
                cert = binascii.a2b_base64(cert)

            key = cert_data.get('tls.key', None)

            if key:
                key = binascii.a2b_base64(key)

        # if not cert:
        #     # This is an error. Having a cert but no key might be OK, we'll let our caller decide.
        #     self.logger.error("TLSContext %s: SCC.secret_reader found data but no cert in %s?" %
        #                       (self.context.name, yaml_path))
        #     return None

        if cert:
            secret_dir = os.path.join(self.cache_dir, self.namespace, "secrets-decoded", self.secret_name)

            try:
                os.makedirs(secret_dir)
            except FileExistsError:
                pass

            cert_path = os.path.join(secret_dir, "tls.crt")
            open(cert_path, "w").write(cert.decode("utf-8"))

            if key:
                key_path = os.path.join(secret_dir, "tls.key")
                open(key_path, "w").write(key.decode("utf-8"))

        return SavedSecret(self.secret_name, self.namespace, cert_path, key_path, cert_data)
