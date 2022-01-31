# Copyright 2018-2022 Datawire. All rights reserved.
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

import os

# Keep this in-sync with cmd/busyambassador/main.go.
#
# We don't report or log errors here, we just silently fall back to some static "MISSING(XXX)"
# strings.  This is in-part because the code here is running pretty early, and logging setup hasn't
# happened yet.  Also because any errors will be evident when the version number gets logged and
# it's this static string.
Version = "MISSING(FILE)"
Commit = "MISSING(FILE)"
try:
    with open(os.path.join(os.path.dirname(__file__), "..", "ambassador.version")) as version:
        info = version.read().split('\n')
        while len(info) < 2:
            info.append("MISSING(VAL)")

        Version = info[0]
        Commit = info[1]
except FileNotFoundError:
    pass
