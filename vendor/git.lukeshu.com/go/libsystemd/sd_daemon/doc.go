// Copyright 2015-2016 Luke Shumaker
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package sd_daemon implements utilities for writing "new-style"
// daemons.
//
// The daemon(7) manual page has historically documented the very long
// list of things that a daemon must do at start-up to be a
// well-behaved SysV daemon.  Modern service managers allow daemons to
// be much simpler; modern versions of the daemon(7) page on GNU/Linux
// systems also describe "new-style" daemons.  Though many of the
// mechanisms described there and implemented here originated with
// systemd, they are all very simple mechanisms which can easily be
// implemented with a variety of service managers.
//
// daemon(7): https://www.freedesktop.org/software/systemd/man/daemon.html
//
// BUG(lukeshu): Logically, sd_id128.GetInvocationID might belong in
// this package, but we defer to the C-language libsystemd's placement
// of sd_id128_get_invocation() in sd-id128.h.
package sd_daemon

import "errors"

// ErrDisabled is the error returned when the service manager does not
// want/support a mechanism; or when that mechanism has been disabled
// for this process by setting unsetEnv=true when calling one of these
// functions.
var ErrDisabled = errors.New("Mechanism Disabled")
