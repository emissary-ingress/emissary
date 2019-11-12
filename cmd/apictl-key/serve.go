package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
)

var hubspotKey = os.Getenv("HUBSPOT_API_KEY")

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

func init() {
	create := &cobra.Command{
		Use:   "serve-aes-signup",
		Short: "Generate an AES license key and trigger a hubspot workflow",
	}

	create.RunE = func(cmd *cobra.Command, args []string) error {
		if hubspotKey == "" {
			return errors.New("please set the HUBSPOT_API_KEY environment variable")
		}

		http.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
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
			licenseKey := createTokenString(false, s.Email, s.Email, communityLicenseClaims.EnabledFeatures, communityLicenseClaims.EnforcedLimits, now, expiresAt)
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

		http.HandleFunc("/darwin/edgectl", func(w http.ResponseWriter, r *http.Request) {
			version := getEdgectlStable()
			url := fmt.Sprintf("https://datawire-static-files.s3.amazonaws.com/edgectl/%s/darwin/amd64/edgectl", version)
			http.Redirect(w, r, url, http.StatusFound) // 302
		})

		http.HandleFunc("/linux/edgectl", func(w http.ResponseWriter, r *http.Request) {
			version := getEdgectlStable()
			url := fmt.Sprintf("https://datawire-static-files.s3.amazonaws.com/edgectl/%s/linux/amd64/edgectl", version)
			http.Redirect(w, r, url, http.StatusFound) // 302
		})

		return http.ListenAndServe(":8080", nil)
	}

	argparser.AddCommand(create)
}
