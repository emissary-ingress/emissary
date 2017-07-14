#!python

import sys

import os

registry = os.environ.get('DOCKER_REGISTRY', None)

if not registry:
    sys.stderr.write("DOCKER_REGISTRY must be set\n")
    sys.exit(1)

if registry == '-':
    registry = ''

if registry and not registry.endswith("/"):
    registry += "/"

ambassador_registry = os.environ.get('AMBASSADOR_REGISTRY', registry)
statsd_registry = os.environ.get('STATSD_REGISTRY', registry)

TemplateVars = dict(os.environ)

TemplateVars['DOCKER_REGISTRY'] = registry
TemplateVars['AMREG'] = ambassador_registry
TemplateVars['STREG'] = statsd_registry
# TemplateVars['VERSION'] = os.environ.get('VERSION')

template = sys.stdin.read()

try:
    # for key in sorted(TemplateVars.keys()):
    #     sys.stderr.write("%s: %s\n" % (key, TemplateVars[key]))

    data = template.format(**TemplateVars)
    sys.stdout.write(data)
except KeyError as e:
    sys.stderr.write("Missing key: %s" % e.args)

