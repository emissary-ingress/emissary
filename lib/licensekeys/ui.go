package licensekeys

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	// Field in the Secret where the license is stored
	DefaultSecretLicenseField = "license-key"

	// Environment variable where users can pass the license
	DefaultLicenseEnvVar = "AMBASSADOR_LICENSE_KEY"
)

// userConfigDir returns the default directory to use for
// user-specific config data.  It is similar to os.UserCacheDir().
func userConfigDir(goos string) (string, error) {
	var dir string

	switch goos {
	case "darwin":
		// https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/FileSystemOverview/FileSystemOverview.html
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Application Support"

	case "linux":
		// http://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html
		dir = os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
			}
			dir += "/.config"
		}

	default:
		return "", errors.New(`Only the "darwin" and "linux" GOOS are supported at this time`)
	}

	return dir, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func defaultLicenseFile() (string, error) {
	filename := os.Getenv("AMBASSADOR_LICENSE_FILE")
	if filename != "" {
		return filename, nil
	}
	cfgDir, err := userConfigDir(runtime.GOOS)
	if err != nil {
		return "", err
	}
	filename = filepath.Join(cfgDir, "ambassador", "license-key")
	if runtime.GOOS == "darwin" && !fileExists(filename) {
		// Some macOS users expect XDG file paths to work,
		// because other cross-platform applications (like
		// gcloud & pgcli) use them.
		if xdgDir, err := userConfigDir("linux"); err == nil {
			xdgFile := filepath.Join(xdgDir, "ambassador", "license-key")
			if fileExists(xdgFile) {
				filename = xdgFile
			}
		}
	}
	if !fileExists(filename) {
		// for compatibility with < 0.1.1
		if home := os.Getenv("HOME"); home != "" {
			legacyFile := filepath.Join(home, ".ambassador.key")
			if fileExists(legacyFile) {
				filename = legacyFile
			}
		}
	}
	return filename, nil
}

// LicenseContext is a license, provided in a string or in a file
type LicenseContext struct {
	Keyfile string
	key     string
}

// Clear clears any license loaded (this does not affect the env var or license file)
func (ctx *LicenseContext) Clear() {
	ctx.SetKey([]byte{})
}

func (ctx *LicenseContext) SetKey(k []byte) {
	ctx.key = string(k)
}

func (ctx *LicenseContext) CopyKeyFrom(c *LicenseContext) {
	ctx.key = c.key
}

// HasKey returns True if the current context does have a key (ignoring the license file)
func (ctx LicenseContext) HasKey() bool {
	if ctx.key != "" {
		return true
	}
	if e := os.Getenv(DefaultLicenseEnvVar); len(e) > 0 {
		return true
	}
	return false
}

// String implements the Stringer
func (ctx LicenseContext) String() string {
	if ctx.key != "" {
		return fmt.Sprintf("str:%s...", ctx.key[:10])
	}
	if e := os.Getenv(DefaultLicenseEnvVar); len(e) > 0 {
		return fmt.Sprintf("%q...", ctx.key[:10])
	}
	return "(no key)"
}

// GetClaims checks that the license contained in the key or in the keyfile is valid
func (ctx *LicenseContext) GetClaims() (*LicenseClaimsLatest, error) {
	k := ""
	if ctx.key != "" {
		k = ctx.key
	} else if e := os.Getenv(DefaultLicenseEnvVar); len(e) > 0 {
		k = e
	} else if ctx.Keyfile != "" {
		key, err := ioutil.ReadFile(ctx.Keyfile)
		if err == nil {
			k = strings.TrimSpace(string(key))
		}
	}

	if k != "" {
		if claim, err := ParseKey(k); err == nil {
			return claim, nil
		}
	}

	claims := NewUnregisteredLicenseClaims()
	claims.CustomerID = DefUnregisteredCustomerID
	return claims, nil
}

// AddFlagsTo adds license flags to the command line parser
func (ctx *LicenseContext) AddFlagsTo(cmd *cobra.Command) error {
	defaultKeyfile, err := defaultLicenseFile()
	if err != nil {
		return err
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&ctx.key, "license-key", "", "ambassador license key")
	flags.StringVar(&ctx.Keyfile, "license-file", defaultKeyfile, "ambassador license file")
	return nil
}
