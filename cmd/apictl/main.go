package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
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

const (
	VERSION = "0.0.7"
)

var (
	LICENSE_KEY   string
	LICENSE_FILE  string
	LICENSE_PAUSE = map[*cobra.Command]bool{
		watch: true,
	}
)

func init() {
	apictl.PersistentFlags().StringVarP(&LICENSE_KEY, "license-key", "", os.Getenv("AMBASSADOR_LICENSE_KEY"),
		"ambassador license key")
	keyfile := os.Getenv("AMBASSADOR_LICENSE_FILE")
	if keyfile == "" {
		usr, err := user.Current()
		if err == nil {
			keyfile = filepath.Join(usr.HomeDir, ".ambassador.key")
		}
	}
	apictl.PersistentFlags().StringVarP(&LICENSE_FILE, "license-file", "", keyfile, "ambassador license file")
}

func keyCheck(cmd *cobra.Command, args []string) {
	var keysource string

	if LICENSE_KEY == "" {
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
		data["version"] = VERSION
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
