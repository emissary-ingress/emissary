# The `test-ratelimit` image gets built by `build-aux/check.mk` for
# use by `python/kat/harness.py:TestImage`.

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

FROM node:10-alpine

LABEL PROJECT_REPO_URL         = "git@github.com:datawire/ambassador.git" \
      PROJECT_REPO_BROWSER_URL = "https://github.com/datawire/ambassador" \
      DESCRIPTION              = "Ambassador REST Service" \
      VENDOR                   = "Datawire" \
      VENDOR_URL               = "https://datawire.io/"

ENV NODE_ENV production

WORKDIR /app

ADD . .
RUN npm install

CMD ["npm", "start"]
