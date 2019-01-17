package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
)

var apictl = &cobra.Command{
	Use:              "apictl [command]",
	PersistentPreRun: keyCheck,
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

var (
	LICENSE_KEY   string
	LICENSE_FILE  string
	LICENSE_PAUSE = map[*cobra.Command]bool{
		watch: true,
	}
)

// userConfigDir returns the default directory to use for
// user-specific config data.  It is similar to os.UserCacheDir().
func userConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
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
	cfgDir, err := userConfigDir()
	if err != nil {
		return "", err
	}
	filename = filepath.Join(cfgDir, "ambassador", "license-key")
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

var defaultKeyfile string
var defaultKeyfileErr error

func init() {
	apictl.PersistentFlags().StringVar(&LICENSE_KEY, "license-key", os.Getenv("AMBASSADOR_LICENSE_KEY"), "ambassador license key")
	defaultKeyfile, defaultKeyfileErr = defaultLicenseFile()
	apictl.PersistentFlags().StringVar(&LICENSE_FILE, "license-file", defaultKeyfile, "ambassador license file")
}

func keyCheck(cmd *cobra.Command, args []string) {
	var keysource string

	if LICENSE_KEY == "" {
		if !cmd.Flag("license-file").Changed && defaultKeyfileErr != nil {
			fmt.Fprintln(os.Stderr, "error determining license key file:", defaultKeyfileErr)
			os.Exit(1)
		}
		if LICENSE_FILE == "" {
			fmt.Fprintln(os.Stderr, "no license key or license key file specified")
			os.Exit(1)
		}
		key, err := ioutil.ReadFile(LICENSE_FILE)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading license key:", err)
			os.Exit(1)
		}
		LICENSE_KEY = strings.TrimSpace(string(key))
		keysource = "key from file " + LICENSE_FILE
	} else {
		if cmd.Flag("license-key").Changed {
			keysource = "key from command line"
		} else {
			keysource = "key from environment"
		}
	}

	claims, token, err := licensekeys.ParseKey(LICENSE_KEY)

	go func() {
		err := licensekeys.PhoneHome(claims, Version)
		if err != nil {
			fmt.Fprintln(os.Stderr, "metriton error:", err)
		}
	}()

	if !token.Valid || err != nil {
		fmt.Fprintf(os.Stderr, "error validating %s: %v\n", keysource, err)
		pause, ok := LICENSE_PAUSE[cmd]
		if !ok {
			pause = false
		}
		if pause {
			time.Sleep(5 * 60 * time.Second)
		}
		os.Exit(1)
	}
}

func main() {
	apictl.Execute()
}

func die(err error, args ...interface{}) {
	if err != nil {
		if args != nil {
			fmt.Printf("%v: %v\n", err, args)
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}
