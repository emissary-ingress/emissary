package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/lib/metriton"
)

var apictl = &cobra.Command{
	Use: "apictl [command]",
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

var licenseClaims *licensekeys.LicenseClaimsLatest

func init() {
	cmdContext := &licensekeys.LicenseContext{}
	if err := cmdContext.AddFlagsTo(apictl); err != nil {
		panic(err)
	}

	apictl.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true // https://github.com/spf13/cobra/issues/340
		if cmd.Name() == "help" {
			return
		}
		var err error
		licenseClaims, err = cmdContext.GetClaims()
		if err == nil {
			go metriton.PhoneHome(licenseClaims, nil, "apictl", Version)
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	apictl.Version = Version
	apictl.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
Copyright 2019 Datawire. All rights reserved.

Information about open source code used in this executable is found at
<https://s3.amazonaws.com/datawire-static-files/{{.Name}}/{{.Version}}/` + runtime.GOOS + `/` + runtime.GOARCH + `/{{.Name}}.opensource.tar.gz>.

Information about open source code used in the Docker image installed by
'apictl traffic initialize' is found in the '/traffic-proxy.opensource.tar.gz'
file in the 'quay.io/datawire/aes:traffic-proxy-{{.Version}}'
Docker image.

Information about open source code used in the Docker image installed by
'apictl traffic inject' is found in the '/app-sidecar.opensource.tar.gz'
file in the 'quay.io/datawire/aes:app-sidecar-{{.Version}}'
Docker image.
`)
}

func recoverFromCrash() {
	if r := recover(); r != nil {
		fmt.Println("---")
		fmt.Println("\nThe apictl command has crashed. Sorry about that!")
		fmt.Println(r)
	}
}

func main() {
	defer recoverFromCrash()
	err := apictl.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func die(err error, args ...interface{}) {
	if err != nil {
		if args != nil {
			fmt.Printf("%v: %v\n", err, args)
		} else {
			fmt.Println(err)
		}
		panic(err)
	}
}
