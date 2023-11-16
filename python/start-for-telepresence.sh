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

ln -sf $TELEPRESENCE_ROOT/var/run/secrets /var/run/secrets
export LC_ALL=C.UTF-8
export LANG=C.UTF-8
python3 setup.py develop
AMBASSADOR_NO_DIAGD=true bash entrypoint.sh
