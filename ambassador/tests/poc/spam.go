package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
	}
	client := &http.Client{Transport: tr}

	n := 1
	for j := 0; j < n; j++ {
		count := 1000
		queries := make(chan bool)
		for i := 0; i < count; i++ {
			go func(id int) {
				
//				resp, err := client.Get("http://54.146.61.117:31646/HTTP-SimpleMapping-AddRequestHeaders/")
				resp, err := client.Get("http://ambassador-plain/HTTP-SimpleMapping-AddRequestHeaders/")
				var result bool
				if err != nil {
					log.Print(id, err)
					result = false
				} else {
					log.Print(id, resp.Status)
					resp.Body.Close()
					result = true
				}
				queries <- result
			}(i)
		}

		done := 0
		good := 0

		for ; done < count; {
			result := <- queries
			if result {
				good += 1
			}
			done += 1
		}
		log.Print("Good ", good, " Bad ", done - good)
	}
}
