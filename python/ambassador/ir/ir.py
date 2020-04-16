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
from typing import Any, Callable, Dict, Iterable, List, Optional, Type, Union, ValuesView
from typing import cast as typecast

import datetime
import json
import logging
import os

from ipaddress import ip_address

from ..constants import Constants

from ..utils import RichStatus, SavedSecret, SecretHandler, SecretInfo
from ..config import Config
from ..config import ACResource
from ..config import ResourceFetcher

from .irresource import IRResource
from .irambassador import IRAmbassador
from .irauth import IRAuth
from .irfilter import IRFilter
from .ircluster import IRCluster
from .irbasemappinggroup import IRBaseMappingGroup
from .irbasemapping import IRBaseMapping
from .irhttpmapping import IRHTTPMapping
from .irhost import IRHost, HostFactory
from .irmappingfactory import MappingFactory
from .irratelimit import IRRateLimit
from .irtls import TLSModuleFactory, IRAmbassadorTLS
from .irlistener import ListenerFactory, IRListener
from .irlogservice import IRLogService, IRLogServiceFactory
from .irtracing import IRTracing
from .irtlscontext import IRTLSContext, TLSContextFactory
from .irserviceresolver import IRServiceResolver, IRServiceResolverFactory, SvcEndpoint, SvcEndpointSet

from ..VERSION import Version, Build

#############################################################################
## ir.py -- the Ambassador Intermediate Representation (IR)
##
## After getting an ambassador.Config, you can create an ambassador.IR. The
## IR is the basis for everything else: you can use it to configure an Envoy
## or to run diagnostics.


