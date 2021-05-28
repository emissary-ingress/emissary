// Incorporates: git://github.com/docker/docker.git 18c7c67308bd4a24a41028e63c2603bb74eac85e pkg/systemd/sd_notify.go
// Incorporates: git://github.com/coreos/go-systemd.git a606a1e936df81b70d85448221c7b1c6d8a74ef1 daemon/sdnotify.go
//
// Copyright 2013, 2015 Docker, Inc.
// Copyright 2014 CoreOS, Inc.
// Copyright 2015-2019 Luke Shumaker
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
)

// Notification is a message to be sent to the service manager about
// state changes.
type Notification struct {
	// PID specifies which process to send a notification about.
	// If PID <= 0, or if the current process does not have
	// privileges to send messages on behalf of other processes,
	// then the message is simply sent from the current process.
	//
	// BUG(lukeshu): Spoofing the PID is not implemented on
	// non-Linux kernels.  If you are knowledgable about how to do
	// this on other kernels, please let me know at
	// <lukeshu@lukeshu.com>!
	PID int

	// State should contain a newline-separated list of variable
	// assignments.  See the documentation for sd_notify(3) for
	// well-known variable assignments.
	//
	// https://www.freedesktop.org/software/systemd/man/sd_notify.html
	State string

	// Files is a list of file descriptors to send to the service
	// manager with the message.  This is useful for keeping files
	// open across restarts, as it enables the service manager to
	// pass those files to the new process when it is restarted
	// (see ListenFds).
	//
	// Note: The service manager will only actually store the file
	// descriptors if you include "FDSTORE=1" in the state (again,
	// see sd_notify(3) for well-known variable assignments).
	Files []*os.File
}

// Send sends the Notification to the service manager.
//
// If unsetEnv is true, then (regardless of whether the function call
// itself succeeds or not) it will unset the environmental variable
// NOTIFY_SOCKET, which will cause further notify operations to fail.
//
// If the service manager is not listening for notifications from this
// process tree (or a Notification has has already been send with
// unsetEnv=true), then ErrDisabled is returned.  If the service
// manager appears to be listening, but there is an error sending the
// message, then that error is returned.
//
// It is generally recommended that you ignore the return value: if
// there is an error, then this is function no-op; meaning that by
// calling the function but ignoring the return value, you can easily
// support both service managers that support these notifications and
// those that do not.
func (msg Notification) Send(unsetEnv bool) error {
	return msg.send(unsetEnv)
}
