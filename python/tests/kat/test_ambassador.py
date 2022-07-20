import sys

import pytest

# Import all the real tests from other files, to make it easier to pick and choose during development.
import t_basics
import t_bufferlimitbytes
import t_chunked_length
import t_circuitbreaker
import t_cluster_tag
import t_consul
import t_cors
import t_dns_type
import t_envoy_logs
import t_error_response
import t_extauth
import t_grpc
import t_grpc_bridge
import t_grpc_stats
import t_grpc_web
import t_gzip
import t_headerrouting
import t_headerswithunderscoresaction
import t_hosts
import t_ingress
import t_ip_allow_deny
import t_listeneridletimeout
import t_loadbalancer
import t_logservice
import t_lua_scripts
import t_mappingtests_default  # mapping tests executed in the default namespace
import t_max_req_header_kb
import t_no_ui
import t_plain  # t_plain include t_mappingtests_plain and t_optiontests as imports; these tests require each other and need to be executed as a set
import t_queryparameter_routing
import t_ratelimit
import t_redirect
import t_regexrewrite_forwarding
import t_request_header
import t_retrypolicy

# import t_shadow
# import t_stats # t_stats has tests for statsd and dogstatsd. It's too flaky to run all the time.
import t_tcpmapping
import t_tls
import t_tracing
from abstract_tests import AmbassadorTest
from kat.harness import EDGE_STACK, Runner

# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
kat = Runner(AmbassadorTest)
