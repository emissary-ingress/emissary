from kat.harness import Runner, EDGE_STACK

from abstract_tests import AmbassadorTest

# Import all the real tests from other files, to make it easier to pick and choose during development.

import t_basics
import t_cors
import t_extauth
import t_grpc
import t_grpc_bridge
import t_grpc_web
import t_gzip
import t_headerrouting
import t_hosts
import t_loadbalancer
import t_logservice
import t_lua_scripts
import t_mappingtests
import t_no_ui
import t_optiontests
import t_plain
import t_ratelimit
import t_redirect
#import t_shadow
#import t_stats
import t_tcpmapping
import t_tls
import t_tracing
import t_retrypolicy
import t_consul
#import t_circuitbreaker
import t_envoy_logs
import t_ingress
import t_listeneridletimeout
import t_cluster_tag

# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
main = Runner(AmbassadorTest)
