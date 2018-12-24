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

from typing import Dict, Optional, TYPE_CHECKING

import base64
import binascii
import socket
import threading
import time
import os
import logging

from kubernetes import client, config
from enum import Enum

from .VERSION import Version

if TYPE_CHECKING:
    from .ir.irtlscontext import IRTLSContext

logger = logging.getLogger("utils")
logger.setLevel(logging.INFO)


class TLSPaths(Enum):
    mount_cert_dir = "/etc/certs"
    mount_tls_crt = os.path.join(mount_cert_dir, "tls.crt")
    mount_tls_key = os.path.join(mount_cert_dir, "tls.key")

    client_mount_dir = "/etc/cacert"
    client_mount_crt = os.path.join(client_mount_dir, "tls.crt")

    cert_dir = "/ambassador/certs"
    tls_crt = os.path.join(cert_dir, "tls.crt")
    tls_key = os.path.join(cert_dir, "tls.key")

    client_cert_dir = "/ambassador/cacert"
    client_tls_crt = os.path.join(client_cert_dir, "tls.crt")

    @staticmethod
    def generate(directory):
        return {
            'crt': os.path.join(directory, 'tls.crt'),
            'key': os.path.join(directory, 'tls.key')
        }

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

class SourcedDict (dict):
    def __init__(self, _source="--internal--", _from=None, **kwargs):
        super().__init__(self, **kwargs)

        if _from and ('_source' in _from):
            self['_source'] = _from['_source']
        else:
            self['_source'] = _source

        # self['_referenced_by'] = []

    def referenced_by(self, source):
        refby = self.setdefault('_referenced_by', [])

        if source not in refby:
            refby.append(source)

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


def read_cert_secret(k8s_api, secret_name, namespace):
    cert_data = None
    cert = None
    key = None

    try:
        cert_data = k8s_api.read_namespaced_secret(secret_name, namespace)
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

    return cert, key, cert_data


