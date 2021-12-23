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

import os

Version = "dirty"
Commit = "HEAD"

try:
    with open(os.path.join(os.path.dirname(__file__), "..", "ambassador.version")) as version:
        info = version.read().split('\n')
        if len(info) < 2:
            info.append("MISSING")
            if len(info) < 1:
                info.append("MISSING")

        Version = info[0]
        Commit = info[1]

except FileNotFoundError:
    pass

if __name__ == "__main__":
    import sys

    cmd = "--compact"

    if len(sys.argv) > 1:
        cmd = sys.argv[1].lower()

    if (cmd == '--version') or (cmd == '-V'):
        print(Version)
    elif cmd == '--desc':
        print(Build.git.description)
    elif cmd == '--branch':
        print(Build.git.branch)
    elif cmd == '--commit':
        print(Build.git.commit)
    elif cmd == '--dirty':
        print(Build.git.dirty)
    elif cmd == '--all':
        print("version:         %s" % Version)
        print("git.branch:      %s" % Build.git.branch)
        print("git.commit:      %s" % Build.git.commit)
        print("git.dirty:       %s" % Build.git.dirty)
        print("git.description: %s" % Build.git.description)
    else:   # compact
        print("%s (%s at %s on %s%s)" %
              (Version, Build.git.description, Build.git.commit, Build.git.branch,
               " - dirty" if Build.git.dirty else ""))
