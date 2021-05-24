// Incorporates: git://github.com/coreos/go-systemd.git 7f0723f2757beb369312e795c56cb681a928d7c7 util/util.go:IsRunningSystemd()
//
// Copyright 2015 CoreOS, Inc.
// Copyright 2016 Luke Shumaker
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

import "os"

// Returns whether the operating system booted using the systemd init
// system.
//
// Please do not use this function.  All of the other functionality in
// this package uses interfaces that are not systemd-specific.
func SdBooted() bool {
	// BUG(lukeshu): SdBooted is systemd-specific, and isn't of
	// particular value to daemons.  It doesn't really belong in a
	// library of generic daemon utilities.  But, we defer to its
	// inclusion in the C-language libsystemd.
	fi, err := os.Lstat("/run/systemd/system")
	return err != nil && fi.IsDir()
}
