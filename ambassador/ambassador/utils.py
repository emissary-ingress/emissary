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

from typing import Dict, Optional, TextIO, TYPE_CHECKING

import binascii
import io
import socket
import threading
import time
import os
import logging
import requests
import yaml

from kubernetes import client, config

from .VERSION import Version

if TYPE_CHECKING:
    from .ir.irtlscontext import IRTLSContext

logger = logging.getLogger("utils")
logger.setLevel(logging.INFO)


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
                    logger.error("couldn't save Kubernetes service resources: %s" % e)
                except Exception as e:
                    logger.error("couldn't read Kubernetes service resources: %s" % e)
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
    MyHostName = 'localhost'
    MyResolvedName = '127.0.0.1'

    try:
        MyHostName = socket.gethostname()
        MyResolvedName = socket.gethostbyname(socket.gethostname())
    except:
        pass

class RichStatus:
    def __init__(self, ok, **kwargs):
        self.ok = ok
        self.info = kwargs
        self.info['hostname'] = SystemInfo.MyHostName
        self.info['resolvedname'] = SystemInfo.MyResolvedName
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


class KubeSecretReader:
    def __init__(self, secret_root: str) -> None:
        self.v1 = None
        self.__name__ = 'KubeSecretReader'
        self.secret_root = secret_root

    def __call__(self, context: 'IRTLSContext', secret_name: str, namespace: str):
        # Make sure we have a Kube connection.
        if not self.v1:
            self.v1 = kube_v1()

        cert_data = None
        cert = None
        key = None

        if self.v1:
            try:
                cert_data = self.v1.read_namespaced_secret(secret_name, namespace)
            except client.rest.ApiException as e:
                if e.reason == "Not Found":
                    logger.info("secret {} not found".format(secret_name))
                else:
                    logger.info("secret %s/%s could not be read: %s" % (namespace, secret_name, e))

        if cert_data and cert_data.data:
            cert_data = cert_data.data
            cert = cert_data.get('tls.crt', None)

            if cert:
                cert = binascii.a2b_base64(cert)

            key = cert_data.get('tls.key', None)

            if key:
                key = binascii.a2b_base64(key)

        secret_dir = os.path.join(self.secret_root, namespace, "secrets", secret_name)

        cert_path = None
        key_path = None

        if cert:
            try:
                os.makedirs(secret_dir)
            except FileExistsError:
                pass

            cert_path = os.path.join(secret_dir, "tls.crt")
            open(cert_path, "w").write(cert.decode("utf-8"))

            if key:
                key_path = os.path.join(secret_dir, "tls.key")
                open(key_path, "w").write(key.decode("utf-8"))

        return SavedSecret(secret_name, namespace, cert_path, key_path, cert_data)


class SecretSaver:
    def __init__(self, logger: logging.Logger, source_root: str, cache_dir: str) -> None:
        self.logger = logger
        self.source_root = source_root
        self.cache_dir = cache_dir

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
                objects.extend(list(yaml.safe_load_all(self.serialization)))
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


def kube_v1():
    # Assume we got nothin'.
    k8s_api = None

    # XXX: is there a better way to check if we are inside a cluster or not?
    if "KUBERNETES_SERVICE_HOST" in os.environ:
        # If this goes horribly wrong and raises an exception (it shouldn't),
        # we'll crash, and Kubernetes will kill the pod. That's probably not an
        # unreasonable response.
        config.load_incluster_config()
        if "AMBASSADOR_VERIFY_SSL_FALSE" in os.environ:
            configuration = client.Configuration()
            configuration.verify_ssl=False
            client.Configuration.set_default(configuration)
        k8s_api = client.CoreV1Api()
    else:
        # Here, we might be running in docker, in which case we'll likely not
        # have any Kube secrets, and that's OK.
        try:
            config.load_kube_config()
            k8s_api = client.CoreV1Api()
        except FileNotFoundError:
            # Meh, just ride through.
            logger.info("No K8s")
            pass

    return k8s_api