class SavedSecret:
    def __init__(self, secret_name:str, namespace:str,
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
        return bool(bool(self.cert_path) and bool(self.cert_data))

def save_secret(secret_root: str, secret_name: str, namespace: str, cert, key, cert_data) -> SavedSecret:
    # We always return a SavedSecret, so that our caller has access to the name and namespace.
    # The SavedSecret will evaluate non-True if we found no cert though.

    secret_dir = os.path.join(secret_root, namespace, "secrets", secret_name)

    if cert:
        try:
            os.makedirs(secret_dir)
        except FileExistsError:
            pass

        cert_path = os.path.join(secret_dir, "tls.crt")
        open(cert_path, "w").write(cert.decode("utf-8"))

        key_path = None

        if key:
            key_path = os.path.join(secret_dir, "tls.key")
            open(key_path, "w").write(key.decode("utf-8"))

        return SavedSecret(secret_name, namespace, cert_path, key_path, cert_data)
    else:
        return SavedSecret(secret_name, namespace, None, None, None)


def kube_secret_loader(v1, secret_root: str, secret_name: str, namespace: str) -> SavedSecret:
    # Allow secrets to override namespace when needed.

    if "." in secret_name:
        secret_name, namespace = secret_name.split('.', 1)

    cert, key, data = read_cert_secret(v1, secret_name, namespace)

    # We always return a SavedSecret, so that our caller has access to the name and namespace.
    # The SavedSecret will evaluate non-True if we found no cert though.
    return save_secret(secret_root, secret_name, namespace, cert, key, data)


def kube_tls_secret_resolver(context: 'IRTLSContext', namespace: str,
                             get_kube_api: Optional[callable]=None) -> Optional[Dict[str, str]]:
    # If they don't override this, use our own kube_v1.
    if not get_kube_api:
        get_kube_api = kube_v1

    # If we don't have secret info, something is horribly wrong.
    secret_info: Dict[str, str] = context.get('secret_info', {})

    if not secret_info:
        context.post_error("TLSContext %s has no certificate information at all?" % context.name)
        return None

    # OK. Where is the root of the secret store?
    secret_root = os.environ.get('AMBASSADOR_CONFIG_BASE_DIR', "/ambassador")

    # Assume that we aren't going to muck with Kube...
    v1 = None

    # ...and default to returning {}, which will result in the context not being loaded.
    resolved: Dict[str, str] = {}

    logger.info("resolver working on: %s" % context.as_json())

    # OK. Do we have a secret name?
    secret_name = secret_info.get('secret')

    if secret_name:
        # Yes. Should we try to go check with Kube?
        if get_kube_api:
            v1 = get_kube_api()

        if v1:
            ss = kube_secret_loader(v1, secret_root, secret_name, namespace)

            if not ss:
                # This is definitively an error: they mentioned a secret, it can't be loaded,
                # give up.
                context.post_error("TLSContext %s found no certificate in %s" % (context.name, ss.name))
                return None

            # If they only gave a public key, that's an error too.
            if not ss.key_path:
                context.post_error("TLSContext %s found no private key in %s" % (context.name, ss.name))
                return None

            # So far, so good.
            logger.debug("TLSContext %s saved secret %s" % (context.name, ss.name))

            resolved['cert_chain_file'] = ss.cert_path
            resolved['private_key_file'] = ss.key_path
    else:
        # No secret is named. Did they provide file locations?
        missing = False

        cert_chain_file = secret_info.get('cert_chain_file')
        private_key_file = secret_info.get('private_key_file')

        if cert_chain_file:
            resolved['cert_chain_file'] = cert_chain_file
        else:
            missing = True

        if private_key_file:
            resolved['private_key_file'] = private_key_file
        else:
            missing = True

        # If there's no secret name _and_ no path information, that's also an error
        # (which should've already been caught, honestly).

        if missing:
            # Sigh.
            context.post_error("TLSContext %s was given no certificate" % context.name)
            return None

    ca_secret_name = secret_info.get('ca_secret')

    if ca_secret_name:
        if not resolved.get('cert_chain_file'):
            # DUPLICATED BELOW: This is an error: validation without termination isn't meaningful.
            # (This is duplicated for the case where they gave a validation path.)
            context.post_error("TLSContext %s cannot validate client certs without TLS termination" %
                               context.name)
            return None

        # They gave a secret name for the validation cert.. Should we try to go check with Kube?
        if get_kube_api:
            v1 = get_kube_api()

        if v1:
            ss = kube_secret_loader(v1, secret_root, ca_secret_name, namespace)

            if not ss:
                # This is definitively an error: they mentioned a secret, it can't be loaded,
                # give up.
                context.post_error("TLSContext %s found no validation certificate in %s" % (context.name, ss.name))
                return None

            # Validation certs don't need the private key, but it's not an error if they gave
            # one. We're good to go here.
            logger.debug("TLSContext %s saved CA secret %s" % (context.name, ss.name))

            resolved['cacert_chain_file'] = ss.cert_path

            # While we're here, did they set cert_required _in the secret_?
            if ss.cert_data:
                cert_required = ss.cert_data.get('cert_required')

                if cert_required is not None:
                    decoded = base64.b64decode(cert_required).decode('utf-8').lower() == 'true'

                    resolved['cert_required'] = decoded
    else:
        # No secret is named. Copy the path if they gave one, though.
        cacert_chain_file = secret_info.get('cacert_chain_file')

        if cacert_chain_file:
            if not resolved.get('cert_chain_file'):
                # DUPLICATED ABOVE: This is an error: validation without termination isn't meaningful.
                # (This is duplicated for the case where they gave a validation secret.)
                context.post_error("TLSContext %s cannot validate client certs without TLS termination" %
                                   context.name)
                return None

            resolved['cacert_chain_file'] = cacert_chain_file

    # OK. Check paths.
    errors = 0

    logger.info("resolved: %s" % resolved)

    for key in [ 'cert_chain_file', 'private_key_file', 'cacert_chain_file' ]:
        path = resolved.get(key, None)

        if path:
            if not os.path.isfile(path):
                context.post_error("TLSContext %s found no %s '%s'" % (context.name, key, path))
                errors += 1
        elif key != 'cacert_chain_file':
            context.post_error("TLSContext %s is missing %s" % (context.name, key))
            errors += 1

    if errors:
        return None

    return resolved


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


def check_cert_file(path):
    readable = False

    try:
        data = open(path, "r").read()

        if data and (len(data) > 0):
            readable = True
    except OSError:
        pass
    except IOError:
        pass

    return readable
