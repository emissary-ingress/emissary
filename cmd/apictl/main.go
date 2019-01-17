package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
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

func init() {
	apictl.PersistentFlags().StringVar(&LICENSE_KEY, "license-key", os.Getenv("AMBASSADOR_LICENSE_KEY"), "ambassador license key")
	keyfile, _ := defaultLicenseFile()
	apictl.PersistentFlags().StringVar(&LICENSE_FILE, "license-file", keyfile, "ambassador license file")
}

func keyCheck(cmd *cobra.Command, args []string) {
	var keysource string

	if LICENSE_KEY == "" {
		if LICENSE_FILE == "" {
			fmt.Printf("no license key or license key file specified")
			os.Exit(1)
		}
		key, err := ioutil.ReadFile(LICENSE_FILE)
		if err != nil {
			fmt.Printf("error reading license key from %s: %v\n", LICENSE_FILE, err)
			os.Exit(1)
		}
		LICENSE_KEY = strings.TrimSpace(string(key))
		keysource = LICENSE_FILE
	} else {
		if cmd.Flag("license-key").Changed {
			keysource = "key from command line"
		} else {
			keysource = "key from environment"
		}
	}

	var claims jwt.MapClaims

	token, err := jwt.ParseWithClaims(LICENSE_KEY, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("1234"), nil
	})

	id := fmt.Sprintf("%v", claims["id"])
	go func() {
		space, err := uuid.Parse("a4b394d6-02f4-11e9-87ca-f8344185863f")
		if err != nil {
			panic(err)
		}
		install_id := uuid.NewSHA1(space, []byte(id))
		data := make(map[string]interface{})
		data["application"] = "apictl"
		data["install_id"] = install_id.String()
		data["version"] = Version
		data["metadata"] = map[string]string{"id": id}
		body, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			panic(err)
		}
		_, err = http.Post("https://metriton.datawire.io/scout", "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Println(err)
		}
	}()

	if !token.Valid || err != nil {
		fmt.Printf("error validating %s: %v\n", keysource, err)
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