class IR:
    ambassador_module: IRAmbassador
    ambassador_id: str
    ambassador_namespace: str
    ambassador_nodename: str
    aconf: Config
    clusters: Dict[str, IRCluster]
    agent_active: bool
    agent_service: Optional[str]
    agent_origination_ctx: Optional[IRTLSContext]
    edge_stack_allowed: bool
    file_checker: Callable[[str], bool]
    filters: List[IRFilter]
    groups: Dict[str, IRBaseMappingGroup]
    grpc_services: Dict[str, IRCluster]
    hosts: Dict[str, IRHost]
    listeners: List[IRListener]
    log_services: Dict[str, IRLogService]
    ratelimit: Optional[IRRateLimit]
    redirect_cleartext_from: Optional[int]
    resolvers: Dict[str, IRServiceResolver]
    router_config: Dict[str, Any]
    saved_resources: Dict[str, IRResource]
    saved_secrets: Dict[str, SavedSecret]
    secret_handler: SecretHandler
    secret_root: str
    sidecar_cluster_name: Optional[str]
    tls_contexts: Dict[str, IRTLSContext]
    tls_module: Optional[IRAmbassadorTLS]
    tracing: Optional[IRTracing]

    def __init__(self, aconf: Config, secret_handler=None, file_checker=None, logger=None, watch_only=False) -> None:
        self.ambassador_id = Config.ambassador_id
        self.ambassador_namespace = Config.ambassador_namespace
        self.ambassador_nodename = aconf.ambassador_nodename
        self.statsd = aconf.statsd

        self.logger = logger or logging.getLogger("ambassador.ir")

        # We're using setattr since since mypy complains about assigning directly to a method.
        secret_root = os.environ.get('AMBASSADOR_CONFIG_BASE_DIR', "/ambassador")
        setattr(self, 'file_checker', file_checker if file_checker is not None else os.path.isfile)

        # The secret_handler is _required_.
        self.secret_handler = secret_handler

        assert self.secret_handler, "Ambassador.IR requires a SecretHandler at initialization"

        self.logger.debug("IR __init__:")
        self.logger.debug("IR: Version         %s built from %s on %s" % (Version, Build.git.commit, Build.git.branch))
        self.logger.debug("IR: AMBASSADOR_ID   %s" % self.ambassador_id)
        self.logger.debug("IR: Namespace       %s" % self.ambassador_namespace)
        self.logger.debug("IR: Nodename        %s" % self.ambassador_nodename)
        self.logger.debug("IR: Endpoints       %s" % "enabled" if Config.enable_endpoints else "disabled")

        self.logger.debug("IR: file checker:   %s" % getattr(self, 'file_checker').__name__)
        self.logger.debug("IR: secret handler: %s" % type(self.secret_handler).__name__)

        # First up: save the Config object. Its source map may be necessary later.
        self.aconf = aconf

        # Next, we'll want a way to keep track of resources we end up working
        # with. It starts out empty.
        self.saved_resources = {}

        # Also, we have no saved secret stuff yet...
        self.saved_secrets = {}
        self.secret_info: Dict[str, SecretInfo] = {}

        # ...and the initial IR state is empty _except for k8s_status_updates_.
        #
        # Note that we use a map for clusters, not a list -- the reason is that
        # multiple mappings can use the same service, and we don't want multiple
        # clusters.

        self.breakers = {}
        self.clusters = {}
        self.filters = []
        self.groups = {}
        self.grpc_services = {}
        self.hosts = {}
        # self.k8s_status_updates is handled below.
        self.listeners = []
        self.log_services = {}
        self.outliers = {}
        self.ratelimit = None
        self.redirect_cleartext_from = None
        self.resolvers = {}
        self.saved_secrets = {}
        self.secret_info = {}
        self.services = {}
        self.sidecar_cluster_name = None
        self.tls_contexts = {}
        self.tls_module = None
        self.tracing = None

        # Copy k8s_status_updates from our aconf.
        self.k8s_status_updates = aconf.k8s_status_updates

        # Check on the intercept agent and edge stack. Note that the Edge Stack touchfile is _not_
        # within $AMBASSADOR_CONFIG_BASE_DIR: it stays in /ambassador no matter what.

        self.agent_active = (os.environ.get("AGENT_SERVICE", None) != None)
        self.edge_stack_allowed = os.path.exists('/ambassador/.edge_stack')
        self.agent_origination_ctx = None

        # OK, time to get this show on the road. First things first: set up the
        # Ambassador module.
        #
        # The Ambassador module is special: it doesn't do anything in its setup() method, but
        # instead defers all its heavy lifting to its finalize() method. Why? Because we need
        # to create the Ambassador module very early to allow IRResource.lookup() to work, but
        # we need to go pull in secrets and such before we can get all the Ambassador-module
        # stuff fully set up.
        #
        # So. First, create the module.
        self.ambassador_module = typecast(IRAmbassador, self.save_resource(IRAmbassador(self, aconf)))

        # Next, grab whatever information our aconf has about secrets...
        self.save_secret_info(aconf)

        # ...and then it's on to default TLS stuff, both from the TLS module and from
        # any TLS contexts.
        #
        # XXX This feels like a hack -- shouldn't it be class-wide initialization
        # in TLSModule or TLSContext? So far it's the only place we need anything like
        # this though.

        TLSModuleFactory.load_all(self, aconf)
        TLSContextFactory.load_all(self, aconf)

        # ...then grab whatever we know about Hosts...
        HostFactory.load_all(self, aconf)

        # ...then set up for the intercept agent, if that's a thing.
        self.agent_init(aconf)

        # Finally, finalize all the Host stuff (including the !*@#&!* fallback context).
        HostFactory.finalize(self, aconf)

        # Now we can finalize the Ambassador module, to tidy up secrets et al. We do this
        # here so that secrets and TLS contexts are available.
        if not self.ambassador_module.finalize(self, aconf):
            # Uhoh.
            self.ambassador_module.set_active(False)    # This can't be good.

        _activity_str = 'watching' if watch_only else 'starting'
        _mode_str = 'OSS'

        if self.agent_active:
            _mode_str = 'Intercept Agent'
        elif self.edge_stack_allowed:
            _mode_str = 'Edge Stack'

        self.logger.info(f"IR: {_activity_str} {_mode_str}")

        # Next up, initialize our IRServiceResolvers...
        IRServiceResolverFactory.load_all(self, aconf)

        # ...and then we can finalize the agent, if that's a thing.
        self.agent_finalize(aconf)

        # Once here, if we're only watching, we're done.
        if watch_only:
            return

        # REMEMBER FOR SAVING YOU NEED TO CALL save_resource!
        # THIS IS VERY IMPORTANT!

        # Save circuit breakers, outliers, and services.
        self.breakers = aconf.get_config("CircuitBreaker") or {}
        self.outliers = aconf.get_config("OutlierDetection") or {}
        self.services = aconf.get_config("service") or {}

        self.cluster_ingresses_to_mappings()

        # Save tracing, ratelimit, and logging settings.
        self.tracing = typecast(IRTracing, self.save_resource(IRTracing(self, aconf)))
        self.ratelimit = typecast(IRRateLimit, self.save_resource(IRRateLimit(self, aconf)))
        IRLogServiceFactory.load_all(self, aconf)

        # After the Ambassador and TLS modules are done, we need to set up the
        # filter chains. Note that order of the filters matters. Start with auth,
        # since it needs to be able to override everything...
        self.save_filter(IRAuth(self, aconf))

        # ...then deal with the non-configurable cors filter...
        self.save_filter(IRFilter(ir=self, aconf=aconf,
                                  rkey="ir.cors", kind="ir.cors", name="cors",
                                  config={}))

        # ...then the ratelimit filter...
        if self.ratelimit:
            self.save_filter(self.ratelimit, already_saved=True)

        # ...and, finally, the barely-configurable router filter.
        router_config = {}

        if self.tracing:
            router_config['start_child_span'] = True

        self.save_filter(IRFilter(ir=self, aconf=aconf,
                                  rkey="ir.router", kind="ir.router", name="router", type="decoder",
                                  config=router_config))

        # We would handle other modules here -- but guess what? There aren't any.
        # At this point ambassador, tls, and the deprecated auth module are all there
        # are, and they're handled above. So. At this point go sort out all the Mappings
        ListenerFactory.load_all(self, aconf)
        MappingFactory.load_all(self, aconf)

        self.walk_saved_resources(aconf, 'add_mappings')

        TLSModuleFactory.finalize(self, aconf)
        ListenerFactory.finalize(self, aconf)
        MappingFactory.finalize(self, aconf)

        # At this point we should know the full set of clusters, so we can normalize
        # any long cluster names.
        collisions: Dict[str, List[str]] = {}

        for name in sorted(self.clusters.keys()):
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
                self.clusters[name]['name'] = mangled_name

        # After we have the cluster names fixed up, go finalize filters.
        if self.tracing:
            self.tracing.finalize()

        if self.ratelimit:
            self.ratelimit.finalize()

        for filter in self.filters:
            filter.finalize()

    # XXX Brutal hackery here! Probably this is a clue that Config and IR and such should have
    # a common container that can hold errors.
    def post_error(self, rc: Union[str, RichStatus], resource: Optional[IRResource]=None, rkey: Optional[str]=None):
        self.aconf.post_error(rc, resource=resource, rkey=rkey)

    def agent_init(self, aconf: Config) -> None:
        """
        Initialize as the Intercept Agent, if we're doing that.

        THIS WHOLE METHOD NEEDS TO GO AWAY: instead, just configure the agent with CRDs as usual.
        However, that's just too painful to contemplate without `edgectl inject-agent`.

        :param aconf: Config to work with
        :return: None
        """

        # Intercept stuff is an Edge Stack thing.
        if not (self.edge_stack_allowed and self.agent_active):
            self.logger.info("Intercept agent not active, skipping initialization")
            return

        self.agent_service = os.environ.get("AGENT_SERVICE", None)

        if self.agent_service is None:
            # This is technically impossible, but whatever.
            self.logger.info("Intercept agent active but no AGENT_SERVICE? skipping initialization")
            self.agent_active = False
            return

        self.logger.info(f"Intercept agent active for {self.agent_service}, initializing")

        # We're going to either create a Host to terminate TLS, or to do cleartext. In neither
        # case will we do ACME.
        host_args = {
            "hostname": "*",
            "selector": {
                "matchLabels": {
                    "intercept": self.agent_service
                }
            },
            "acmeProvider": {
                "authority": "none"
            }
        }

        # Have they asked us to do TLS?
        agent_termination_secret = os.environ.get("AGENT_TLS_TERM_SECRET", None)

        if agent_termination_secret is not None:
            # Yup.
            host_args["tlsSecret"] = { "name": agent_termination_secret }
        else:
            # No termination secret, so do cleartext.
            host_args["requestPolicy"] = {
                "insecure": {
                    "action": "Route"
                }
            }

        host = IRHost(self, aconf, rkey=self.ambassador_module.rkey, location=self.ambassador_module.location,
                      name="agent-host",
                      **host_args)

        if host.is_active():
            host.referenced_by(self.ambassador_module)
            host.sourced_by(self.ambassador_module)

            self.logger.info(f"Intercept agent: saving host {host.pretty()}")
            self.logger.info(host.as_json())
            self.save_host(host)
        else:
            self.logger.info(f"Intercept agent: not saving inactive host {host.pretty()}")

        # How about originating TLS?
        agent_origination_secret = os.environ.get("AGENT_TLS_ORIG_SECRET", None)

        if agent_origination_secret is not None:
            # Uhhhh. Synthesize a TLSContext for this, I guess.
            #
            # XXX What if they already have a context with this name?
            ctx = IRTLSContext(self, aconf, rkey=self.ambassador_module.rkey, location=self.ambassador_module.location,
                               name="agent-origination-context",
                               secret=agent_origination_secret)

            ctx.referenced_by(self.ambassador_module)
            self.save_tls_context(ctx)

            self.logger.info(f"Intercept agent: saving origination TLSContext {ctx.name}")
            self.logger.info(ctx.as_json())

            self.agent_origination_ctx = ctx

    def agent_finalize(self, aconf) -> None:
        if not (self.edge_stack_allowed and self.agent_active):
            self.logger.info(f"Intercept agent not active, skipping finalization")
            return

        self.logger.info(f"Intercept agent active for {self.agent_service}, finalizing")

        agent_port_str = os.environ.get("AGENT_PORT", None)

        if agent_port_str is None:
            self.post_error("Intercept agent requires both AGENT_SERVICE and AGENT_PORT to be set")
            self.agent_active = False
            return

        agent_port = -1

        try:
            agent_port = int(agent_port_str)
        except:
            self.post_error(f"Intercept agent port {agent_port_str} is not valid")
            self.agent_active = False
            return

        self.logger.info(f"Intercept agent active for {self.agent_service}:{agent_port}, adding fallback mapping")

        # XXX OMG this is a crock. Don't use precedence -1000000 for this, because otherwise Edge
        # Stack might decide it's the Edge Policy Console fallback mapping and force it to be
        # routed insecure. !*@&#*!@&#* We need per-mapping security settings.
        #
        # XXX What if they already have a mapping with this name?

        ctx_name = None

        if self.agent_origination_ctx:
            ctx_name = self.agent_origination_ctx.name

        mapping = IRHTTPMapping(self, aconf, rkey=self.ambassador_module.rkey, location=self.ambassador_module.location,
                                name="agent-fallback-mapping",
                                metadata_labels={"ambassador_diag_class": "private"},
                                prefix="/",
                                rewrite="/",
                                service=f"127.0.0.1:{agent_port}",
                                tls=ctx_name,
                                precedence=-999999) # No, really. See comment above.

        mapping.referenced_by(self.ambassador_module)
        self.add_mapping(aconf, mapping)

    def save_resource(self, resource: IRResource) -> IRResource:
        if resource.is_active():
            self.saved_resources[resource.rkey] = resource

        return resource

    def save_host(self, host: IRHost) -> None:
        extant_host = self.hosts.get(host.name, None)
        is_valid = True

        if extant_host:
            self.post_error("Duplicate Host %s; keeping definition from %s" % (host.name, extant_host.location))
            is_valid = False

        if is_valid:
            self.hosts[host.name] = host

    # Get saved hosts.
    def get_hosts(self) -> List[IRHost]:
        return list(self.hosts.values())

    # Save secrets from our aconf.
    def save_secret_info(self, aconf):
        aconf_secrets = aconf.get_config("secrets") or {}
        self.logger.debug(f"IR: aconf has secrets: {aconf_secrets.keys()}")

        for secret_key, aconf_secret in aconf_secrets.items():
            # Ignore anything that doesn't at least have a public half.
            if aconf_secret.get('tls_crt') or aconf_secret.get('cert-chain_pem'):
                secret_info = SecretInfo.from_aconf_secret(aconf_secret)
                secret_name = secret_info.name
                secret_namespace = secret_info.namespace

                self.logger.debug(f'saving {secret_name}.{secret_namespace} (from {secret_key}) in secret_info')
                self.secret_info[f'{secret_name}.{secret_namespace}'] = secret_info

    def save_tls_context(self, ctx: IRTLSContext) -> None:
        extant_ctx = self.tls_contexts.get(ctx.name, None)
        is_valid = True

        if extant_ctx:
            self.post_error("Duplicate TLSContext %s; keeping definition from %s" % (ctx.name, extant_ctx.location))
            is_valid = False

        if ctx.get('redirect_cleartext_from', None) is not None:
            if self.redirect_cleartext_from is None:
                self.redirect_cleartext_from = ctx.redirect_cleartext_from
            else:
                if self.redirect_cleartext_from != ctx.redirect_cleartext_from:
                    self.post_error("TLSContext: %s; configured conflicting redirect_from port: %s" % (ctx.name, ctx.redirect_cleartext_from))
                    is_valid = False

        if is_valid:
            self.tls_contexts[ctx.name] = ctx

    def get_resolver(self, name: str) -> Optional[IRServiceResolver]:
        return self.resolvers.get(name, None)

    def add_resolver(self, resolver: IRServiceResolver) -> None:
        self.resolvers[resolver.name] = resolver

    def has_tls_context(self, name: str) -> bool:
        return bool(self.get_tls_context(name))

    def get_tls_context(self, name: str) -> Optional[IRTLSContext]:
        return self.tls_contexts.get(name, None)

    def get_tls_contexts(self) -> ValuesView[IRTLSContext]:
        return self.tls_contexts.values()

    def resolve_secret(self, resource: IRResource, secret_name: str, namespace: str):
        # OK. Do we already have a SavedSecret for this?
        ss_key = f'{secret_name}.{namespace}'

        ss = self.saved_secrets.get(ss_key, None)

        if ss:
            # Done. Return it.
            self.logger.debug(f"resolve_secret {ss_key}: using cached SavedSecret")
            self.secret_handler.still_needed(resource, secret_name, namespace)
            return ss

        # OK, do we have a secret_info for it??
        # self.logger.debug(f"resolve_secret {ss_key}: checking secret_info")

        secret_info = self.secret_info.get(ss_key, None)

        if secret_info:
            self.logger.debug(f"resolve_secret {ss_key}: found secret_info")
            self.secret_handler.still_needed(resource, secret_name, namespace)
        else:
            # No secret_info, so ask the secret_handler to find us one.
            self.logger.debug(f"resolve_secret {ss_key}: no secret_info, asking handler to load")
            secret_info = self.secret_handler.load_secret(resource, secret_name, namespace)

        if not secret_info:
            self.logger.error(f"Secret {ss_key} unknown")

            ss = SavedSecret(secret_name, namespace, None, None, None, None, None)
        else:
            self.logger.debug(f"resolve_secret {ss_key}: found secret, asking handler to cache")

            # OK, we got a secret_info. Cache that using the secret handler.
            ss = self.secret_handler.cache_secret(resource, secret_info)

            # Save this for next time.
            self.saved_secrets[secret_name] = ss
        return ss

    def resolve_targets(self, cluster: IRCluster, resolver_name: Optional[str],
                        hostname: str, namespace: str, port: int) -> Optional[SvcEndpointSet]:
        # Is the host already an IP address?
        is_ip_address = False

        try:
            x = ip_address(hostname)
            is_ip_address = True
        except ValueError:
            pass

        if is_ip_address:
            # Already an IP address, great.
            self.logger.debug(f'cluster {cluster.name}: {hostname} is already an IP address')

            return [
                {
                    'ip': hostname,
                    'port': port,
                    'target_kind': 'IPaddr'
                }
            ]

        # Which resolver should we use?
        if not resolver_name:
            resolver_name = self.ambassador_module.get('resolver', 'kubernetes-service')

        # Casting to str is OK because the Ambassador module's resolver must be a string,
        # so all the paths for resolver_name land with it being a string.
        resolver = self.get_resolver(typecast(str, resolver_name))

        # It should not be possible for resolver to be unset here.
        if not resolver:
            self.post_error(f"cluster {cluster.name} has invalid resolver {resolver_name}?", rkey=cluster.rkey)
            return None

        # OK, ask the resolver for the target list. Understanding the mechanics of resolution
        # and the load balancer policy and all that is up to the resolver.
        return resolver.resolve(self, cluster, hostname, namespace, port)

    def save_filter(self, resource: IRFilter, already_saved=False) -> None:
        if resource.is_active():
            if not already_saved:
                resource = typecast(IRFilter, self.save_resource(resource))

            self.filters.append(resource)

    def walk_saved_resources(self, aconf, method_name):
        for res in self.saved_resources.values():
            getattr(res, method_name)(self, aconf)

    def add_listener(self, listener: IRListener) -> None:
        self.listeners.append(listener)

    def add_mapping(self, aconf: Config, mapping: IRBaseMapping) -> Optional[IRBaseMappingGroup]:
        mapping.check_status()

        if mapping.is_active():
            if mapping.group_id not in self.groups:
                group_name = "GROUP: %s" % mapping.name
                group_class = mapping.group_class()
                group = group_class(ir=self, aconf=aconf,
                                    location=mapping.location,
                                    name=group_name,
                                    mapping=mapping)

                self.groups[group.group_id] = group
            else:
                group = self.groups[mapping.group_id]
                group.add_mapping(aconf, mapping)

            return group
        else:
            return None

    def add_mapping_to_config(self, mapping: ACResource, mapping_identifier: str):
        if 'mappings' not in self.aconf.config:
            self.aconf.config['mappings'] = {}

        self.aconf.config['mappings'][mapping_identifier] = mapping

    def cluster_ingresses_to_mappings(self):
        cluster_ingresses = self.aconf.get_config("ClusterIngress")
        knative_ingresses = self.aconf.get_config("KnativeIngress")

        final_knative_ingresses = {}
        if cluster_ingresses is not None:
            final_knative_ingresses.update(cluster_ingresses)
        if knative_ingresses is not None:
            final_knative_ingresses.update(knative_ingresses)

        for ci_name, ci in final_knative_ingresses.items():
            kind = ci['kind']
            current_generation = ci['generation']

            if kind == 'KnativeIngress':
                kind = 'ingress.networking.internal.knative.dev'
            else:
                kind = kind.lower() + ".networking.internal.knative.dev"

            self.logger.debug(f"Parsing {kind} {ci_name}")

            ci_rules = ci.get('rules', [])
            for rule_count, ci_rule in enumerate(ci_rules):
                ci_hosts = ci_rule.get('hosts', [])

                mapping_host = ""
                for ci_host in ci_hosts:
                    if mapping_host == "":
                        mapping_host = ci_host
                    else:
                        mapping_host += "|" + ci_host

                ci_http = ci_rule.get('http', None)
                if ci_http is not None:
                    ci_paths = ci_http.get('paths', [])
                    for path_count, ci_path in enumerate(ci_paths):
                        ci_headers = ci_path.get('appendHeaders', {})

                        ci_splits = ci_path.get('splits', [])
                        for split_count, ci_split in enumerate(ci_splits):
                            unique_suffix = f"{rule_count}-{path_count}"
                            mapping_identifier = ci_name + '-' + unique_suffix
                            ci_mapping = {
                                'rkey': mapping_identifier,
                                'location': mapping_identifier,
                                'apiVersion': 'getambassador.io/v2',
                                'kind': 'Mapping',
                                'name': mapping_identifier,
                                'prefix': '/'
                            }

                            if mapping_host != "":
                                ci_mapping['host'] = f"^({mapping_host})$"
                                ci_mapping['host_regex'] = True

                            split_headers = ci_split.get('appendHeaders', {})
                            final_headers = {**ci_headers, **split_headers}

                            if len(final_headers) > 0:
                                ci_mapping['add_request_headers'] = final_headers

                            ci_percent = ci_split.get('percent', None)
                            if ci_percent is not None:
                                ci_mapping['weight'] = ci_percent

                            ci_service = ci_split.get('serviceName', None)
                            if ci_service is None:
                                continue

                            ci_namespace = ci_split.get('serviceNamespace', 'default')
                            ci_port = ci_split.get('servicePort', 80)

                            ci_mapping['namespace'] = ci_namespace
                            ci_mapping['service'] = f"{ci_service}.{ci_namespace}:{ci_port}"

                            self.logger.debug(f"Generated mapping from ClusterIngress: {ci_mapping}")
                            self.add_mapping_to_config(mapping=ci_mapping, mapping_identifier=mapping_identifier)

                            # Remember that we need to update status on this resource.
                            utcnow = datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
                            status_update = (kind, ci_namespace, {
                                "observedGeneration": current_generation,
                                "conditions": [
                                    {
                                        "lastTransitionTime": utcnow,
                                        "status": "True",
                                        "type": "LoadBalancerReady"
                                    },
                                    {
                                        "lastTransitionTime": utcnow,
                                        "status": "True",
                                        "type": "NetworkConfigured"
                                    },
                                    {
                                        "lastTransitionTime": utcnow,
                                        "status": "True",
                                        "type": "Ready"
                                    }
                                ]
                            })

                            self.logger.debug(f"Updating {ci_name}'s status to: {status_update}")
                            self.k8s_status_updates[ci_name] = status_update

    def ordered_groups(self) -> Iterable[IRBaseMappingGroup]:
        return reversed(sorted(self.groups.values(), key=lambda x: x['group_weight']))

    def has_cluster(self, name: str) -> bool:
        return name in self.clusters

    def get_cluster(self, name: str) -> Optional[IRCluster]:
        return self.clusters.get(name, None)

    def add_cluster(self, cluster: IRCluster) -> IRCluster:
        if not self.has_cluster(cluster.name):
            self.clusters[cluster.name] = cluster

            if cluster.is_edge_stack_sidecar():
                self.logger.info(f"IR: cluster {cluster.name} is the sidecar")
                self.sidecar_cluster_name = cluster.name

        return self.clusters[cluster.name]

    def merge_cluster(self, cluster: IRCluster) -> bool:
        extant = self.get_cluster(cluster.name)

        if extant:
            return extant.merge(cluster)
        else:
            self.add_cluster(cluster)
            return True

    def has_grpc_service(self, name: str) -> bool:
        return name in self.grpc_services

    def add_grpc_service(self, name: str, cluster: IRCluster) -> IRCluster:
        if not self.has_grpc_service(name):
            if not self.has_cluster(cluster.name):
                self.clusters[cluster.name] = cluster

            self.grpc_services[name] = cluster

        return self.grpc_services[name]

    def as_dict(self) -> Dict[str, Any]:
        od = {
            'identity': {
                'ambassador_id': self.ambassador_id,
                'ambassador_namespace': self.ambassador_namespace,
                'ambassador_nodename': self.ambassador_nodename,
            },
            'ambassador': self.ambassador_module.as_dict(),
            'clusters': { cluster_name: cluster.as_dict()
                          for cluster_name, cluster in self.clusters.items() },
            'grpc_services': { svc_name: cluster.as_dict()
                               for svc_name, cluster in self.grpc_services.items() },
            'hosts': [ host.as_dict() for host in self.hosts.values() ],
            'listeners': [ listener.as_dict() for listener in self.listeners ],
            'filters': [ filt.as_dict() for filt in self.filters ],
            'groups': [ group.as_dict() for group in self.ordered_groups() ],
            'tls_contexts': [ context.as_dict() for context in self.tls_contexts.values() ],
            'services': self.services,
            'k8s_status_updates': self.k8s_status_updates
        }

        if self.log_services:
            od['log_services'] = [ srv.as_dict() for srv in self.log_services.values() ]

        if self.tracing:
            od['tracing'] = self.tracing.as_dict()

        if self.ratelimit:
            od['ratelimit'] = self.ratelimit.as_dict()

        return od

    def as_json(self) -> str:
        return json.dumps(self.as_dict(), sort_keys=True, indent=4)

    def features(self) -> Dict[str, Any]:
        od: Dict[str, Union[bool, int, Optional[str]]] = {}

        if self.aconf.helm_chart:
            od['helm_chart'] = self.aconf.helm_chart
        od['managed_by'] = self.aconf.pod_labels.get('app.kubernetes.io/managed-by', '')

        tls_termination_count = 0   # TLS termination contexts
        tls_origination_count = 0   # TLS origination contexts

        using_tls_module = False
        using_tls_contexts = False

        for ctx in self.get_tls_contexts():
            if ctx:
                secret_info = ctx.get('secret_info', {})

                if secret_info:
                    using_tls_contexts = True

                    if secret_info.get('certificate_chain_file', None):
                        tls_termination_count += 1

                    if secret_info.get('cacert_chain_file', None):
                        tls_origination_count += 1

                if ctx.get('_legacy', False):
                    using_tls_module = True

        od['tls_using_module'] = using_tls_module
        od['tls_using_contexts'] = using_tls_contexts
        od['tls_termination_count'] = tls_termination_count
        od['tls_origination_count'] = tls_origination_count

        for key in [ 'diagnostics', 'liveness_probe', 'readiness_probe', 'statsd' ]:
            od[key] = self.ambassador_module.get(key, {}).get('enabled', False)

        for key in [ 'use_proxy_proto', 'use_remote_address', 'x_forwarded_proto_redirect', 'enable_http10',
                     'add_linkerd_headers', 'use_ambassador_namespace_for_service_resolution', 'proper_case' ]:
            od[key] = self.ambassador_module.get(key, False)

        od['service_resource_total'] = len(list(self.services.keys()))

        od['xff_num_trusted_hops'] = self.ambassador_module.get('xff_num_trusted_hops', 0)

        od['listener_idle_timeout_ms'] = self.ambassador_module.get('listener_idle_timeout_ms', None)

        od['server_name'] = bool(self.ambassador_module.server_name != 'envoy')

        od['custom_ambassador_id'] = bool(self.ambassador_id != 'default')

        default_port = Constants.SERVICE_PORT_HTTPS if tls_termination_count else Constants.SERVICE_PORT_HTTP

        od['custom_listener_port'] = bool(self.ambassador_module.service_port != default_port)
        od['custom_diag_port'] = bool(self.ambassador_module.diag_port != Constants.DIAG_PORT)

        cluster_count = 0
        cluster_grpc_count = 0      # clusters using GRPC upstream
        cluster_http_count = 0      # clusters using HTTP or HTTPS upstream
        cluster_tls_count = 0       # clusters using TLS origination

        cluster_routing_kube_count = 0          # clusters routing using kube
        cluster_routing_envoy_rr_count = 0      # clusters routing using envoy round robin
        cluster_routing_envoy_rh_count = 0      # clusters routing using envoy ring hash
        cluster_routing_envoy_maglev_count = 0  # clusters routing using envoy maglev
        cluster_routing_envoy_lr_count = 0      # clusters routing using envoy least request

        endpoint_grpc_count = 0     # endpoints using GRPC upstream
        endpoint_http_count = 0     # endpoints using HTTP/HTTPS upstream
        endpoint_tls_count = 0      # endpoints using TLS origination

        endpoint_routing_kube_count = 0         # endpoints Kube is routing to
        endpoint_routing_envoy_rr_count = 0     # endpoints Envoy round robin is routing to
        endpoint_routing_envoy_rh_count = 0     # endpoints Envoy ring hash is routing to
        endpoint_routing_envoy_maglev_count = 0  # endpoints Envoy maglev is routing to
        endpoint_routing_envoy_lr_count = 0     # endpoints Envoy least request is routing to

        for cluster in self.clusters.values():
            cluster_count += 1
            using_tls = False
            using_http = False
            using_grpc = False

            lb_type = 'kube'

            if cluster.get('enable_endpoints', False):
                lb_type = cluster.get('lb_type', 'round_robin')

            if lb_type == 'kube':
                cluster_routing_kube_count += 1
            elif lb_type == 'ring_hash':
                cluster_routing_envoy_rh_count += 1
            elif lb_type == 'maglev':
                cluster_routing_envoy_maglev_count += 1
            elif lb_type == 'least_request':
                cluster_routing_envoy_lr_count += 1
            else:
                cluster_routing_envoy_rr_count += 1

            if cluster.get('tls_context', None):
                using_tls = True
                cluster_tls_count += 1

            if cluster.get('grpc', False):
                using_grpc = True
                cluster_grpc_count += 1
            else:
                using_http = True
                cluster_http_count += 1

            cluster_endpoints = cluster.urls if (lb_type == 'kube') else cluster.get('targets', [])

            # Paranoia, really.
            if not cluster_endpoints:
                cluster_endpoints = []

            num_endpoints = len(cluster_endpoints)

            # self.logger.debug(f'cluster {cluster.name}: lb_type {lb_type}, endpoints {cluster_endpoints} ({num_endpoints})')

            if using_tls:
                endpoint_tls_count += num_endpoints

            if using_http:
                endpoint_http_count += num_endpoints

            if using_grpc:
                endpoint_grpc_count += num_endpoints

            if lb_type == 'kube':
                endpoint_routing_kube_count += num_endpoints
            elif lb_type == 'ring_hash':
                endpoint_routing_envoy_rh_count += num_endpoints
            elif lb_type == 'maglev':
                endpoint_routing_envoy_maglev_count += num_endpoints
            elif lb_type == 'least_request':
                endpoint_routing_envoy_lr_count += num_endpoints
            else:
                endpoint_routing_envoy_rr_count += num_endpoints

        od['cluster_count'] = cluster_count
        od['cluster_grpc_count'] = cluster_grpc_count
        od['cluster_http_count'] = cluster_http_count
        od['cluster_tls_count'] = cluster_tls_count
        od['cluster_routing_kube_count'] = cluster_routing_kube_count
        od['cluster_routing_envoy_rr_count'] = cluster_routing_envoy_rr_count
        od['cluster_routing_envoy_rh_count'] = cluster_routing_envoy_rh_count
        od['cluster_routing_envoy_maglev_count'] = cluster_routing_envoy_maglev_count
        od['cluster_routing_envoy_lr_count'] = cluster_routing_envoy_lr_count

        od['endpoint_routing'] = Config.enable_endpoints

        od['endpoint_grpc_count'] = endpoint_grpc_count
        od['endpoint_http_count'] = endpoint_http_count
        od['endpoint_tls_count'] = endpoint_tls_count
        od['endpoint_routing_kube_count'] = endpoint_routing_kube_count
        od['endpoint_routing_envoy_rr_count'] = endpoint_routing_envoy_rr_count
        od['endpoint_routing_envoy_rh_count'] = endpoint_routing_envoy_rh_count
        od['endpoint_routing_envoy_maglev_count'] = endpoint_routing_envoy_maglev_count
        od['endpoint_routing_envoy_lr_count'] = endpoint_routing_envoy_lr_count

        cluster_ingresses = self.aconf.get_config("ClusterIngress")
        od['cluster_ingress_count'] = len(cluster_ingresses.keys()) if cluster_ingresses else 0

        knative_ingresses = self.aconf.get_config("KnativeIngress")
        od['knative_ingress_count'] = len(knative_ingresses.keys()) if knative_ingresses else 0

        od['k8s_ingress_count'] = len(self.aconf.k8s_ingresses)
        od['k8s_ingress_class_count'] = len(self.aconf.k8s_ingress_classes)

        extauth = False
        extauth_proto: Optional[str] = None
        extauth_allow_body = False
        extauth_host_count = 0

        ratelimit = False
        ratelimit_data_plane_proto = False
        ratelimit_custom_domain = False

        tracing = False
        tracing_driver: Optional[str] = None

        for filter in self.filters:
            if filter.kind == 'IRAuth':
                extauth = True
                extauth_proto = filter.get('proto', 'http')
                extauth_allow_body = filter.get('allow_request_body', False)
                extauth_host_count = len(filter.hosts.keys())

        if self.ratelimit:
            ratelimit = True
            ratelimit_data_plane_proto = self.ratelimit.get('data_plane_proto', False)
            ratelimit_custom_domain = bool(self.ratelimit.domain != 'ambassador')

        if self.tracing:
            tracing = True
            tracing_driver = self.tracing.driver

        od['extauth'] = extauth
        od['extauth_proto'] = extauth_proto
        od['extauth_allow_body'] = extauth_allow_body
        od['extauth_host_count'] = extauth_host_count
        od['ratelimit'] = ratelimit
        od['ratelimit_data_plane_proto'] = ratelimit_data_plane_proto
        od['ratelimit_custom_domain'] = ratelimit_custom_domain
        od['tracing'] = tracing
        od['tracing_driver'] = tracing_driver

        group_count = 0
        group_http_count = 0              # HTTPMappingGroups
        group_tcp_count = 0               # TCPMappingGroups
        group_precedence_count = 0        # groups using explicit precedence
        group_header_match_count = 0      # groups using header matches
        group_regex_header_count = 0      # groups using regex header matches
        group_regex_prefix_count = 0      # groups using regex prefix matches
        group_shadow_count = 0            # groups using shadows
        group_shadow_weighted_count = 0   # groups using shadows with non-100% weights
        group_host_redirect_count = 0     # groups using host_redirect
        group_host_rewrite_count = 0      # groups using host_rewrite
        group_canary_count = 0            # groups coalescing multiple mappings
        group_resolver_kube_service = 0   # groups using the KubernetesServiceResolver
        group_resolver_kube_endpoint = 0  # groups using the KubernetesServiceResolver
        group_resolver_consul = 0         # groups using the ConsulResolver
        mapping_count = 0                 # total mappings

        for group in self.ordered_groups():
            group_count += 1

            if group.get('kind', "IRHTTPMappingGroup") == 'IRTCPMappingGroup':
                group_tcp_count += 1
            else:
                group_http_count += 1

            if group.get('precedence', 0) != 0:
                group_precedence_count += 1

            using_headers = False
            using_regex_headers = False

            for header in group.get('headers', []):
                using_headers = True

                if header['regex']:
                    using_regex_headers = True
                    break

            if using_headers:
                group_header_match_count += 1

            if using_regex_headers:
                group_regex_header_count += 1

            if len(group.mappings) > 1:
                group_canary_count += 1

            mapping_count += len(group.mappings)

            if group.get('shadows', []):
                group_shadow_count += 1

                if group.get('weight', 100) != 100:
                    group_shadow_weighted_count += 1

            if group.get('host_redirect', {}):
                group_host_redirect_count += 1

            if group.get('host_rewrite', None):
                group_host_rewrite_count += 1

            res_name = group.get('resolver', self.ambassador_module.get('resolver', 'kubernetes-service'))
            resolver = self.get_resolver(res_name)

            if resolver:
                if resolver.kind == 'KubernetesServiceResolver':
                    group_resolver_kube_service += 1
                elif resolver.kind == 'KubernetesEndpoinhResolver':
                    group_resolver_kube_endpoint += 1
                elif resolver.kind == 'ConsulResolver':
                    group_resolver_consul += 1

        od['group_count'] = group_count
        od['group_http_count'] = group_http_count
        od['group_tcp_count'] = group_tcp_count
        od['group_precedence_count'] = group_precedence_count
        od['group_header_match_count'] = group_header_match_count
        od['group_regex_header_count'] = group_regex_header_count
        od['group_regex_prefix_count'] = group_regex_prefix_count
        od['group_shadow_count'] = group_shadow_count
        od['group_shadow_weighted_count'] = group_shadow_weighted_count
        od['group_host_redirect_count'] = group_host_redirect_count
        od['group_host_rewrite_count'] = group_host_rewrite_count
        od['group_canary_count'] = group_canary_count
        od['group_resolver_kube_service'] = group_resolver_kube_service
        od['group_resolver_kube_endpoint'] = group_resolver_kube_endpoint
        od['group_resolver_consul'] = group_resolver_consul
        od['mapping_count'] = mapping_count

        od['listener_count'] = len(self.listeners)
        od['host_count'] = len(self.hosts)

        return od
