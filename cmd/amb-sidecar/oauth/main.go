package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/config"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/logger"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/secret"
	"github.com/datawire/apro/lib/licensekeys"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	argparser := &cobra.Command{
		Use:     os.Args[0],
		Version: Version,
		Run:     Main,
	}
	keycheck := licensekeys.InitializeCommandFlags(argparser.PersistentFlags(), Version)
	argparser.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		err := keycheck(cmd.PersistentFlags())
		if err == nil {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		time.Sleep(5 * 60 * time.Second)
		os.Exit(1)
	}
	err := argparser.Execute()
	if err != nil {
		os.Exit(2)
	}
}

func Main(flags *cobra.Command, args []string) {
	c := config.New()
	l := logger.New(c)
	s := secret.New(c, l)
	d := discovery.New(c)
	cl := client.NewRestClient(c.BaseURL)

	ct := &controller.Controller{
		Config: c,
		Logger: l.WithFields(logrus.Fields{"MAIN": "controller"}),
	}

	go ct.Watch()

	a := app.App{
		Config:     c,
		Logger:     l,
		Secret:     s,
		Discovery:  d,
		Controller: ct,
		Rest:       cl,
	}

	// Server
	if err := http.ListenAndServe(":8080", a.Handler()); err != nil {
		l.Fatal(err)
	}
}
