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

import sys

import collections
import datetime
import json
import logging
import os
import re
from urllib.parse import urlparse

import jsonschema
import semantic_version
import yaml

from pkg_resources import Requirement, resource_filename

from jinja2 import Environment, FileSystemLoader

from .utils import RichStatus, SourcedDict, read_cert_secret, save_cert, TLSPaths, kube_v1, check_cert_file
from .mapping import Mapping

from scout import Scout

from .VERSION import Version

#############################################################################
## config.py -- the main configuration parser for Ambassador
##
## Ambassador configures itself by creating a new Config object, which calls
## Config.__init__().
##
## __init__() sets up all the defaults for everything, then walks over all the
## YAML it can find and calls self.load_yaml() to load each YAML file. After
## everything is loaded, it calls self.process_all_objects() to build the
## config objects.
##
## load_yaml() does the heavy lifting around YAML parsing and such, including
## managing K8s annotations if so requested. Every object in every YAML file is
## parsed and saved before any object is processed.
##
## process_all_objects() walks all the saved objects and creates an internal
## representation of the Ambassador config in the data structures initialized
## by __init__(). Each object is processed with self.process_object(). This
## internal representation is called the intermediate config.
##
## process_object() handles a single parsed object from YAML. It uses
## self.validate_object() to make sure of a schema match; assuming that's
## good, most of the heavy lifting is done by a handler method. The handler
## method for a given type is named handle_kind(), with kind in lowercase,
## so e.g. the Mapping object is processed using the handle_mapping() method.
##
## After all of that, the actual Envoy config is generated from the intermediate
## config using generate_envoy_config().
##
## The diag service also uses generate_intermediate_for() to extract the
## intermediate config for a given mapping or service.

def get_semver(what, version_string):
    semver = None

    try:
        semver = semantic_version.Version(version_string)
    except ValueError:
        pass

    return semver

