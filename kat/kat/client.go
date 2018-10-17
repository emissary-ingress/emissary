package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"
)

type semaphore chan bool

func Semaphore(n int) semaphore {
	sem := make(semaphore, n)
	for i := 0; i < n; i++ {
		sem.Release()
	}
	return sem
}

func (s semaphore) Acquire() {
	<- s
}

func (s semaphore) Release() {
	s <- true
}

func rlimit() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("Error getting rlimit:", err)
	} else {
		log.Println("Initial rlimit:", rLimit)
	}

	rLimit.Max = 999999
	rLimit.Cur = 999999
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("Error setting rlimit:", err)
	}

	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("Error getting rlimit:", err)
	} else {
		log.Println("Final rlimit", rLimit)
	}
}

func main() {
	rlimit()

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
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	insecure_tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	insecure_client := &http.Client{
		Transport: insecure_tr,
		Timeout: time.Duration(10 * time.Second),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	count := len(specs)
	queries := make(chan bool)

	limitStr := os.Getenv("KAT_QUERY_LIMIT")
	if limitStr == "" {
		limitStr = "25"
	}
	limit, err := strconv.Atoi(limitStr)

	sem := Semaphore(limit)

	for i := 0; i < count; i++ {
		go func(idx int) {
			sem.Acquire()
			defer func() {
				queries <- true
				sem.Release()
			}()

			query := specs[idx]
			result := make(map[string]interface{})
			query["result"] = result
			imethod, ok := query["method"]
			var method string
			if ok {
				method = imethod.(string)
			} else {
				method = "GET"
			}
			url := query["url"].(string)
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				log.Printf("%v: %v", url, err)
				result["error"] = err.Error()
				return
			}

			headers, ok := query["headers"]
			if ok {
				for key, val := range headers.(map[string]interface{}) {
					req.Header.Add(key, val.(string))
				}
			}

			insecure, ok := query["insecure"]
			var cli *http.Client
			if ok && insecure.(bool) {
				cli = insecure_client
			} else {
				cli = client
			}

			resp, err := cli.Do(req)
			if err != nil {
				log.Printf("%v: %v", url, err)
				result["error"] = err.Error()
				return
			}

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
