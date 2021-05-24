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

// +build !linux

package sd_daemon

import (
	"bytes"
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func (msg Notification) send(unsetEnv bool) error {
	if unsetEnv {
		defer func() { _ = os.Unsetenv("NOTIFY_SOCKET") }()
	}

	socketAddr := &net.UnixAddr{
		Name: os.Getenv("NOTIFY_SOCKET"),
		Net:  "unixgram",
	}

	if socketAddr.Name == "" {
		return ErrDisabled
	}

	conn, err := socketUnixgram(socketAddr.Name)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	var cmsgs [][]byte

	if len(msg.Files) > 0 {
		fds := make([]int, len(msg.Files))
		for i := range msg.Files {
			fds[i] = int(msg.Files[i].Fd())
		}
		cmsg := unix.UnixRights(fds...)
		cmsgs = append(cmsgs, cmsg)
	}

	havePid := msg.PID > 0 && msg.PID != os.Getpid()
	if havePid {
		// BUG(lukeshu): Spoofing the socket credentials is
		// not implemnted on non-Linux kernels.  If you are
		// knowledgable about how to do this on other kernels,
		// please let me know at <lukeshu@lukeshu.com>!
		havePid = false
	}

	// If the 2nd argument is empty, this is equivalent to
	//
	//    conn, _ := net.DialUnix(socketAddr.Net, nil, socketAddr)
	//    conn.Write([]byte(msg.State))
	_, _, err = conn.WriteMsgUnix([]byte(msg.State), bytes.Join(cmsgs, nil), socketAddr)

	if err != nil && havePid {
		// Maybe it failed because we don't have privileges to
		// spoof our pid; retry without spoofing the pid.
		//
		// I'm not too happy that we do this silently without
		// notifying the caller, but that's what
		// sd_pid_notify_with_fds does.
		cmsgs = cmsgs[:len(cmsgs)-1]
		_, _, err = conn.WriteMsgUnix([]byte(msg.State), bytes.Join(cmsgs, nil), socketAddr)
	}

	return err
}

// socketUnixgram wraps socket(2), but doesn't bind(2) or connect(2)
// the socket to anything.  This is an ugly hack because none of the
// functions in "net" actually allow you to get a AF_UNIX socket not
// bound/connected to anything.
//
// At some point you begin to question if it is worth it to keep up
// the high-level interface of "net", and messing around with FileConn
// and UnixConn.  Maybe we just drop to using unix.Socket and
// unix.SendmsgN directly.
//
// See: net/sys_cloexec.go:sysSocket()
func socketUnixgram(name string) (*net.UnixConn, error) {
	syscall.ForkLock.RLock()
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	if err == nil {
		syscall.CloseOnExec(fd)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer unix.Close(fd)
	// Don't bother calling unix.SetNonblock(), net.FileConn()
	// will call syscall.SetNonblock().
	conn, err := net.FileConn(os.NewFile(uintptr(fd), name))
	if err != nil {
		return nil, err
	}
	unixConn := conn.(*net.UnixConn)
	return unixConn, nil
}
