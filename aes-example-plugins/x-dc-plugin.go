package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/hashicorp/consul/api"
)

func PluginMain(w http.ResponseWriter, r *http.Request) {

	// Get userID from URL param
	u, err := r.URL.Parse(r.URL.String())
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusOK)
		return
	}
	m, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusOK)
		return
	}
	uID := m["userID"]
	if len(uID) <= 0 {
		log.Println("userID not provided")
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Println("userID: ", uID[0])

	//Connect to consul api for key-value lookup
	log.Printf("Connecting to consul api\n")

	config := api.DefaultConfig()
	config.Address = "consul-server:8500"

	consul, err := api.NewClient(config)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Query for userID
	query := api.QueryOptions{}

	KVPair, qm, err := consul.KV().Get(uID[0], &query)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if qm != nil {
		log.Println(qm)
	}

	if KVPair == nil {
		log.Printf("userID: %s not found.\n", uID[0])
		w.WriteHeader(http.StatusOK)
		return
	}

	// Write the DC value to the X-DC header
	log.Printf("Directing userID %s to DC %s\n", uID[0], KVPair.Value)
	w.Header().Set("X-DC", string(KVPair.Value))

	w.WriteHeader(http.StatusOK)
}
