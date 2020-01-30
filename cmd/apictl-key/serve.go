package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/health"
	"github.com/datawire/apro/cmd/apictl-key/datasource"
	"github.com/datawire/apro/cmd/apictl-key/dns"
	"github.com/datawire/apro/lib/licensekeys"

	"github.com/datawire/ambassador/pkg/dlog"
)

var (
	hubspotKey         = os.Getenv("HUBSPOT_API_KEY")
	hostedZoneId       = os.Getenv("AWS_HOSTED_ZONE_ID")
	dnsRegistrationTLD = getEnv("DNS_REGISTRATION_TLD", ".edgestack.me")

	postgresUser     = getEnv("POSTGRES_USER", "aes")
	postgresPassword = getEnv("POSTGRES_PASSWORD", "aes")
	postgresDb       = getEnv("POSTGRES_DB", "aes")
	postgresHost     = getEnv("POSTGRES_HOST", "localhost")
	postgresPort     = getEnv("POSTGRES_PORT", "5432")
	postgresUrl      = fmt.Sprintf("psql://%s:%s@%s:%s/%s", postgresUser, postgresPassword, postgresHost, postgresPort, postgresDb)
)

func getEnv(name, fallback string) string {
	res := os.Getenv(name)
	if res == "" {
		res = fallback
	}

	return res
}

func encode(value interface{}) []byte {
	bytes, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return bytes
}

func getEdgectlStable() string {
	const fallback = "0.8.0"
	res, err := http.Get("https://s3.amazonaws.com/datawire-static-files/edgectl/stable.txt")
	if err != nil {
		return fallback
	}
	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return fallback
	}
	if res.StatusCode != http.StatusOK {
		return fallback
	}
	return strings.TrimSpace(string(data))
}

type HubspotUsageProbe struct {
	l *logrus.Logger
}

func (p *HubspotUsageProbe) Check() bool {
	url := fmt.Sprintf("https://api.hubapi.com/integrations/v1/limit/daily?hapikey=%s", hubspotKey)
	httpClient := &http.Client{
		Timeout: time.Second * 2,
	}
	// #nosec G107
	resp, err := httpClient.Get(url)
	if err != nil {
		p.l.WithError(err).Error("Request to hubspot API failed!")
		// TODO(alexgervais): We don't really want to crash our health probe if Hubspot is down.
		//                    Post AES launch, we should really instrument metrics about hubspot and build observability
		//                    around our API usage and downstream health.
		return true
	}
	defer resp.Body.Close()
	if resp != nil && resp.StatusCode != 200 {
		p.l.Error("Request to hubspot API resulted in ", resp.StatusCode)
		return true
	}
	p.l.Debug("hubspot API health check result: ", resp)
	return true
}

