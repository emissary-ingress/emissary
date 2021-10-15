import sys
from conftest import letter_range

from kat.harness import Runner, EDGE_STACK

from abstract_tests import AmbassadorTest

# Import all the real tests from other files, to make it easier to pick and choose during development.

if letter_range not in ["all", "ah", "ip", "qz"]:
    print("Unknown test file name letter range: %s!" % letter_range)
    sys.exit(1)

if letter_range in ["all","ah"]:
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

if letter_range in ["all","ip"]:
	import t_ingress
	import t_ip_allow_deny
	import t_listeneridletimeout
	import t_loadbalancer
	import t_logservice
	import t_lua_scripts
	import t_mappingtests_default
	import t_max_req_header_kb
	import t_no_ui
	import t_plain # includes t_mappingtests_plain, t_optiontests

if letter_range in ["all","qz"]:
	import t_queryparameter_routing
	import t_ratelimit
	import t_redirect
	import t_regexrewrite_forwarding
	import t_request_header
	import t_retrypolicy
	#import t_shadow
	#import t_stats
	import t_tcpmapping
	import t_tls
	import t_tracing

# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
main = Runner(AmbassadorTest)
