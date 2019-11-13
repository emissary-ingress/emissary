package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	devportalcontent "github.com/datawire/apro/cmd/amb-sidecar/devportal/content"
	devportalserver "github.com/datawire/apro/cmd/amb-sidecar/devportal/server"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter/mocks"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func parse(urlStr string) *url.URL {
	url, err := url.Parse(urlStr)
	if err != nil {
		panic(err)
	}
	return url
}

var devportalCmd = &cobra.Command{
	Use:  "local-devportal [command]",
	Long: "Local devportal version " + Version,
}

var serveCmd = &cobra.Command{
	Use:   "serve [devportal-content-dir]",
	Short: "serve the specified directory or git URL [default .]",
	Run:   serve,
}

var branch string
var path string

func init() {
	devportalCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&branch, "branch", "master", "Branch to checkout when cloning a git URL")
	serveCmd.Flags().StringVar(&path, "path", "/", "Subdirectory to serve within the specified git URL")
}

func main() {
	if err := devportalCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func serve(cmd *cobra.Command, args []string) {

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cwdURL, err := url.Parse(cwd + "/")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	contentURL, err := cwdURL.Parse(dir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	config := types.Config{
		AmbassadorAdminURL:     parse("http://localhost:8877/"),
		AmbassadorInternalURL:  parse("http://localhost:8877/"),
		AmbassadorExternalURL:  parse("http://localhost:8877/"),
		DevPortalPollInterval:  2000 * time.Second,
		DevPortalContentURL:    contentURL,
		DevPortalContentBranch: branch,
		DevPortalContentSubdir: path,
	}

	content, err := devportalcontent.NewContent(
		config.DevPortalContentURL,
		config.DevPortalContentBranch,
		config.DevPortalContentSubdir)
	if err != nil {
		log.Fatal(err)
	}

	yml, err := content.Fs().Open("devportal.yaml")
	if yml != nil {
		defer yml.Close()
	}
	if os.IsNotExist(err) {
		root, err := content.Fs().Glob("/", "*")
		fmt.Printf("\nPlease specify a devportal checkout. %v\n\n", err)
		fmt.Printf("Looking at: %v\n\n", root)
		os.Exit(1)
	}

	docs := "/local-devportal"
	limiter := mocks.NewMockLimiter()
	server := devportalserver.NewServer(docs, content, limiter)

	amb := newMockAmbassador()
	amb.addMapping("default", "ambassador-devportal", docs, server.Router())
	amb.addMapping("default", "ambassador-devportal-api", "/openapi", server.Router())
	amb.addMapping("ns1", "example-a", "/example-a", newSampleService("/example-a", true))
	amb.addMapping("ns2", "example-b", "/example-b", newSampleService("/example-b", true))
	amb.addMapping("ns1", "example-c", "/example-c", newSampleService("/example-c", false))

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(rsp http.ResponseWriter, rq *http.Request) {
		if rq.URL.Path == "/" || rq.URL.Path == "" {
			location := "http://localhost:8877" + docs + "/"
			log.Infof("Redirecting from %s to %s", rq.URL, location)
			rsp.Header().Add("Location", location)
			rsp.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			amb.ServeHTTP(rsp, rq)
		}

	})

	group, ctx := errgroup.WithContext(context.Background())

	group.Go(func() error {
		return http.ListenAndServe("0.0.0.0:8877", router)
	})

	group.Go(func() error {
		//time.Sleep(100 * time.Millisecond)
		fetcher := devportalserver.NewFetcher(server, devportalserver.HTTPGet, server.KnownServices(), config)
		fetcher.Run(ctx)
		return nil
	})

	log.Fatal(group.Wait())
}
