package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	var input, output string
	flag.StringVar(&input, "input", "", "input filename")
	flag.StringVar(&output, "output", "", "output filename")
	flag.Parse()

	var data []byte
	var err error

	if input == "" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(input)
	}
	if err != nil { panic(err) }

	var specs []map[string]interface{}

	err = json.Unmarshal(data, &specs)
	if err != nil { panic(err) }

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
	}
	client := &http.Client{
		Transport: tr,
		Timeout: time.Duration(10 * time.Second),
	}

	count := len(specs)
	queries := make(chan bool)
	for i := 0; i < count; i++ {
		go func(idx int) {
			query := specs[idx]
			result := make(map[string]interface{})
			query["result"] = result
			url := query["url"].(string)
			resp, err := client.Get(url)
			if err != nil {
				log.Printf("%v: %v", url, err)
				result["error"] = err.Error()
			} else {
				result["status"] = resp.StatusCode
				result["headers"] = resp.Header
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Printf("%v: %v", url, err)
					result["error"] = err.Error()
				} else {
					log.Printf("%v: %v", url, resp.Status)
					result["body"] = body
					var jsonBody interface{}
					err = json.Unmarshal(body, &jsonBody)
					if err == nil {
						result["json"] = jsonBody
					} else {
						result["text"] = string(body)
					}
				}
			}
			queries <- true
		}(i)
	}

	for i := 0 ; i < count; i++ {
		<- queries
	}

	bytes, err := json.MarshalIndent(specs, "", "  ")
	if err != nil {
		log.Print(err)
	} else if (output == "") {
		fmt.Print(string(bytes))
	} else {
		err = ioutil.WriteFile(output, bytes, 0644)
		if err != nil {
			log.Print(err)
		}
	}
}
