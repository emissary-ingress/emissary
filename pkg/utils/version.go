package utils

import (
	"os"
	"strings"
)

const (
	defaultMissingVersion = "MISSING(FILE)"
	versionFileName       = "/buildroot/ambassador/python/ambassador.version"
)

// GetVersion parses the version file bundled with the binary
//
// The version number is set at run-time by reading the 'ambassador.version' file.  We do
// this instead of compiling in a version so that we can promote RC images to GA without
// recompiling anything.
//
// Keep this parsing logic in-sync with VERSION.py.
func GetVersion() string {
	if verBytes, err := os.ReadFile(versionFileName); err == nil {
		verLines := strings.Split(string(verBytes), "\n")
		for len(verLines) < 2 {
			verLines = append(verLines, "MISSING(VAL)")
		}
		return verLines[0]
	}

	return defaultMissingVersion
}
