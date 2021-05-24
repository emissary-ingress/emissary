// Copyright 2015-2016, 2018-2019 Luke Shumaker
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
	"fmt"
	"os"
)

// daemon(7) recommends using the exit codes defined in the "LSB
// recommendations for SysV init scripts"[1].
//
// BSD sysexits.h (which is also in GNU libc) defines several exit
// codes in the range 64-78.  These are typically used in the context
// of mail delivery; originating with BSD delivermail (the NCP
// predecessor to the TCP/IP sendmail), and are still used by modern
// mail systems such as Postfix to interpret the local(8) delivery
// agent's exit status.  Using these for service exit codes isn't
// recommended by LSB (which says they are in the range reserved for
// future LSB use) or by daemon(7).  However, they are used in
// practice, and are recognized by systemd.
//
// [1]: http://refspecs.linuxbase.org/LSB_5.0.0/LSB-Core-generic/LSB-Core-generic/iniscrptact.html
//
// sysexits(3): https://www.freebsd.org/cgi/man.cgi?query=sysexits
//
// local(8): http://www.postfix.org/local.8.html
const (
	//   0-  8 are currently defined by LSB.
	EXIT_SUCCESS         uint8 = 0
	EXIT_FAILURE         uint8 = 1
	EXIT_INVALIDARGUMENT uint8 = 2
	EXIT_NOTIMPLEMENTED  uint8 = 3
	EXIT_NOPERMISSION    uint8 = 4
	EXIT_NOTINSTALLED    uint8 = 5
	EXIT_NOTCONFIGURED   uint8 = 6
	EXIT_NOTRUNNING      uint8 = 7
	//
	//   8- 99 are reserved for future LSB use.
	//         However, let's provide the EX_ codes from
	//         sysexits.h anyway.
	EX_OK          uint8 = EXIT_SUCCESS
	EX_USAGE       uint8 = 64 // command line usage error
	EX_DATAERR     uint8 = 65 // data format error
	EX_NOINPUT     uint8 = 66 // cannot open input
	EX_NOUSER      uint8 = 67 // addressee unknown
	EX_NOHOST      uint8 = 68 // host name unknown
	EX_UNAVAILABLE uint8 = 69 // service unavailable
	EX_SOFTWARE    uint8 = 70 // internal software error
	EX_OSERR       uint8 = 71 // system error (e.g., can't fork)
	EX_OSFILE      uint8 = 72 // critical OS file missing
	EX_CANTCREAT   uint8 = 73 // can't create (user) output file
	EX_IOERR       uint8 = 74 // input/output error
	EX_TEMPFAIL    uint8 = 75 // temp failure; user is invited to retry
	EX_PROTOCOL    uint8 = 76 // remote error in protocol
	EX_NOPERM      uint8 = 77 // permission denied
	EX_CONFIG      uint8 = 78 // configuration error
	//
	// 100-149 are reserved for distribution use.
	//
	// 150-199 are reserved for application use.
	//
	// 200-254 are reserved (for init system use).
	//         So, take codes 200+ from systemd's
	//         `src/basic/exit-status.h`
	//         (last updated for SD v242)
	EXIT_CHDIR                   uint8 = 200 // SD v8+
	EXIT_NICE                    uint8 = 201 // SD v8+
	EXIT_FDS                     uint8 = 202 // SD v8+
	EXIT_EXEC                    uint8 = 203 // SD v8+
	EXIT_MEMORY                  uint8 = 204 // SD v8+
	EXIT_LIMITS                  uint8 = 205 // SD v8+
	EXIT_OOM_ADJUST              uint8 = 206 // SD v8+
	EXIT_SIGNAL_MASK             uint8 = 207 // SD v8+
	EXIT_STDIN                   uint8 = 208 // SD v8+
	EXIT_STDOUT                  uint8 = 209 // SD v8+
	EXIT_CHROOT                  uint8 = 210 // SD v8+
	EXIT_IOPRIO                  uint8 = 211 // SD v8+
	EXIT_TIMERSLACK              uint8 = 212 // SD v8+
	EXIT_SECUREBITS              uint8 = 213 // SD v8+
	EXIT_SETSCHEDULER            uint8 = 214 // SD v8+
	EXIT_CPUAFFINITY             uint8 = 215 // SD v8+
	EXIT_GROUP                   uint8 = 216 // SD v8+
	EXIT_USER                    uint8 = 217 // SD v8+
	EXIT_CAPABILITIES            uint8 = 218 // SD v8+
	EXIT_CGROUP                  uint8 = 219 // SD v8+
	EXIT_SETSID                  uint8 = 220 // SD v8+
	EXIT_CONFIRM                 uint8 = 221 // SD v8+
	EXIT_STDERR                  uint8 = 222 // SD v8+
	EXIT_TCPWRAP                 uint8 = 223 // SD v8-v211
	EXIT_PAM                     uint8 = 224 // SD v8+
	EXIT_NETWORK                 uint8 = 225 // SD v33+
	EXIT_NAMESPACE               uint8 = 226 // SD v38+
	EXIT_NO_NEW_PRIVILEGES       uint8 = 227 // SD v187+
	EXIT_SECCOMP                 uint8 = 228 // SD v187+
	EXIT_SELINUX_CONTEXT         uint8 = 229 // SD v209+
	EXIT_PERSONALITY             uint8 = 230 // SD v209+
	EXIT_APPARMOR_PROFILE        uint8 = 231 // SD v210+
	EXIT_ADDRESS_FAMILIES        uint8 = 232 // SD v211+
	EXIT_RUNTIME_DIRECTORY       uint8 = 233 // SD v211+
	EXIT_MAKE_STARTER            uint8 = 234 // SD v214-v234
	EXIT_CHOWN                   uint8 = 235 // SD v214+
	EXIT_SMACK_PROCESS_LABEL     uint8 = 236 // SD v230+; was BUS_ENDPOINT in SD v217-v229
	EXIT_KEYRING                 uint8 = 237 // SD v233+; was SMACK_PROCESS_LABEL in SD v218-v229
	EXIT_STATE_DIRECTORY         uint8 = 238 // SD v235+
	EXIT_CACHE_DIRECTORY         uint8 = 239 // SD v235+
	EXIT_LOGS_DIRECTORY          uint8 = 240 // SD v235+
	EXIT_CONFIGURATION_DIRECTORY uint8 = 241 // SD v235+
)

// Recover is a utility function to defer at the beginning of a
// goroutine in order to have the correct exit code in the case of a
// panic.
func Recover() {
	if r := recover(); r != nil {
		Log.Err(fmt.Sprintf("panic: %v", r))
		os.Exit(int(EXIT_FAILURE))
	}
}
