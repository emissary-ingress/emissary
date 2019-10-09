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

import sys

import json
import dpath.util

x = json.load(sys.stdin)
y = None

try:
    y = dpath.util.get(x, sys.argv[1])
    print(json.dumps(y, sort_keys=True, indent=4))
    sys.exit(0)
except KeyError:
    sys.exit(1)

