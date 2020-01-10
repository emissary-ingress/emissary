package main

import (
	"net/http"
)

func PluginMain(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{
		// Don't follow redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("https://en.wikipedia.org/wiki/Special:Random")
	if err != nil {
		// fail open?
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("X-Wikipedia", resp.Header.Get("Location"))
	w.WriteHeader(http.StatusOK)
}
