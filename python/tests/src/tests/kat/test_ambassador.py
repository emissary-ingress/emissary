# Import all the real tests from other files, to make it easier to pick and choose during development.

# import t_shadow
# import t_stats # t_stats has tests for statsd and dogstatsd. It's too flaky to run all the time.
from abstract_tests import AmbassadorTest
from kat.harness import Runner

# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
kat = Runner(AmbassadorTest)
