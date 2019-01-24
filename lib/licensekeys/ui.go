package licensekeys

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
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

type cmdContext struct {
	defaultKeyfile    string
	defaultKeyfileErr error

	keyfile string
	key     string

	application string
	version     string
}

func (ctx *cmdContext) KeyCheck(flags *flag.FlagSet) error {
	var keysource string

	if ctx.key == "" {
		if !flags.Changed("license-file") && ctx.defaultKeyfileErr != nil {
			return errors.Wrap(ctx.defaultKeyfileErr, "error determining license key file")
		}
		if ctx.keyfile == "" {
			return errors.New("no license key or license key file specified")
		}
		key, err := ioutil.ReadFile(ctx.keyfile)
		if err != nil {
			return errors.Wrap(err, "error reading license key")
		}
		ctx.key = strings.TrimSpace(string(key))
		keysource = "key from file " + ctx.keyfile
	} else {
		if flags.Changed("license-key") {
			keysource = "key from command line"
		} else {
			keysource = "key from environment"
		}
	}

	claims, token, err := ParseKey(ctx.key)

	go func() {
		err := PhoneHome(claims, ctx.application, ctx.version)
		if err != nil {
			fmt.Fprintln(os.Stderr, "metriton error:", err)
		}
	}()

	if err != nil {
		return errors.Wrapf(err, "error validating %s", keysource)
	}
	if !token.Valid {
		// Can this even happen if err == nil?  IDK, that kind
		// of useful thing is missing from the
		// github.com/dgrijalva/jwt-go docs.
		return errors.Errorf("%s is not valid", keysource)
	}

	return nil
}

func InitializeCommandFlags(flags *flag.FlagSet, application, version string) func(*flag.FlagSet) error {
	ctx := &cmdContext{
		application: application,
		version:     version,
	}
	ctx.defaultKeyfile, ctx.defaultKeyfileErr = defaultLicenseFile()

	flags.StringVar(&ctx.key, "license-key", os.Getenv("AMBASSADOR_LICENSE_KEY"), "ambassador license key")
	flags.StringVar(&ctx.keyfile, "license-file", ctx.defaultKeyfile, "ambassador license file")

	return ctx.KeyCheck
}
