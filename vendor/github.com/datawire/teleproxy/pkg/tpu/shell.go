package tpu

import (
	"os/exec"
	"strings"

	"github.com/google/shlex"
)

func Shell(command string) (result string, err error) {
	return ShellLog(command, func(string) {})
}

func ShellLogf(command string, logf func(string, ...interface{})) (string, error) {
	return ShellLog(command, func(line string) { logf("%s", line) })
}

func ShellLog(command string, logln func(string)) (string, error) {
	return CmdLog([]string{"sh", "-c", command}, logln)
}

func ShlexLogf(command string, logf func(string, ...interface{})) (string, error) {
	return ShlexLog(command, func(line string) { logf("%s", line) })
}

func ShlexLog(command string, logln func(string)) (string, error) {
	parts, err := shlex.Split(command)
	if err != nil {
		logln(err.Error())
		return "", err
	}
	return CmdLog(parts, logln)
}

func CmdLogf(command []string, logf func(string, ...interface{})) (string, error) {
	return CmdLog(command, func(line string) { logf("%s", line) })
}

func CmdLog(command []string, logln func(string)) (string, error) {
	logln(strings.Join(command, " "))
	cmd := exec.Command(command[0], command[1:]...)
	out, err := cmd.CombinedOutput()
	str := string(out)
	lines := strings.Split(str, "\n")
	for idx, line := range lines {
		if strings.TrimSpace(line) != "" {
			logln(line)
		} else if idx != len(lines)-1 {
			logln(line)
		}
	}
	if err != nil {
		logln(err.Error())
	}
	return str, err
}