func init() {
	create := &cobra.Command{
		Use:   "serve-aes-backend",
		Short: "Start the AES-backend server",
		Long:  "Start the AES-backend server, allowing for registering license keys using the hubspot workflow; generating *.edgestack.me domain names; and more",
	}

	logrusFormatter := &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	}
	l := logrus.New()
	l.Formatter = logrusFormatter
	l.Level = logrus.DebugLevel
	l.Out = os.Stdout

	create.RunE = func(cmd *cobra.Command, args []string) error {
		if hubspotKey == "" {
			return errors.New("please set the HUBSPOT_API_KEY environment variable")
		}
		if hostedZoneId == "" {
			return errors.New("please set the AWS_HOSTED_ZONE_ID environment variable")
		}

		router := mux.NewRouter()

		// Liveness and Readiness probes
		healthprobe := health.MultiProbe{
			Logger: dlog.WrapLogrus(l),
		}
		healthprobe.RegisterProbe("static", &health.StaticProbe{Value: true})
		healthprobe.RegisterProbe("hubspot-usage", &HubspotUsageProbe{l: l})
		healthprobeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			healthy := healthprobe.Check()
			if healthy {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		router.HandleFunc("/signup/sys/readyz", healthprobeHandler)
		router.HandleFunc("/signup/sys/healthz", healthprobeHandler)

		router.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			if r.Method == http.MethodOptions {
				return
			}

			decoder := json.NewDecoder(r.Body)
			var s struct {
				Firstname string
				Lastname  string
				Email     string
				Phone     string
				Company   string
			}

			err := decoder.Decode(&s)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			l.Infof("New signup request from %s", s.Email)
			url := fmt.Sprintf("https://api.hubapi.com/contacts/v1/contact/createOrUpdate/email/%s/?hapikey=%s",
				s.Email,
				hubspotKey)

			// #nosec G107
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(encode(map[string]interface{}{
				"properties": []interface{}{
					map[string]string{
						"property": "trigger_aes_community_license_workflow",
						"value":    "",
					},
				},
			})))
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode > 300 {
				_, err = io.Copy(w, resp.Body)
				if err != nil {
					panic(err)
				}
				return
			}

			now := time.Now()
			expiresAt := now.Add(time.Duration(365) * 24 * time.Hour)
			communityLicenseClaims := licensekeys.NewCommunityLicenseClaims()
			metadata := map[string]string{}
			licenseKey := createTokenString(false, s.Email, s.Email, communityLicenseClaims.EnabledFeatures, communityLicenseClaims.EnforcedLimits, metadata, now, expiresAt)
			// #nosec G107
			resp, err = http.Post(url, "application/json", bytes.NewBuffer(encode(map[string]interface{}{
				"properties": []interface{}{
					map[string]string{
						"property": "trigger_aes_community_license_workflow",
						"value":    "yes",
					},
					map[string]string{
						"property": "aes_community_license_key",
						"value":    licenseKey,
					},
				},
			})))
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				panic(err)
			}
		})

		pgURL, err := url.Parse(postgresUrl)
		if err != nil {
			panic(err)
		}
		ds, err := datasource.NewDatasource(l, pgURL)
		if err != nil {
			panic(err)
		}

		dnsClient := dns.NewController(l, hostedZoneId, dnsRegistrationTLD, ds)
		router.HandleFunc("/register-domain", dnsClient.ServeHTTP)

		router.HandleFunc("/downloads/darwin/edgectl", func(w http.ResponseWriter, r *http.Request) {
			version := getEdgectlStable()
			url := fmt.Sprintf("https://datawire-static-files.s3.amazonaws.com/edgectl/%s/darwin/amd64/edgectl", version)
			http.Redirect(w, r, url, http.StatusFound) // 302
		})

		router.HandleFunc("/downloads/linux/edgectl", func(w http.ResponseWriter, r *http.Request) {
			version := getEdgectlStable()
			url := fmt.Sprintf("https://datawire-static-files.s3.amazonaws.com/edgectl/%s/linux/amd64/edgectl", version)
			http.Redirect(w, r, url, http.StatusFound) // 302
		})

		router.HandleFunc("/downloads/windows/edgectl", func(w http.ResponseWriter, r *http.Request) {
			version := getEdgectlStable()
			url := fmt.Sprintf("https://datawire-static-files.s3.amazonaws.com/edgectl/%s/windows/amd64/edgectl.exe", version)
			http.Redirect(w, r, url, http.StatusFound) // 302
		})

		router.HandleFunc("/downloads/windows/edgectl.exe", func(w http.ResponseWriter, r *http.Request) {
			version := getEdgectlStable()
			url := fmt.Sprintf("https://datawire-static-files.s3.amazonaws.com/edgectl/%s/windows/amd64/edgectl.exe", version)
			http.Redirect(w, r, url, http.StatusFound) // 302
		})

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		httpServer := &http.Server{
			Addr:    ":8080",
			Handler: router,
		}
		go func() {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				l.Fatalf("listen: %s\n", err)
			}
		}()
		l.Infof("Serving requests on %s", httpServer.Addr)

		<-done
		l.Info("Stopping server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			if err := ds.Close(); err != nil {
				l.WithError(err).Error("failed to close datasource connection")
			}
			cancel()
		}()

		if err := httpServer.Shutdown(ctx); err != nil {
			l.WithError(err).Fatal("http server shutdown failed")
		}

		return nil
	}

	argparser.AddCommand(create)
}