class Config (object):
    # Weird stuff. The build version looks like
    #
    # 0.12.0                    for a prod build, or
    # 0.12.1-b2.da5d895.DIRTY   for a dev build (in this case made from a dirty true)
    #
    # Now:
    # - Scout needs a build number (semver "+something") to flag a non-prod release;
    #   but
    # - DockerHub cannot use a build number at all; but
    # - 0.12.1-b2 comes _before_ 0.12.1+b2 in SemVer land.
    #
    # FFS.
    #
    # We cope with this by transforming e.g.
    #
    # 0.12.1-b2.da5d895.DIRTY into 0.12.1-b2+da5d895.DIRTY
    #
    # for Scout.

    scout_version = Version

    if '-' in scout_version:
        # TODO(plombardi): This version code needs to be rewritten. We should only report RC and GA versions.
        #
        # As of the time when we moved to streamlined branch, merge and release model the way versions in development
        # land are rendered has changed. A development version no longer has any <MAJOR>.<MINOR>.<PATCH> information and
        # is instead rendered as <BRANCH_NAME>-<GIT_SHORT_HASH>[-dirty] where [-dirty] is only appended for modified
        # source trees.
        #
        # Long term we are planning to remove the version report for development branches anyways so all of this
        # formatting for versions

        scout_version = "0.0.0-" + Version.split("-")[1]  # middle part is commit hash
        # Dev build!
        # v, p = scout_version.split('-')
        # p, b = p.split('.', 1) if ('.' in p) else (0, p)
        #
        # scout_version = "%s-%s+%s" % (v, p, b)

    # Use scout_version here, not __version__, because the version
    # coming back from Scout will use build numbers for dev builds, but
    # __version__ won't, and we need to be consistent for comparison.
    current_semver = get_semver("current", scout_version)

    # When using multiple Ambassadors in one cluster, use AMBASSADOR_ID to distinguish them.
    ambassador_id = os.environ.get('AMBASSADOR_ID', 'default')

    runtime = "kubernetes" if os.environ.get('KUBERNETES_SERVICE_HOST', None) else "docker"
    namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

    # Default to using the Nil UUID unless the environment variable is set explicitly
    scout_install_id = os.environ.get('AMBASSADOR_SCOUT_ID', "00000000-0000-0000-0000-000000000000")

    try:
        scout = Scout(app="ambassador", version=scout_version, install_id=scout_install_id)
        scout_error = None
    except OSError as e:
        scout_error = e

    scout_latest_version = None
    scout_latest_semver = None
    scout_notices = []

    scout_last_response = None
    scout_last_update = datetime.datetime.now() - datetime.timedelta(hours=24)
    scout_update_frequency = datetime.timedelta(hours=4)

    @classmethod
    def scout_report(klass, force_result=None, **kwargs):
        _notices = []

        env_result = os.environ.get("AMBASSADOR_SCOUT_RESULT", None)
        if env_result:
            force_result = json.loads(env_result)

        result = force_result
        result_timestamp = None
        result_was_cached = False

        if not result:
            if Config.scout:
                if 'runtime' not in kwargs:
                    kwargs['runtime'] = Config.runtime

                # How long since the last Scout update? If it's been more than an hour,
                # check Scout again.

                now = datetime.datetime.now()

                if (now - Config.scout_last_update) > Config.scout_update_frequency:
                    result = Config.scout.report(**kwargs)

                    Config.scout_last_update = now
                    Config.scout_last_result = dict(**result)
                else:
                    # _notices.append({ "level": "debug", "message": "Returning cached result" })
                    result = dict(**Config.scout_last_result)
                    result_was_cached = True

                result_timestamp = Config.scout_last_update
            else:
                result = { "scout": "unavailable" }
                result_timestamp = datetime.datetime.now()
        else:
            _notices.append({ "level": "debug", "message": "Returning forced result" })
            result_timestamp = datetime.datetime.now()

        if not Config.current_semver:
            _notices.append({
                "level": "warning",
                "message": "Ambassador has bad version '%s'??!" % Config.scout_version
            })

        result['cached'] = result_was_cached
        result['timestamp'] = result_timestamp.timestamp()

        # Do version & notices stuff.
        if 'latest_version' in result:
            latest_version = result['latest_version']
            latest_semver = get_semver("latest", latest_version)

            if latest_semver:
                Config.scout_latest_version = latest_version
                Config.scout_latest_semver = latest_semver
            else:
                _notices.append({
                    "level": "warning",
                    "message": "Scout returned bad version '%s'??!" % latest_version
                })

        if (Config.scout_latest_semver and
            ((not Config.current_semver) or
             (Config.scout_latest_semver > Config.current_semver))):
            _notices.append({
                "level": "info",
                "message": "Upgrade available! to Ambassador version %s" % Config.scout_latest_semver
            })

        if 'notices' in result:
            _notices.extend(result['notices'])

        Config.scout_notices = _notices

        return result

    def __init__(self, config_dir_path, k8s=False, schema_dir_path=None, template_dir_path=None):
        self.config_dir_path = config_dir_path

        if not template_dir_path:
            template_dir_path = resource_filename(Requirement.parse("ambassador"),"templates")

        if not schema_dir_path:
            schema_dir_path = resource_filename(Requirement.parse("ambassador"),"schemas")

        self.schema_dir_path = schema_dir_path
        self.template_dir_path = template_dir_path
        self.namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

        self.logger = logging.getLogger("ambassador.config")

        self.logger.debug("Scout version %s" % Config.scout_version)
        self.logger.debug("Runtime       %s" % Config.runtime)

        self.logger.debug("CONFIG DIR    %s" % os.path.abspath(self.config_dir_path))
        self.logger.debug("TEMPLATE DIR  %s" % os.path.abspath(self.template_dir_path))
        self.logger.debug("SCHEMA DIR    %s" % os.path.abspath(self.schema_dir_path))

        if Config.scout_error:
            self.logger.warning("Couldn't do version check: %s" % str(Config.scout_error))

        self.schemas = {}
        self.config = {}
        self.tls_contexts = {}

        self.envoy_config = {}
        self.envoy_clusters = {}
        self.envoy_routes = {}

        self.sources = {
            "--internal--": {
                "_source": "--internal--",
                "kind": "Internal",
                "version": "v0",
                "name": "Ambassador Internals",
                "filename": "--internal--",
                "index": 0,
                "description": "The '--internal--' source marks objects created by Ambassador's internal logic."
            },
            "--diagnostics--": {
                "_source": "--diagnostics--",
                "kind": "diagnostics",
                "version": "v0",
                "name": "Ambassador Diagnostics",
                "filename": "--diagnostics--",
                "index": 0,
                "description": "The '--diagnostics--' source marks objects created by Ambassador to assist with diagnostic output."
            }
        }

        self.source_map = {
            '--internal--': { '--internal--': True }
        }

        self.source_overrides = {}

        self.default_liveness_probe = {
            "enabled": True,
            "prefix": "/ambassador/v0/check_alive",
            "rewrite": "/ambassador/v0/check_alive",
            # "service" gets added later
        }

        self.default_readiness_probe = {
            "enabled": True,
            "prefix": "/ambassador/v0/check_ready",
            "rewrite": "/ambassador/v0/check_ready",
            # "service" gets added later
        }

        self.default_diagnostics = {
            "enabled": True,
            "prefix": "/ambassador/v0/",
            "rewrite": "/ambassador/v0/",
            # "service" gets added later
        }

        # 'server' and 'client' are special contexts. Others
        # use cert_chain_file defaulting to context.crt,
        # private_key_file (context.key), and cacert_chain_file
        # (context.pem).

        self.default_tls_config = {
            "server": {},
            "client": {},
        }
        if os.path.isfile(TLSPaths.mount_tls_crt.value):
            self.default_tls_config["server"]["cert_chain_file"] = TLSPaths.mount_tls_crt.value
        if os.path.isfile(TLSPaths.mount_tls_key.value):
            self.default_tls_config["server"]["private_key_file"] = TLSPaths.mount_tls_key.value
        if os.path.isfile(TLSPaths.client_mount_crt.value):
            self.default_tls_config["client"]["cacert_chain_file"] = TLSPaths.client_mount_crt.value

        self.tls_config = None

        self.errors = {}
        self.fatal_errors = 0
        self.object_errors = 0

        self.objects_to_process = []

        if not os.path.isdir(self.config_dir_path):
            raise Exception("ERROR ERROR ERROR configuration directory %s does not exist; exiting" % self.config_dir_path)

        for dirpath, dirnames, filenames in os.walk(self.config_dir_path, topdown=True):
            # Modify dirnames in-place (dirs[:]) to remove any weird directories
            # whose names start with '.' -- why? because my GKE cluster mounts my
            # ConfigMap with a self-referential directory named
            # /etc/ambassador-config/..9989_25_09_15_43_06.922818753, and if we don't
            # ignore that, we end up trying to read the same config files twice, which
            # triggers the collision checks. Sigh.

            dirnames[:] = sorted([ d for d in dirnames if not d.startswith('.') ])

            # self.logger.debug("WALK %s: dirs %s, files %s" % (dirpath, dirnames, filenames))

            for filename in sorted([ x for x in filenames if x.endswith(".yaml") ]):
                filepath = os.path.join(dirpath, filename)

                self.load_yaml(filepath, filename, open(filepath, "r").read(), ocount=1, k8s=k8s)

        self.process_all_objects()

        if self.fatal_errors:
            # Kaboom.
            raise Exception("ERROR ERROR ERROR Unparseable configuration; exiting")

        if self.errors:
            self.logger.error("ERROR ERROR ERROR Starting with configuration errors")

        self.generate_intermediate_config()

    def load_yaml(self, filepath, filename, serialization, resource_identifier=None, ocount=1, k8s=False):
        try:
            # XXX This is a bit of a hack -- yaml.safe_load_all returns a
            # generator, and if we don't use list() here, any exception
            # dealing with the actual object gets deferred
            for obj in yaml.safe_load_all(serialization):
                if k8s:
                    ocount = self.prep_k8s(filepath, filename, ocount, obj)
                else:
                    # k8s objects will have an identifier, for other objects use filepath
                    object_unique_id = resource_identifier or filepath
                    self.objects_to_process.append((object_unique_id, filename, ocount, obj))
                    ocount += 1
        except Exception as e:
            # No sense letting one attribute with bad YAML take down the whole
            # gateway, so post the error but keep any objects we were able to
            # parse before hitting the error.
            self.resource_identifier = resource_identifier or filepath
            self.filename = filename
            self.ocount = ocount

            self.post_error(RichStatus.fromError("%s: could not parse YAML" % filepath))

        return ocount

    def prep_k8s(self, filepath, filename, ocount, obj):
        kind = obj.get('kind', None)

        if kind != "Service":
            self.logger.debug("%s/%s: ignoring K8s %s object" %
                             (filepath, ocount, kind))
            return ocount

        metadata = obj.get('metadata', None)

        if not metadata:
            self.logger.debug("%s/%s: ignoring unannotated K8s %s" %
                              (filepath, ocount, kind))
            return ocount

        # Use metadata to build an unique resource identifier
        resource_name = metadata.get('name')

        # This should never happen as the name field is required in metadata for Service
        if not resource_name:
            self.logger.debug("%s/%s: ignoring unnamed K8s %s" %
                              (filepath, ocount, kind))
            return ocount

        resource_namespace = metadata.get('namespace', 'default')

        # This resource identifier is useful for log output since filenames can be duplicated (multiple subdirectories)
        resource_identifier = '{name}.{namespace}'.format(namespace=resource_namespace, name=resource_name)

        annotations = metadata.get('annotations', None)

        if annotations:
            annotations = annotations.get('getambassador.io/config', None)

        # self.logger.debug("annotations %s" % annotations)

        if not annotations:
            self.logger.debug("%s/%s: ignoring K8s %s without Ambassador annotation" %
                              (filepath, ocount, kind))
            return ocount

        return self.load_yaml(filepath, filename + ":annotation", annotations, ocount=ocount, resource_identifier=resource_identifier)

    def process_all_objects(self):
        for resource_identifier, filename, ocount, obj in sorted(self.objects_to_process):
            # resource_identifier is either a filepath or <name>.<namespace>
            self.resource_identifier = resource_identifier
            # This fallback prevents issues for internal/diagnostics objects
            self.filename = filename
            self.ocount = ocount

            if self.filename in self.source_overrides:
                # Let Pragma objects override source information for this filename.
                override = self.source_overrides[self.filename]

                self.source = override.get('source', self.filename)
                self.ocount += override.get('ocount_delta', 0)
            else:
                # No pragma involved here; just default to the filename.
                self.source = self.filename

            # Is the object empty?
            if obj == None :
                self.logger.debug("Annotation has empty config")
                return

            # Is an ambassador_id present in this object?
            allowed_ids = obj.get('ambassador_id', 'default')

            if allowed_ids:
                # Make sure it's a list. Yes, this is Draconian,
                # but the jsonschema will allow only a string or a list,
                # and guess what? Strings are iterables.
                if type(allowed_ids) != list:
                    allowed_ids = [ allowed_ids ]

                if Config.ambassador_id not in allowed_ids:
                    self.logger.debug("PROCESS: skip %s.%d; id %s not in %s" %
                                      (self.resource_identifier, self.ocount, Config.ambassador_id, allowed_ids))
                    continue

            self.logger.debug("PROCESS: %s.%d => %s" % (self.resource_identifier, self.ocount, self.source))

            rc = self.process_object(obj)

            if not rc:
                # Object error. Not good but we'll allow the system to start.
                self.post_error(rc)

    def clean_and_copy(self, d):
        out = []

        for key in sorted(d.keys()):
            original = d[key]
            copy = dict(**original)

            if '_source' in original:
                del(original['_source'])

            if '_referenced_by' in original:
                del(original['_referenced_by'])

            out.append(copy)

        return out

    def current_source_key(self):
        return("%s.%d" % (self.filename, self.ocount))

    def post_error(self, rc, key=None):
        if not key:
            key = self.current_source_key()

        # Yuck.
        filename = re.sub(r'\.\d+$', '', key)

        # Fetch the relevant source info. If it doesn't exist, stuff
        # in a fake record.
        source_info = self.sources.setdefault(key, {
            'kind': 'error',
            'version': 'error',
            'name': 'error',
            'filename': filename,
            'index': self.ocount,
            'yaml': 'error'
        })

        source_info.setdefault('errors', [])
        source_info['errors'].append(rc.toDict())

        source_map = self.source_map.setdefault(filename, {})
        source_map[key] = True

        errors = self.errors.setdefault(key, [])
        errors.append(rc.toDict())
        self.logger.error("%s (%s): %s" % (key, filename, rc))

    def process_object(self, obj):
        # Cache the source key first thing...
        source_key = self.current_source_key()

        # This should be impossible.
        if not obj:
            return RichStatus.fromError("undefined object???")

        try:
            obj_version = obj['apiVersion']
            obj_kind = obj['kind']
        except KeyError:
            return RichStatus.fromError("need apiVersion, kind")

        # Is this a pragma object?
        if obj_kind == 'Pragma':
            # Yes. Handle this inline and be done.
            return self.handle_pragma(source_key, obj)

        # Not a pragma. It needs a name...
        if 'name' not in obj:
            return RichStatus.fromError("need name")

        obj_name = obj['name']

        # ...and off we go. Save the source info...
        self.sources[source_key] = {
            'kind': obj_kind,
            'version': obj_version,
            'name': obj_name,
            'filename': self.filename,
            'index': self.ocount,
            'yaml': yaml.safe_dump(obj, default_flow_style=False)
        }

        # ...and figure out if this thing is OK.
        rc = self.validate_object(obj)

        if not rc:
            # Well that's no good.
            return rc

        # Make sure it has a source: use what's in the object if present,
        # otherwise use self.source.
        self.sources[source_key]['_source'] = obj.get('source', self.source)
        # self.logger.debug("source for %s is %s" % (source_key, self.sources[source_key]['_source']))

        source_map = self.source_map.setdefault(self.filename, {})
        source_map[source_key] = True

        # OK, so far so good. Grab the handler for this object type.
        handler_name = "handle_%s" % obj_kind.lower()
        handler = getattr(self, handler_name, None)

        if not handler:
            handler = self.save_object
            self.logger.warning("%s[%d]: no handler for %s, just saving" %
                                (self.resource_identifier, self.ocount, obj_kind))
        # else:
        #     self.logger.debug("%s[%d]: handling %s..." %
        #                       (self.filename, self.ocount, obj_kind))

        try:
            handler(source_key, obj, obj_name, obj_kind, obj_version)
        except Exception as e:
            # Bzzzt.
            return RichStatus.fromError("could not process %s object: %s" % (obj_kind, e))

        # OK, all's well.
        return RichStatus.OK(msg="%s object processed successfully" % obj_kind)

    def validate_object(self, obj):
        # Each object must be a dict, and must include "apiVersion"
        # and "type" at toplevel.

        if not isinstance(obj, collections.Mapping):
            return RichStatus.fromError("not a dictionary")

        if not (("apiVersion" in obj) and ("kind" in obj) and ("name" in obj)):
            return RichStatus.fromError("must have apiVersion, kind, and name")

        obj_version = obj['apiVersion']
        obj_kind = obj['kind']
        obj_name = obj['name']

        if obj_version.startswith("ambassador/"):
            obj_version = obj_version.split('/')[1]
        else:
            return RichStatus.fromError("apiVersion %s unsupported" % obj_version)

        schema_key = "%s-%s" % (obj_version, obj_kind)

        schema = self.schemas.get(schema_key, None)

        if not schema:
            schema_path = os.path.join(self.schema_dir_path, obj_version,
                                       "%s.schema" % obj_kind)

            try:
                schema = json.load(open(schema_path, "r"))
            except OSError:
                self.logger.debug("no schema at %s, skipping" % schema_path)
            except json.decoder.JSONDecodeError as e:
                self.logger.warning("corrupt schema at %s, skipping (%s)" %
                                    (schema_path, e))

        if schema:
            self.schemas[schema_key] = schema
            try:
                jsonschema.validate(obj, schema)
            except jsonschema.exceptions.ValidationError as e:
                return RichStatus.fromError("not a valid %s: %s" % (obj_kind, e))

        return RichStatus.OK(msg="valid %s" % obj_kind,
                             details=(obj_kind, obj_version, obj_name))

    def safe_store(self, source_key, storage_name, obj_name, obj_kind, value, allow_log=True):
        storage = self.config.setdefault(storage_name, {})

        if obj_name in storage:
            # Oooops.
            raise Exception("%s[%d] defines %s %s, which is already present" %
                            (self.resource_identifier, self.ocount, obj_kind, obj_name))

        if allow_log:
            self.logger.debug("%s[%d]: saving %s %s" %
                          (self.resource_identifier, self.ocount, obj_kind, obj_name))

        storage[obj_name] = value
        return storage[obj_name]

    def save_object(self, source_key, obj, obj_name, obj_kind, obj_version):
        return self.safe_store(source_key, obj_kind, obj_name, obj_kind,
                               SourcedDict(_source=source_key, **obj))

    def handle_pragma(self, source_key, obj):
        keylist = sorted([x for x in sorted(obj.keys()) if ((x != 'apiVersion') and (x != 'kind'))])

        # self.logger.debug("PRAGMA: %s" % keylist)

        for key in keylist:
            if key == 'source':
                override = self.source_overrides.setdefault(self.filename, {})
                override['source'] = obj['source']

                self.logger.debug("PRAGMA: override %s to %s" %
                                  (self.resource_identifier, self.source_overrides[self.filename]['source']))
            elif key == 'autogenerated':
                override = self.source_overrides.setdefault(self.filename, {})
                override['ocount_delta'] = -1

            #     self.logger.debug("PRAGMA: autogenerated, setting ocount_delta to -1")
            # else:
            #     self.logger.debug("PRAGMA: skip %s" % key)

        return RichStatus.OK(msg="handled pragma object")

    def handle_module(self, source_key, obj, obj_name, obj_kind, obj_version):
        return self.safe_store(source_key, "modules", obj_name, obj_kind,
                               SourcedDict(_source=source_key, **obj['config']))

    def handle_ratelimitservice(self, source_key, obj, obj_name, obj_kind, obj_version):
        return self.safe_store(source_key, "ratelimit_configs", obj_name, obj_kind,
                               SourcedDict(_source=source_key, **obj))

    def handle_tracingservice(self, source_key, obj, obj_name, obj_kind, obj_version):
        return self.safe_store(source_key, "tracing_configs", obj_name, obj_kind,
                               SourcedDict(_source=source_key, **obj))

    def handle_authservice(self, source_key, obj, obj_name, obj_kind, obj_version):
        return self.safe_store(source_key, "auth_configs", obj_name, obj_kind,
                               SourcedDict(_source=source_key, **obj))

    def handle_mapping(self, source_key, obj, obj_name, obj_kind, obj_version):
        mapping = Mapping(source_key, **obj)

        return self.safe_store(source_key, "mappings", obj_name, obj_kind, mapping)

    def diag_port(self):
        modules = self.config.get("modules", {})
        amod = modules.get("ambassador", {})

        return amod.get("diag_port", 8877)

    def diag_service(self):
        return "127.0.0.1:%d" % self.diag_port()

    def add_intermediate_cluster(self, _source, name, _service, urls,
                                 type="strict_dns", lb_type="round_robin",
                                 cb_name=None, od_name=None, originate_tls=None,
                                 grpc=False, host_rewrite=None, ssl_context=None):
        if name not in self.envoy_clusters:
            self.logger.debug("CLUSTER %s: new from %s" % (name, _source))

            cluster = SourcedDict(
                _source=_source,
                _referenced_by=[ _source ],
                _service=_service,
                name=name,
                type=type,
                lb_type=lb_type,
                urls=urls
            )

            if cb_name and (cb_name in self.breakers):
                cluster['circuit_breakers'] = self.breakers[cb_name]
                self.breakers[cb_name]._mark_referenced_by(_source)

            if od_name and (od_name in self.outliers):
                cluster['outlier_detection'] = self.outliers[od_name]
                self.outliers[od_name]._mark_referenced_by(_source)

            if originate_tls == True:
                cluster['tls_context'] = { '_ambassador_enabled': True }
                cluster['tls_array'] = []
            elif (originate_tls and (originate_tls in self.tls_contexts)):
                cluster['tls_context'] = self.tls_contexts[originate_tls]
                self.tls_contexts[originate_tls]._mark_referenced_by(_source)

                tls_array = []

                for key, value in cluster['tls_context'].items():
                    if key.startswith('_'):
                        continue

                    tls_array.append({ 'key': key, 'value': value })
                    cluster['tls_array'] = sorted(tls_array, key=lambda x: x['key'])
            elif ssl_context:
                cluster['tls_context'] = ssl_context
                tls_array = []
                for key, value in ssl_context.items():
                    tls_array.append({ 'key': key, 'value': value })
                    cluster['tls_array'] = sorted(tls_array, key=lambda x: x['key'])

            if host_rewrite and originate_tls:
                cluster['tls_array'].append({'key': 'sni', 'value': host_rewrite })

            if grpc:
                cluster['features'] = 'http2'

            self.envoy_clusters[name] = cluster
        else:
            self.logger.debug("CLUSTER %s: referenced by %s" % (name, _source))

            self.envoy_clusters[name]._mark_referenced_by(_source)

    # XXX This is a silly API. We should have a Cluster object that can carry what kind
    #     of cluster it is (this is a target cluster of weight 50%, this is a shadow cluster,
    #     whatever) and the API should be "add this cluster to this Mapping".
    def add_intermediate_route(self, _source, mapping, svc, cluster_name, shadow=False):
        route = self.envoy_routes.get(mapping.group_id, None)
        host_redirect = mapping.get('host_redirect', False)
        shadow = mapping.get('shadow', False)

        if route:
            # Is this a host_redirect? If so, that's an error.
            if host_redirect:
                self.logger.error("ignoring non-unique host_redirect mapping %s (see also %s)" %
                                  (mapping['name'], route['_source']))

            # Is this a shadow? If so, is there already a shadow marked?
            elif shadow:
                extant_shadow = route.get('shadow', None)

                if extant_shadow:
                    shadow_name = extant_shadow.get('name', None)

                    if shadow_name != cluster_name:
                        self.logger.error("mapping %s defines multiple shadows! Ignoring %s" %
                                        (mapping['name'], cluster_name))
                else:
                    # XXX CODE DUPLICATION with mapping.py!!
                    # We're going to need to support shadow weighting later, so use a dict here.
                    route['shadow'] = {
                        'name': cluster_name
                    }
                    route.setdefault('clusters', [])
            else:
                # Take the easy way out -- just add a new entry to this
                # route's set of weighted clusters.
                route["clusters"].append( { "name": cluster_name,
                                            "weight": mapping.attrs.get("weight", None) } )

            route._mark_referenced_by(_source)

            return

        # OK, if here, we don't have an extent route group for this Mapping. Make a
        # new one.
        route = mapping.new_route(svc, cluster_name)
        self.envoy_routes[mapping.group_id] = route

    def service_tls_check(self, svc, context, host_rewrite):
        originate_tls = False
        name_fields = None

        if svc.lower().startswith("http://"):
            originate_tls = False
            svc = svc[len("http://"):]
        elif svc.lower().startswith("https://"):
            originate_tls = True
            name_fields = [ 'otls' ]
            svc = svc[len("https://"):]
        elif context == True:
            originate_tls = True
            name_fields = [ 'otls' ]

        # Separate if here because you need to be able to specify a context
        # even after you say "https://" for the service.

        if context and (context != True):
            if context in self.tls_contexts:
                name_fields = [ 'otls', context ]
                originate_tls = context
            else:
                self.logger.error("Originate-TLS context %s is not defined" % context)

        if originate_tls and host_rewrite:
            name_fields.append("hr-%s" % host_rewrite)

        port = 443 if originate_tls else 80
        context_name = "_".join(name_fields) if name_fields else None

        svc_url = 'tcp://%s' % svc

        if ':' not in svc:
            svc_url = '%s:%d' % (svc_url, port)

        return (svc, svc_url, originate_tls, context_name)

    def add_clusters_for_mapping(self, mapping):
        svc = mapping['service']
        tls_context = mapping.get('tls', None)
        grpc = mapping.get('grpc', False)
        host_rewrite = mapping.get('host_rewrite', None)

        # Given the service and the TLS context, first initialize the cluster name for the
        # main service with the incoming service string...

        cluster_name_fields = [ svc ]

        host_redirect = mapping.get('host_redirect', False)
        shadow = mapping.get('shadow', False)

        if host_redirect:
            if shadow:
                # Not allowed.
                errstr = "At most one of host_redirect and shadow may be set; ignoring host_redirect"
                self.post_error(RichStatus.fromError(errstr), key=mapping['_source'])
                host_redirect = False
            else:
                # Short-circuit. You needn't actually create a cluster for a
                # host_redirect mapping.
                return svc, None

        if shadow:
            cluster_name_fields.insert(0, "shadow")

        # ...then do whatever normalization we need for the name and the URL. This can
        # change the service name (e.g. "http://foo" will turn into "foo"), so we set
        # up cluster_name_fields above in order to preserve compatibility with older
        # versions of Ambassador. (This isn't a functional issue, just a matter of
        # trying not to confuse people on upgrades.)

        (svc, url, originate_tls, otls_name) = self.service_tls_check(svc, tls_context, host_rewrite)

        # Build up the common name stuff that we'll need for the service and
        # the shadow service.

        aux_name_fields = []

        cb_name = mapping.get('circuit_breaker', None)

        if cb_name:
            if cb_name in self.breakers:
                aux_name_fields.append("cb_%s" % cb_name)
            else:
                self.logger.error("CircuitBreaker %s is not defined (mapping %s)" %
                                  (cb_name, mapping.name))

        od_name = mapping.get('outlier_detection', None)

        if od_name:
            if od_name in self.outliers:
                aux_name_fields.append("od_%s" % od_name)
            else:
                self.logger.error("OutlierDetection %s is not defined (mapping %s)" %
                                  (od_name, mapping.name))

        # OK. Use the main service stuff to build up the main clustor.

        if otls_name:
            cluster_name_fields.append(otls_name)

        cluster_name_fields.extend(aux_name_fields)

        cluster_name = 'cluster_%s' % "_".join(cluster_name_fields)
        cluster_name = re.sub(r'[^0-9A-Za-z_]', '_', cluster_name)

        self.logger.debug("%s: svc %s -> cluster %s" % (mapping.name, svc, cluster_name))

        self.add_intermediate_cluster(mapping['_source'], cluster_name,
                                      svc, [ url ],
                                      cb_name=cb_name, od_name=od_name, grpc=grpc,
                                      originate_tls=originate_tls, host_rewrite=host_rewrite)

        return svc, cluster_name

    def merge_tmods(self, tls_module, generated_module, key):
        """
        Merge TLS module configuration for a particular key. In the event of conflicts, the 
        tls_module element wins, and an error is posted so that the diagnostics service can 
        show it.

        Returns a TLS module with a correctly-merged config element. This will be the
        tls_module (possibly modified) unless no tls_module is present, in which case
        the generated_module will be promoted. If any changes were made to the module, it
        will be marked as referenced by the generated_module.

        :param tls_module: the `tls` module; may be None
        :param generated_module: the `tls-from-ambassador-certs` module; may be None
        :param key: the key in the module config to merge
        :return: TLS module object; see above.
        """

        # First up, the easy cases. If either module is missing, return the other.
        # (The other might be None too, of course.)
        if generated_module is None:
            return tls_module
        elif tls_module is None:
            return generated_module
        else:
            self.logger.debug("tls_module %s" % json.dumps(tls_module, indent=4))
            self.logger.debug("generated_module %s" % json.dumps(generated_module, indent=4))

            # OK, no easy cases. We know that both modules exist: grab the config dicts.
            tls_source = tls_module['_source']
            tls_config = tls_module.get(key, {})

            gen_source = generated_module['_source']
            gen_config = generated_module.get(key, {})

            # Now walk over the tls_config and copy anything needed.
            any_changes = False

            for ckey in gen_config:
                if ckey in tls_config:
                    # ckey exists in both modules. Do they have the same value?
                    if tls_config[ckey] != gen_config[ckey]:
                        # No -- post an error, but let the version from the TLS module win.
                        errfmt = "CONFLICT in TLS config for {}.{}: using {} from TLS module in {}"
                        errstr = errfmt.format(key, ckey, tls_config[ckey], tls_source)
                        self.post_error(RichStatus.fromError(errstr))
                    else:
                        # They have the same value. Worth mentioning in debug.
                        self.logger.debug("merge_tmods: {}.{} duplicated with same value".format(key, ckey))
                else:
                    # ckey only exists in gen_config. Copy it over.
                    self.logger.debug("merge_tmods: copy {}.{} from gen_config".format(key, ckey))
                    tls_config[ckey] = gen_config[ckey]
                    any_changes = True

            # If we had changes...
            if any_changes:
                # ...then mark the tls_module as referenced by the generated_module's
                # source..
                tls_module._mark_referenced_by(gen_source)

                # ...and copy the tls_config back in (in case the key wasn't in the tls_module 
                # config at all originally).
                tls_module[key] = tls_config

            # Finally, return the tls_module.
            return tls_module

    def generate_intermediate_config(self):
        # First things first. The "Ambassador" module always exists; create it with
        # default values now.

        self.ambassador_module = SourcedDict(
            service_port = 80,
            admin_port = 8001,
            diag_port = 8877,
            auth_enabled = None,
            liveness_probe = { "enabled": True },
            readiness_probe = { "enabled": True },
            diagnostics = { "enabled": True },
            tls_config = None,
            use_proxy_proto = False,
            x_forwarded_proto_redirect = False,
        )

        # Next up: let's define initial clusters, routes, and filters.
        #
        # Our set of clusters starts out empty; we use add_intermediate_cluster()
        # to build it up while making sure that all the source-tracking stuff
        # works out.
        #
        # Note that we use a map for clusters, not a list -- the reason is that
        # multiple mappings can use the same service, and we don't want multiple
        # clusters.
        self.envoy_clusters = {}

        # Our initial set of routes is empty...
        self.envoy_routes = {}

        # Our initial list of grpc_services is empty...
        self.envoy_config['grpc_services'] = []

        # Now we look at user-defined modules from our config...
        modules = self.config.get('modules', {})

        # ...most notably the 'ambassador' and 'tls' modules, which are handled first.
        amod = modules.get('ambassador', None)
        tls_module = modules.get('tls', None)

        # Part of handling the 'tls' module is folding in the 'tls-from-ambassador-certs'
        # module, so grab that too...
        generated_module = modules.get('tls-from-ambassador-certs', None)

        # ...and merge the 'server' and 'client' config elements.
        tls_module = self.merge_tmods(tls_module, generated_module, 'server')
        tls_module = self.merge_tmods(tls_module, generated_module, 'client')

        # OK, done. Make sure we have _something_ for the TLS module going forward.
        tmod = tls_module or {}
        self.logger.debug("TLS module after merge: %s" % json.dumps(tmod, indent=4))

        if amod or tmod:
            self.module_config_ambassador("ambassador", amod, tmod)

        router_config = {}

        tracing_configs = self.config.get('tracing_configs', None)
        self.module_config_tracing(tracing_configs)
        if 'tracing' in self.envoy_config:
            router_config['start_child_span'] = True

        # !!!! WARNING WARNING WARNING !!!! Filters are actually ORDER-DEPENDENT.
        self.envoy_config['filters'] = []
        # Start with authentication filter
        auth_mod = modules.get('authentication', None)
        auth_configs = self.config.get('auth_configs', None)
        auth_filter = self.module_config_authentication("authentication", amod, auth_mod, auth_configs)
        if auth_filter:
            self.envoy_config['filters'].append(auth_filter)
        # Then append the rate-limit filter, because we might rate-limit based on auth headers
        ratelimit_configs = self.config.get('ratelimit_configs', None)
        (ratelimit_filter, ratelimit_grpc_service) = self.module_config_ratelimit(ratelimit_configs)
        if ratelimit_filter and ratelimit_grpc_service:
            self.envoy_config['filters'].append(ratelimit_filter)
            self.envoy_config['grpc_services'].append(ratelimit_grpc_service)
        # Then append non-configurable cors and decoder filters
        self.envoy_config['filters'].append(SourcedDict(name="cors", config={}))
        self.envoy_config['filters'].append(SourcedDict(type="decoder", name="router", config=router_config))

        # For mappings, start with empty sets for everything.
        mappings = self.config.get("mappings", {})

        self.breakers = self.config.get("CircuitBreaker", {})

        for key, breaker in self.breakers.items():
            breaker['_referenced_by'] = []

        self.outliers = self.config.get("OutlierDetection", {})

        for key, outlier in self.outliers.items():
            outlier['_referenced_by'] = []

        # OK. Given those initial sets, let's look over our global modules.
        for module_name in modules.keys():
            if ((module_name == 'ambassador') or
                (module_name == 'tls') or
                (module_name == 'authentication') or
                (module_name == 'tls-from-ambassador-certs')):
                continue

            handler_name = "module_config_%s" % module_name
            handler = getattr(self, handler_name, None)

            if not handler:
                self.logger.error("module %s: no configuration generator, skipping" % module_name)
                continue

            handler(module_name, modules[module_name])

        # Once modules are handled, we can set up our admin config...
        self.envoy_config['admin'] = SourcedDict(
            _from=self.ambassador_module,
            admin_port=self.ambassador_module["admin_port"]
        )

        # ...and our listeners.
        primary_listener = SourcedDict(
            _from=self.ambassador_module,
            service_port=self.ambassador_module["service_port"],
            require_tls=False,
            use_proxy_proto=self.ambassador_module['use_proxy_proto']
        )

        if 'use_remote_address' in self.ambassador_module:
            primary_listener['use_remote_address'] = self.ambassador_module['use_remote_address']

        # If x_forwarded_proto_redirect is set, then we enable require_tls in primary listener, which in turn sets
        # require_ssl to true in envoy config. Once set, then all requests that contain X-FORWARDED-PROTO set to
        # https, are processes normally by envoy. In all the other cases, including X-FORWARDED-PROTO set to http,
        # a 301 redirect response to https://host is sent
        if self.ambassador_module.get('x_forwarded_proto_redirect', False):
            primary_listener['require_tls'] = True
            self.logger.debug("x_forwarded_proto_redirect is set to true, enabling 'require_tls' in listener")

        redirect_cleartext_from = None
        tmod = self.ambassador_module.get('tls_config', None)

        # ...TLS config, if necessary...
        if tmod:
            # self.logger.debug("USING TLS")
            primary_listener['tls'] = tmod
            if self.tmod_certs_exist(primary_listener['tls']) > 0:
                primary_listener['tls']['ssl_context'] = True
            redirect_cleartext_from = tmod.get('redirect_cleartext_from')

        self.envoy_config['listeners'] = [ primary_listener ]

        if redirect_cleartext_from:
            # We only want to set `require_tls` on the primary listener when certs are present on the pod
            if self.tmod_certs_exist(primary_listener['tls']) > 0:
                primary_listener['require_tls'] = True

            new_listener = SourcedDict(
                _from=self.ambassador_module,
                service_port=redirect_cleartext_from,
                require_tls=True,
                # Note: no TLS context here, this is a cleartext listener.
                # We can set require_tls True because we can let the upstream
                # tell us about that.
                use_proxy_proto=self.ambassador_module['use_proxy_proto']
            )

            if 'use_remote_address' in self.ambassador_module:
                new_listener['use_remote_address'] = self.ambassador_module['use_remote_address']

            self.envoy_config['listeners'].append(new_listener)

        self.default_liveness_probe['service'] = self.diag_service()
        self.default_readiness_probe['service'] = self.diag_service()
        self.default_diagnostics['service'] = self.diag_service()

        for name, cur, dflt in [
            ("liveness",    self.ambassador_module['liveness_probe'],
                            self.default_liveness_probe),
            ("readiness",   self.ambassador_module['readiness_probe'],
                            self.default_readiness_probe),
            ("diagnostics", self.ambassador_module['diagnostics'],
                            self.default_diagnostics)
        ]:
            if cur and cur.get("enabled", False):
                prefix = cur.get("prefix", dflt['prefix'])
                rewrite = cur.get("rewrite", dflt['rewrite'])
                service = cur.get("service", dflt['service'])

                # Push a fake mapping to handle this.
                name = "internal_%s_probe_mapping" % name

                mappings[name] = Mapping(
                    _from=self.ambassador_module,
                    kind='Mapping',
                    name=name,
                    prefix=prefix,
                    rewrite=rewrite,
                    service=service
                )

                # self.logger.debug("PROBE %s: %s -> %s%s" % (name, prefix, service, rewrite))

        # OK! We have all the mappings we need. Process them (don't worry about sorting
        # yet, we'll do that on routes).

        for mapping_name in sorted(mappings.keys()):
            mapping = mappings[mapping_name]

            # OK. Set up clusters for this service...
            svc, cluster_name = self.add_clusters_for_mapping(mapping)

            # ...and route.
            self.add_intermediate_route(mapping['_source'], mapping, svc, cluster_name)

        # OK. Walk the set of clusters and normalize names...
        collisions = {}
        mangled = {}

        for name in sorted(self.envoy_clusters.keys()):
            if len(name) > 60:
                # Too long.
                short_name = name[0:40]

                collision_list = collisions.setdefault(short_name, [])
                collision_list.append(name)

        for short_name in sorted(collisions.keys()):
            name_list = collisions[short_name]

            i = 0

            for name in sorted(name_list):
                mangled_name = "%s-%d" % (short_name, i)
                i += 1

                self.logger.info("%s => %s" % (name, mangled_name))

                mangled[name] = mangled_name
                self.envoy_clusters[name]['name'] = mangled_name

        # We need to default any unspecified weights and renormalize to 100
        for group_id, route in self.envoy_routes.items():
            clusters = route["clusters"]

            total = 0.0
            unspecified = 0

            # If this is a websocket route, it will support only one cluster right now.
            if route.get('use_websocket', False):
                if len(clusters) > 1:
                    errmsg = "Only one cluster is supported for websockets; using %s" % clusters[0]['name']
                    self.post_error(RichStatus.fromError(errmsg))

            for c in clusters:
                # Mangle the name, if need be.
                c_name = c["name"]

                if c_name in mangled:
                    c["name"] = mangled[c_name]
                    # self.logger.info("%s: mangling cluster %s to %s" % (group_id, c_name, c["name"]))

                if c["weight"] is None:
                    unspecified += 1
                else:
                    total += c["weight"]

            if unspecified:
                for c in clusters:
                    if c["weight"] is None:
                        c["weight"] = (100.0 - total)/unspecified
            elif total != 100.0:
                for c in clusters:
                    c["weight"] *= 100.0/total

        # OK. When all is said and done, sort the list of routes by route weight...
        self.envoy_config['routes'] = sorted([
            route for group_id, route in self.envoy_routes.items()
        ], reverse=True, key=Mapping.route_weight)

        # ...then map clusters back into a list...
        self.envoy_config['clusters'] = [
            self.envoy_clusters[cluster_key] for cluster_key in sorted(self.envoy_clusters.keys())
        ]

        # ...and finally repeat for breakers and outliers, but copy them in the process so we
        # can mess with the originals.
        #
        # What's going on here is that circuit-breaker and outlier-detection configs aren't
        # included as independent objects in envoy.json, but we want to be able to discuss
        # them in diag. We also don't need to keep the _source and _referenced_by elements
        # in their real Envoy appearances.

        self.envoy_config['breakers'] = self.clean_and_copy(self.breakers)
        self.envoy_config['outliers'] = self.clean_and_copy(self.outliers)

    @staticmethod
    def tmod_certs_exist(tmod):
        """
        Returns the number of certs that are defined in the supplied tmod

        :param tmod: The TLS module configuration
        :return: number of certs in tmod
        :rtype: int
        """
        cert_count = 0
        if tmod.get('cert_chain_file') is not None:
            cert_count += 1
        if tmod.get('private_key_file') is not None:
            cert_count += 1
        if tmod.get('cacert_chain_file') is not None:
            cert_count += 1
        return cert_count

    def _get_intermediate_for(self, element_list, source_keys, value):
        if not isinstance(value, dict):
            return

        good = True

        if '_source' in value:
            good = False

            value_source = value.get("_source", None)
            value_referenced_by = value.get("_referenced_by", [])

            if ((value_source in source_keys) or
                (source_keys & set(value_referenced_by))):
                good = True

        if good:
            element_list.append(value)

    def get_intermediate_for(self, source_key):
        source_keys = []

        if source_key.startswith("grp-"):
            group_id = source_key[4:]

            for route in self.envoy_config['routes']:
                if route['_group_id'] == group_id:
                    source_keys.append(route['_source'])

                    for reference_key in route['_referenced_by']:
                        source_keys.append(reference_key)

            if not source_keys:
                return {
                    "error": "No group matches %s" % group_id
                }
        else:
            if source_key in self.source_map:
                # Exact match for a file in the source map: include all the objects
                # in the file.
                source_keys = self.source_map[source_key]
            elif source_key in self.sources:
                # Exact match for an object in a file: include only that object.
                source_keys.append(source_key)
            else:
                # No match at all. Weird.
                return {
                    "error": "No source matches %s" % source_key
                }

        source_keys = set(source_keys)

        # self.logger.debug("get_intermediate_for: source_keys %s" % source_keys)
        # self.logger.debug("get_intermediate_for: errors %s" % self.errors)

        sources = []

        for key in source_keys:
            source_dict = dict(self.sources[key])
            source_dict['errors'] = [
                {
                    'summary': error['error'].split('\n', 1)[0],
                    'text': error['error']
                }
                for error in self.errors.get(key, [])
            ]
            source_dict['source_key'] = key

            sources.append(source_dict)

        result = {
            "sources": sources
        }

        # self.logger.debug("get_intermediate_for: initial result %s" % result)

        for key in self.envoy_config.keys():
            result[key] = []

            value = self.envoy_config[key]

            if isinstance(value, list):
                for v2 in value:
                    self._get_intermediate_for(result[key], source_keys, v2)
            else:
                self._get_intermediate_for(result[key], source_keys, value)

        return result

    def generate_envoy_config(self, template=None, template_dir=None, **kwargs):
        # Finally! Render the template to JSON...
        envoy_json = self.to_json(template=template, template_dir=template_dir)

        # We used to use the JSON parser as a final sanity check here. That caused
        # Forge some issues, so it's turned off for now.

        # rc = RichStatus.fromError("impossible")

        # # ...and use the JSON parser as a final sanity check.
        # try:
        #     obj = json.loads(envoy_json)
        #     rc = RichStatus.OK(msg="Envoy configuration OK", envoy_config=obj)
        # except json.decoder.JSONDecodeError as e:
        #     rc = RichStatus.fromError("Invalid Envoy configuration: %s" % str(e),
        #                               raw=envoy_json, exception=e)

        # Go ahead and report that we generated an Envoy config, if we can.
        scout_result = Config.scout_report(action="config", result=True, generated=True, **kwargs)

        rc = RichStatus.OK(envoy_config=envoy_json, scout_result=scout_result)

        # self.logger.debug("Scout reports %s" % json.dumps(rc.scout_result))

        return rc

    def set_config_ambassador(self, module, key, value, merge=False):
        if not merge:
            self.ambassador_module[key] = value
        else:
            self.ambassador_module[key].update(value)

        # XXX This is actually wrong sometimes. If, for example, you have an
        # ambassador module that defines the admin_port, sure, bringing in its
        # source makes sense. On the other hand, if you have a TLS module 
        # created by a secret, that source shouldn't really take over the
        # admin document. This will take enough unraveling that I'm going to
        # leave it for now, though.
        self.ambassador_module['_source'] = module['_source']

    def update_config_ambassador(self, module, key, value):
        self.set_config_ambassador(module, key, value, merge=True)

    def tls_config_helper(self, name, amod, tmod):
        tmp_config = SourcedDict(_from=amod)
        some_enabled = False

        for context_name in tmod.keys():
            if context_name.startswith('_'):
                continue

            context = tmod[context_name]

            # self.logger.debug("context %s -- %s" % (context_name, json.dumps(context)))

            if context.get('enabled', True):
                if context_name == 'server':
                    # Server-side TLS is enabled.
                    self.logger.debug("TLS termination enabled!")
                    some_enabled = True

                    # Switch to port 443 by default...
                    self.set_config_ambassador(amod, 'service_port', 443)

                    # ...and merge in the server-side defaults.
                    tmp_config.update(self.default_tls_config['server'])
                    tmp_config.update(tmod['server'])

                    # Check if secrets are supplied for TLS termination and/or TLS auth
                    secret = context.get('secret')
                    if secret is not None:
                        self.logger.debug("config.server.secret is {}".format(secret))
                        # If /{etc,ambassador}/certs/tls.crt does not exist, then load the secrets
                        if check_cert_file(TLSPaths.mount_tls_crt.value):
                            self.logger.debug("Secret already exists, taking no action for secret {}".format(secret))
                        elif check_cert_file(TLSPaths.tls_crt.value):
                            tmp_config['cert_chain_file'] = TLSPaths.tls_crt.value
                            tmp_config['private_key_file'] = TLSPaths.tls_key.value
                        else:
                            (server_cert, server_key, server_data) = read_cert_secret(kube_v1(), secret, self.namespace)
                            if server_cert and server_key:
                                self.logger.debug("saving contents of secret {} to {}".format(
                                    secret, TLSPaths.cert_dir.value))
                                save_cert(server_cert, server_key, TLSPaths.cert_dir.value)
                                tmp_config['cert_chain_file'] = TLSPaths.tls_crt.value
                                tmp_config['private_key_file'] = TLSPaths.tls_key.value

                elif context_name == 'client':
                    # Client-side TLS is enabled.
                    self.logger.debug("TLS client certs enabled!")
                    some_enabled = True

                    # Merge in the client-side defaults.
                    tmp_config.update(self.default_tls_config['client'])
                    tmp_config.update(tmod['client'])

                    secret = context.get('secret')
                    if secret is not None:
                        self.logger.debug("config.client.secret is {}".format(secret))
                        if check_cert_file(TLSPaths.client_mount_crt.value):
                            self.logger.debug("Secret already exists, taking no action for secret {}".format(secret))
                        elif check_cert_file(TLSPaths.client_tls_crt.value):
                            tmp_config['cacert_chain_file'] = TLSPaths.client_tls_crt.value
                        else:
                            (client_cert, _, _) = read_cert_secret(kube_v1(), secret, self.namespace)
                            if client_cert:
                                self.logger.debug("saving contents of secret {} to  {}".format(
                                    secret, TLSPaths.client_cert_dir.value))
                                save_cert(client_cert, None, TLSPaths.client_cert_dir.value)
                                tmp_config['cacert_chain_file'] = TLSPaths.client_tls_crt.value

                else:
                    # This is a wholly new thing.
                    self.tls_contexts[context_name] = SourcedDict(
                        _from=tmod,
                        **context
                    )

        if some_enabled:
            if 'enabled' in tmp_config:
                del(tmp_config['enabled'])

            # Save the TLS config...
            self.set_config_ambassador(amod, 'tls_config', tmp_config)

        self.logger.debug("TLS config: %s" % json.dumps(self.ambassador_module['tls_config'], indent=4))
        self.logger.debug("TLS contexts: %s" % json.dumps(self.tls_contexts, indent=4))

        return some_enabled

    def module_config_ambassador(self, name, amod, tmod):
        # Toplevel Ambassador configuration. First up, check out TLS.

        have_amod_tls = False

        if amod and ('tls' in amod):
            have_amod_tls = self.tls_config_helper(name, amod, amod['tls'])

        if not have_amod_tls and tmod:
            self.tls_config_helper(name, tmod, tmod)

        if amod and ('cors' in amod):
            self.parse_and_save_default_cors(amod)
            
        # After that, check for port definitions, probes, etc., and copy them in
        # as we find them.
        for key in [ 'service_port', 'admin_port', 'diag_port',
                     'liveness_probe', 'readiness_probe', 'auth_enabled',
                     'use_proxy_proto', 'use_remote_address', 'diagnostics', 'x_forwarded_proto_redirect' ]:
            if amod and (key in amod):
                # Yes. It overrides the default.
                self.set_config_ambassador(amod, key, amod[key])

    def parse_and_save_default_cors(self, amod):
        cors_default_temp = {'enabled': True}
        cors = amod['cors']
        origins = cors.get('origins')
        if origins is not None:
            if type(origins) is list:
                cors_default_temp['allow_origin'] = origins
            elif type(origins) is str:
                cors_default_temp['allow_origin'] = origins.split(',')
            else:
                print("invalid cors configuration supplied - {}".format(origins))
                return

        self.save_cors_default_element("max_age", "max_age", cors_default_temp, cors)
        self.save_cors_default_element("credentials", "allow_credentials", cors_default_temp, cors)
        self.save_cors_default_element("methods", "allow_methods", cors_default_temp, cors)
        self.save_cors_default_element("headers", "allow_headers", cors_default_temp, cors)
        self.save_cors_default_element("exposed_headers", "expose_headers", cors_default_temp, cors)                           
        self.envoy_config['cors_default'] = cors_default_temp

    def save_cors_default_element(self, cors_key, route_key, cors_dest, cors_source):                    
        if cors_source.get(cors_key) is not None:
            if type(cors_source.get(cors_key)) is list:
                cors_dest[route_key] = ", ".join(cors_source.get(cors_key))
            else:
                cors_dest[route_key] = cors_source.get(cors_key)

    def module_config_ratelimit(self, ratelimit_config):
        cluster_hosts = None
        sources = []

        if ratelimit_config:
            for config in ratelimit_config.values():
                sources.append(config['_source'])
                cluster_hosts = config.get("service", None)

        if not cluster_hosts or not sources:
            return (None, None)

        host_rewrite = config.get("host_rewrite", None)

        cluster_name = "cluster_ext_ratelimit"
        filter_config = {
            "domain": "ambassador",
            "request_type": "both",
            "timeout_ms": 20
        }
        grpc_service = SourcedDict(
            name="rate_limit_service",
            cluster_name=cluster_name
        )

        first_source = sources.pop(0)

        filter = SourcedDict(
            _source=first_source,
            type="decoder",
            name="rate_limit",
            config=filter_config
        )

        if cluster_name not in self.envoy_clusters:
            (svc, url, originate_tls, otls_name) = self.service_tls_check(cluster_hosts, None, host_rewrite)
            self.add_intermediate_cluster(first_source, cluster_name,
                                          'extratelimit', [url],
                                          type="strict_dns", lb_type="round_robin",
                                          grpc=True, host_rewrite=host_rewrite)

        for source in sources:
            filter._mark_referenced_by(source)
            self.envoy_clusters[cluster_name]._mark_referenced_by(source)

        return (filter, grpc_service)

    def module_config_tracing(self, tracing_config):
        cluster_hosts = None
        driver = None
        driver_config = None
        tag_headers = None
        host_rewrite = None
        sources = []

        if tracing_config:
            for config in tracing_config.values():
                sources.append(config['_source'])
                cluster_hosts = config.get("service", None)
                driver = config.get("driver", None)
                driver_config = config.get("config", {})
                tag_headers = config.get("tag_headers", [])
                host_rewrite = config.get("host_rewrite", None)

        if not cluster_hosts or not sources:
            return

        cluster_name = "cluster_ext_tracing"

        first_source = sources.pop(0)

        if cluster_name not in self.envoy_clusters:
            (svc, url, originate_tls, otls_name) = self.service_tls_check(cluster_hosts, None, host_rewrite)
            grpc = False
            ssl_context = None
            if driver == "lightstep":
                grpc = True
                parsed_url = urlparse(url)
                ssl_context = {
                    "ca_cert_file": "/etc/ssl/certs/ca-certificates.crt",
                    "verify_subject_alt_name": [parsed_url.hostname]
                }
            self.add_intermediate_cluster(first_source, cluster_name,
                                          'exttracing', [url],
                                          type="strict_dns", lb_type="round_robin",
                                          host_rewrite=host_rewrite, grpc=grpc, ssl_context=ssl_context)

        driver_config['collector_cluster'] = cluster_name
        tracing = SourcedDict(
            _source=first_source,
            driver=driver,
            config=driver_config,
            tag_headers=tag_headers,
            cluster_name=cluster_name
        )
        self.envoy_config['tracing'] = tracing

    def auth_helper(self, sources, config, cluster_hosts, module):
        sources.append(module['_source'])

        for key in [ 'path_prefix', 'timeout_ms', 'cluster' ]:
            value = module.get(key, None)

            if value != None:
                previous = config.get(key, None)

                if previous and (previous != value):
                    errstr = (
                        "AuthService cannot support multiple %s values; using %s" %
                        (key, previous)
                    )

                    self.post_error(RichStatus.fromError(errstr), key=module['_source'])
                else:
                    config[key] = value

        headers = module.get('allowed_headers', None)

        if headers:
            allowed_headers = config.get('allowed_headers', [])

            for hdr in headers:
                if hdr not in allowed_headers:
                    allowed_headers.append(hdr)

            config['allowed_headers'] = allowed_headers

        auth_service = module.get("auth_service", None)
        # weight = module.get("weight", 100)
        weight = 100    # Can't support arbitrary weights right now.

        if auth_service:
            cluster_hosts[auth_service] = ( weight, module.get('tls', None) )

    def module_config_authentication(self, name, amod, auth_mod, auth_configs):
        filter_config = {
            "cluster": "cluster_ext_auth",
            "timeout_ms": 5000
        }

        cluster_hosts = {}
        sources = []

        if auth_mod:
            self.auth_helper(sources, filter_config, cluster_hosts, auth_mod)

        if auth_configs:
            # self.logger.debug("auth_configs: %s" % auth_configs)
            for config in auth_configs.values():
                self.auth_helper(sources, filter_config, cluster_hosts, config)

        if not sources:
            return None

        first_source = sources.pop(0)

        filter = SourcedDict(
            _source=first_source,
            _services=sorted(cluster_hosts.keys()),
            type="decoder",
            name="extauth",
            config=filter_config
        )

        cluster_name = filter_config['cluster']
        host_rewrite = filter_config.get('host_rewrite', None)

        if cluster_name not in self.envoy_clusters:
            if not cluster_hosts:
                cluster_hosts = { '127.0.0.1:5000': ( 100, None ) }

            urls = []
            protocols = {}

            for svc in sorted(cluster_hosts.keys()):
                weight, tls_context = cluster_hosts[svc]

                (svc, url, originate_tls, otls_name) = self.service_tls_check(svc, tls_context, host_rewrite)

                if originate_tls:
                    protocols['https'] = True
                else:
                    protocols['http'] = True

                if otls_name:
                    filter_config['cluster'] = cluster_name + "_" + otls_name
                    cluster_name = filter_config['cluster']

                urls.append(url)

            if len(protocols.keys()) != 1:
                raise Exception("auth config cannot try to use both HTTP and HTTPS")

            self.add_intermediate_cluster(first_source, cluster_name,
                                          'extauth', urls,
                                          type="strict_dns", lb_type="round_robin",
                                          originate_tls=originate_tls, host_rewrite=host_rewrite)

        for source in sources:
            filter._mark_referenced_by(source)
            self.envoy_clusters[cluster_name]._mark_referenced_by(source)

        return filter

    ### DIAGNOSTICS
    def diagnostic_overview(self):
        # Build a set of source _files_ rather than source _objects_.
        source_files = {}

        for filename, source_keys in self.source_map.items():
            # self.logger.debug("overview -- filename %s, source_keys %d" %
            #                   (filename, len(source_keys)))

            # # Skip '--internal--' etc.
            # if filename.startswith('--'):
            #     continue

            source_dict = source_files.setdefault(
                filename,
                {
                    'filename': filename,
                    'objects': {},
                    'count': 0,
                    'plural': "objects",
                    'error_count': 0,
                    'error_plural': "errors"
                }
            )

            for source_key in source_keys:
                # self.logger.debug("overview --- source_key %s" % source_key)

                source = self.sources[source_key]

                if ('source' in source) and not ('source' in source_dict):
                    source_dict['source'] = source['source']

                raw_errors = self.errors.get(source_key, [])

                errors = []

                for error in raw_errors:
                    source_dict['error_count'] += 1

                    errors.append({
                        'summary': error['error'].split('\n', 1)[0],
                        'text': error['error']
                    })

                source_dict['error_plural'] = "error" if (source_dict['error_count'] == 1) else "errors"

                source_dict['count'] += 1
                source_dict['plural'] = "object" if (source_dict['count'] == 1) else "objects"

                object_dict = source_dict['objects']
                object_dict[source_key] = {
                    'key': source_key,
                    'kind': source['kind'],
                    'errors': errors
                }

        routes = []

        for route in self.envoy_config['routes']:
            if route['_source'] != "--diagnostics--":
                route['_group_id'] = Mapping.group_id(route.get('method', 'GET'),
                                                      route['prefix'] if 'prefix' in route else route['regex'],
                                                      route.get('headers', []))

                routes.append(route)

        configuration = { key: self.envoy_config[key] for key in self.envoy_config.keys()
                          if key != "routes" }

        cluster_to_service_mapping = {
            "cluster_ext_auth": "AuthService",
            "cluster_ext_tracing": "TracingService",
            "cluster_ext_ratelimit": "RateLimitService"
        }
        ambassador_services = []
        for cluster in configuration.get('clusters', []):
            maps_to_service = cluster_to_service_mapping.get(cluster['name'])
            if maps_to_service:
                service_weigth = 100.0 / len(cluster['urls'])
                for url in cluster['urls']:
                    ambassador_services.append(SourcedDict(
                        _from=cluster,
                        type=maps_to_service,
                        name=url,
                        cluster=cluster['name'],
                        _service_weight=service_weigth
                    ))

        overview = dict(sources=sorted(source_files.values(), key=lambda x: x['filename']),
                        routes=routes,
                        **configuration)

        if len(ambassador_services) > 0:
            overview['ambassador_services'] = ambassador_services

        # self.logger.debug("overview result %s" % json.dumps(overview, indent=4, sort_keys=True))

        return overview

    def pretty(self, obj, out=sys.stdout):
        out.write(obj)
#        json.dump(obj, out, indent=4, separators=(',',':'), sort_keys=True)
#        out.write("\n")

    def to_json(self, template=None, template_dir=None):
        template_paths = [ self.config_dir_path, self.template_dir_path ]

        if template_dir:
            template_paths.insert(0, template_dir)

        if not template:
            env = Environment(loader=FileSystemLoader(template_paths))
            template = env.get_template("envoy.j2")

        return(template.render(**self.envoy_config))

    def dump(self):
        print("==== config")
        self.pretty(self.config)

        print("==== envoy_config")
        self.pretty(self.envoy_config)

if __name__ == '__main__':
    aconf = Config(sys.argv[1])
    print(json.dumps(aconf.diagnostic_overview(), indent=4, sort_keys=True))
