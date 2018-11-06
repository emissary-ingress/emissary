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
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
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

type Query map[string]interface{}

func (q Query) Insecure() bool {
	val, ok := q["insecure"]
	return ok && val.(bool)
}

func (q Query) IsWebsocket() bool {
	return strings.HasPrefix(q.Url(), "ws:")
}

func (q Query) Url() string {
	return q["url"].(string)
}

func (q Query) Method() string {
	val, ok := q["method"]
	if ok {
		return val.(string)
	} else {
		return "GET"
	}
}

func (q Query) Headers() (result http.Header) {
	headers, ok := q["headers"]
	if ok {
		result = make(http.Header)
		for key, val := range headers.(map[string]interface{}) {
			result.Add(key, val.(string))
		}
	}
	return
}

type Result map[string]interface{}

func (q Query) Result() Result {
	val, ok := q["result"]
	if !ok {
		val = make(Result)
		q["result"] = val
	}
	return val.(Result)
}

func (q Query) CheckErr(err error) bool {
	if err != nil {
		log.Printf("%v: %v", q.Url(), err)
		q.Result()["error"] = err.Error()
		return true
	} else {
		return false
	}
}

func (q Query) AddResponse(resp *http.Response) {
	result := q.Result()
	result["status"] = resp.StatusCode
	result["headers"] = resp.Header
	body, err := ioutil.ReadAll(resp.Body)
	if !q.CheckErr(err) {
		log.Printf("%v: %v", q.Url(), resp.Status)
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

	var specs []Query

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
			result := query.Result()
			url := query.Url()

			if query.IsWebsocket() {
				c, resp, err := websocket.DefaultDialer.Dial(url, query.Headers())
				if query.CheckErr(err) { return }
				defer c.Close()
				query.AddResponse(resp)
				messages := query["messages"].([]interface{})
				for _, msg := range messages {
					err = c.WriteMessage(websocket.TextMessage, []byte(msg.(string)))
					if query.CheckErr(err) { return }
				}

				err = c.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if query.CheckErr(err) { return }

				answers := []string{}

				defer func() {
					result["messages"] = answers
				}()

				for {
					_, message, err := c.ReadMessage()
					if err != nil {
						if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
							query.CheckErr(err)
						}
						return
					} else {
						answers = append(answers, string(message))
					}
				}
			} else {
				req, err := http.NewRequest(query.Method(), url, nil)
				if query.CheckErr(err) { return }

				req.Header = query.Headers()
                host := req.Header.Get("Host")
                if host != "" {
                    req.Host = host
                }

				var cli *http.Client
				if query.Insecure() {
					cli = insecure_client
				} else {
					cli = client
				}

				resp, err := cli.Do(req)
				if query.CheckErr(err) { return }

				query.AddResponse(resp)
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
