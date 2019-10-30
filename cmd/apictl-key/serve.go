package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var hubspotKey = os.Getenv("HUBSPOT_API_KEY")

func encode(value interface{}) []byte {
	bytes, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return bytes
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
				panic(err)
			}

			url := fmt.Sprintf("https://api.hubapi.com/contacts/v1/contact/createOrUpdate/email/%s/?hapikey=%s",
				s.Email,
				hubspotKey)

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

			// XXX: what features does it get, how long does it last?
			now := time.Now()
			expiresAt := now.Add(time.Duration(365) * 24 * time.Hour)
			licenseKey := createTokenString(false, s.Email, s.Email, nil, nil, now, expiresAt)
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
		return http.ListenAndServe(":8080", nil)
	}

	argparser.AddCommand(create)
}
