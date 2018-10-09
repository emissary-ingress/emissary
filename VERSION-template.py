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

Version = "{{VERSION}}"
GitDescription = "{{GITDESCRIPTION}}"
GitBranch = "{{GITBRANCH}}"
GitCommit = "{{GITCOMMIT}}"
GitDirty = bool("{{GITDIRTY}}")

if __name__ == "__main__":
    import sys

    cmd = "--compact"

    if len(sys.argv) > 1:
        cmd = sys.argv[1].lower()

    if (cmd == '--version') or (cmd == '-V'):
        print(Version)
    elif cmd == '--desc':
        print(GitDescription)
    elif cmd == '--branch':
        print(GitBranch)
    elif cmd == '--commit':
        print(GitCommit)
    elif cmd == '--dirty':
        print(GitDirty)
    elif cmd == '--all':
        print("Version:        %s" % Version)
        print("GitBranch:      %s" % GitBranch)
        print("GitCommit:      %s" % GitCommit)
        print("GitDirty:       %s" % GitDirty)
        print("GitDescription: %s" % GitDescription)
    else: # compact
        print("%s (%s at %s on %s%s)" %
              (Version, GitDescription, GitCommit, GitBranch, " - dirty" if GitDirty else ""))
