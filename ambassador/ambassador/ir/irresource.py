from typing import ClassVar, Dict, Optional, TYPE_CHECKING

import json

from ..config import Config
from ..resource import Resource

if TYPE_CHECKING:
    from .ir import IR


class IRResource (Resource):
    """
    A resource within the IR.
    """

    _active: bool

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,
                 kind: str,
                 name: str,
                 location: str = "-ir-",
                 apiVersion: str="ambassador/ir",
                 **kwargs) -> None:
        # print("IRResource __init__ (%s %s)" % (kind, name))

        super().__init__(rkey=rkey, location=location,
                         kind=kind, name=name, apiVersion=apiVersion,
                         **kwargs)
        self.ir = ir
        self.logger = ir.logger

        self.set_active(self.setup(ir, aconf))

    def set_active(self, active: bool) -> None:
        self._active = active

    def is_active(self) -> bool:
        return self._active

    def __nonzero__(self) -> bool:
        return self._active and not self._errors

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # If you don't override setup, you end up with an IRResource that's always active.
        return True

    def add_mappings(self, ir: 'IR', aconf: Config) -> None:
        # If you don't override add_mappings, uh, no mappings will get added.
        pass

    def as_dict(self) -> Dict:
        od = {}

        for k in self.keys():
            if (k == 'apiVersion') or (k == 'logger') or (k == 'ir'):
                # print(k)
                continue
            elif k == '_referenced_by':
                refd_by = sorted([ self._referenced_by[k].location
                                   for k in self._referenced_by.keys() ])

                od['_referenced_by'] = refd_by
            elif k == 'rkey':
                od['_rkey'] = self[k]
            elif isinstance(self[k], IRResource):
                od[k] = self[k].as_dict()
            elif self[k] is not None:
                od[k] = self[k]

        # print("returning %s" % repr(od))
        return od

    def as_json(self, indent=4, sort_keys=True, **kwargs):
        return json.dumps(self.as_dict(), indent=indent, sort_keys=sort_keys, **kwargs)

    @staticmethod
    def normalize_service(service: str) -> str:
        normalized_service = service

        if service.lower().startswith("http://"):
            normalized_service = service[len("http://"):]
        elif service.lower().startswith("https://"):
            normalized_service = service[len("https://"):]

        return normalized_service

    def is_service_tls(self, service: str, context: dict) -> bool:
        is_tls: bool = False

        if service.lower().startswith("https://"):
            is_tls = True

        if context is not None:
            if context in self.ir.tls_contexts:
                is_tls = True

        return is_tls

    def get_service_url(self, service: str, context: dict) -> str:
        normalized_service = self.normalize_service(service)
        is_tls = self.is_service_tls(normalized_service, context)

        service_url = 'tcp://%s' % normalized_service

        port = 443 if is_tls else 80
        if ':' not in normalized_service:
            service_url = '%s:%d' % (service_url, port)

        return service_url

    def get_name_fields(self, service: str, context: dict, host_rewrite) -> str:
        name_fields = None
        is_tls = self.is_service_tls(service, context)

        if is_tls:
            name_fields = ['otls']

        if context is not None:
            if context in self.ir.tls_contexts:
                name_fields.append(context)

        if is_tls and host_rewrite:
            name_fields.append("hr-%s" % host_rewrite)

        return "_".join(name_fields) if name_fields else None

    # def service_tls_check(self, svc, context, host_rewrite):
    #     originate_tls = False
    #     name_fields = None
    #
    #     if svc.lower().startswith("http://"):
    #         originate_tls = False
    #         svc = svc[len("http://"):]
    #     elif svc.lower().startswith("https://"):
    #         originate_tls = True
    #         name_fields = [ 'otls' ]
    #         svc = svc[len("https://"):]
    #     elif context == True:
    #         originate_tls = True
    #         name_fields = [ 'otls' ]
    #
    #     # Separate if here because you need to be able to specify a context
    #     # even after you say "https://" for the service.
    #
    #     if context and (context != True):
    #         if context in self.tls_contexts:
    #             name_fields = [ 'otls', context ]
    #             originate_tls = context
    #         else:
    #             self.logger.error("Originate-TLS context %s is not defined" % context)
    #
    #     if originate_tls and host_rewrite:
    #         name_fields.append("hr-%s" % host_rewrite)
    #
    #     port = 443 if originate_tls else 80
    #     context_name = "_".join(name_fields) if name_fields else None
    #
    #     svc_url = 'tcp://%s' % svc
    #
    #     if ':' not in svc:
    #         svc_url = '%s:%d' % (svc_url, port)
    #
    #     return (svc, svc_url, originate_tls, context_name)
