#!python

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

TemplateVars = dict(os.environ)

TemplateVars['DOCKER_REGISTRY'] = registry
TemplateVars['AMREG'] = ambassador_registry
# TemplateVars['VERSION'] = os.environ.get('VERSION')

template = sys.stdin.read()

try:
    # for key in sorted(TemplateVars.keys()):
    #     sys.stderr.write("%s: %s\n" % (key, TemplateVars[key]))

    data = template.format(**TemplateVars)
    sys.stdout.write(data)
except KeyError as e:
    sys.stderr.write("Missing key: %s" % e.args)

