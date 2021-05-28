// Incorporates: git://github.com/coreos/go-systemd.git 0c088eaedf4396216a47ca971d4630f1697186bf daemon/watchdog.go
//
// Copyright 2016 CoreOS, Inc.
// Copyright 2016, 2018 Luke Shumaker
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
	"time"
)

// WatchdogEnabled returns how often the process is expected to send a
// keep-alive notification to the service manager.
//
//     Notification{State: "WATCHDOG=1"}.Send(false) // send keep-alive notification
//
// If unsetEnv is true, then (regardless of whether the function call
// itself succeeds or not) it will unset the environmental variables
// WATCHDOG_USEC and WATCHDOG_PID, which will cause further calls to
// this function to fail.
//
// If an error is not returned, then the duration returned is greater
// than 0; if an error is returned, then the duration is 0.  If the
// service manager is not expecting a keep-alive notification from
// this process (or if this has already been called with
// unsetEnv=true), then the error is ErrDisabled.
func WatchdogEnabled(unsetEnv bool) (time.Duration, error) {
	if unsetEnv {
		defer func() {
			_ = os.Unsetenv("WATCHDOG_USEC")
			_ = os.Unsetenv("WATCHDOG_PID")
		}()
	}

	usecStr, haveUsec := os.LookupEnv("WATCHDOG_USEC")
	if !haveUsec {
		return 0, ErrDisabled
	}
	usec, err := strconv.ParseInt(usecStr, 10, 64)
	if err != nil || usec < 0 {
		return 0, err
	}

	if pidStr, havePid := os.LookupEnv("WATCHDOG_PID"); havePid {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return 0, err
		}
		if pid != os.Getpid() {
			return 0, ErrDisabled
		}
	}

	return time.Duration(usec) * time.Microsecond, nil
}
