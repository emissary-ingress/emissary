# Copyright (C) 2016-2017 Luke Shumaker <lukeshu@lukeshu.com>
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
# limitations under the License.

# header
ifneq ($(topsrcdir),)
include $(topsrcdir)/build-aux/Makefile.head.mk
else
srcdir = .
.DEFAULT_GOAL = files.generate
endif

# body

files.src.gen += log_util.go

$(srcdir)/%.go: $(srcdir)/%.go.gen
	cd $(@D) && ./$(^F) > $(@F)

# footer
ifneq ($(topsrcdir),)
include $(topsrcdir)/build-aux/Makefile.tail.mk
else
files.generate: $(files.src.gen)
maintainer-clean:
	rm -f -- $(files.src.gen) $(files.src.int)
.PHONY: files.generate maintainer-clean
.DELETE_ON_ERROR:
endif
