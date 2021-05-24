// Incorporates: git://github.com/coreos/go-systemd.git 8843929dce14ef03e3f9b78c1707d347cbf661b9 activation/files.go
//
// Copyright 2015 CoreOS, Inc.
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

package sd_daemon

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// ListenFds returns a list of file descriptors passed in by the
// service manager as part of the socket-based activation logic.
//
// If unsetEnv is true, then (regardless of whether the function call
// itself succeeds or not) it will unset the environmental variables
// LISTEN_FDS and LISTEN_PID, which will cause further calls to this
// function to fail.
//
// In the case of an error, this function returns nil.
func ListenFds(unsetEnv bool) []*os.File {
	if unsetEnv {
		defer func() {
			_ = os.Unsetenv("LISTEN_PID")
			_ = os.Unsetenv("LISTEN_FDS")
			_ = os.Unsetenv("LISTEN_FDNAMES")
		}()
	}

	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil || pid != os.Getpid() {
		return nil
	}

	nfds, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || nfds < 1 {
		return nil
	}

	names := strings.Split(os.Getenv("LISTEN_FDNAMES"), ":")

	files := make([]*os.File, 0, nfds)
	for i := 0; i < nfds; i++ {
		fd := i + 3
		unix.CloseOnExec(fd)
		name := "unknown"
		if i < len(names) {
			name = names[i]
		}
		files = append(files, os.NewFile(uintptr(fd), name))
	}

	return files
}
