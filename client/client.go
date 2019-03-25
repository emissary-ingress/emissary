package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	grpc_echo_pb "github.com/datawire/kat-backend/echo"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
)

// Semaphore is a counting semaphore that can be used to limit concurrency.
type Semaphore chan bool

// NewSemaphore returns a new Semaphore with the specified capacity.
func NewSemaphore(n int) Semaphore {
	sem := make(Semaphore, n)
	for i := 0; i < n; i++ {
		sem.Release()
	}
	return sem
}

// Acquire blocks until a slot/token is available.
func (s Semaphore) Acquire() {
	<-s
}

// Release returns a slot/token to the pool.
func (s Semaphore) Release() {
	s <- true
}

// rlimit frobnicates the interplexing beacon. Or maybe it reverses the polarity
// of the neutron flow. I'm not sure. FIXME.
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

// Query represents one kat query as read from the supplied input. It will be
// mutated to include results from that query.
type Query map[string]interface{}

// CACert returns the "ca_cert" field as a string or returns the empty string.
func (q Query) CACert() string {
	val, ok := q["ca_cert"]
	if ok {
		return val.(string)
	}
	return ""
}

// ClientCert returns the "client_cert" field as a string or returns the empty string.
func (q Query) ClientCert() string {
	val, ok := q["client_cert"]
	if ok {
		return val.(string)
	}
	return ""
}

// ClientKey returns the "client_key" field as a string or returns the empty string.
func (q Query) ClientKey() string {
	val, ok := q["client_key"]
	if ok {
		return val.(string)
	}
	return ""
}

// Insecure returns whether the query has a field called "insecure" whose value is true.
func (q Query) Insecure() bool {
	val, ok := q["insecure"]
	return ok && val.(bool)
}

// SNI returns whether the query has a field called "sni" whose value is true.
func (q Query) SNI() bool {
	val, ok := q["sni"]
	return ok && val.(bool)
}

// IsWebsocket returns whether the query's URL starts with "ws:".
func (q Query) IsWebsocket() bool {
	return strings.HasPrefix(q.URL(), "ws:")
}

// URL returns the query's URL.
func (q Query) URL() string {
	return q["url"].(string)
}

// Method returns the query's method or "GET" if unspecified.
func (q Query) Method() string {
	val, ok := q["method"]
	if ok {
		return val.(string)
	}
	return "GET"
}

// Headers returns the an http.Header object populated with any headers passed
// in as part of the query.
func (q Query) Headers() (result http.Header) {
	result = make(http.Header)
	headers, ok := q["headers"]
	if ok {
		for key, val := range headers.(map[string]interface{}) {
			result.Add(key, val.(string))
		}
	}
	return result
}

// IsGrpc checks if the request is to a gRPC service.
func (q Query) IsGrpc() bool {
	headers := q.Headers()
	key := textproto.CanonicalMIMEHeaderKey("content-type")
	for _, val := range headers[key] {
		if strings.Contains(strings.ToLower(val), "application/grpc") {
			return true
		}
	}
	return false
}

// Result represents the result of one kat query. Upon first access to a query's
// result field, the Result object will be created and added to the query.
type Result map[string]interface{}

// Result returns the query's result field as a Result object. If the field
// doesn't exist, a new Result object is created and placed in that field. If
// the field exists and contains something else, panic!
func (q Query) Result() Result {
	val, ok := q["result"]
	if !ok {
		val = make(Result)
		q["result"] = val
	}
	return val.(Result)
}

// CheckErr populates the query result with error information if an error is
// passed in (and logs the error).
func (q Query) CheckErr(err error) bool {
	if err != nil {
		log.Printf("%v: %v", q.URL(), err)
		q.Result()["error"] = err.Error()
		return true
	}
	return false
}

// AddResponse populates a query's result with data from the query's HTTP
// response object.
func (q Query) AddResponse(resp *http.Response) {
	result := q.Result()
	result["status"] = resp.StatusCode
	result["headers"] = resp.Header
	if resp.TLS != nil {
		result["tls"] = resp.TLS.PeerCertificates
	}
	body, err := ioutil.ReadAll(resp.Body)
	if !q.CheckErr(err) {
		log.Printf("%v: %v", q.URL(), resp.Status)
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
	if err != nil {
		panic(err)
	}

	var specs []Query

	err = json.Unmarshal(data, &specs)
	if err != nil {
		panic(err)
	}

	transport := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(10 * time.Second),
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

	sem := NewSemaphore(limit)

	for i := 0; i < count; i++ {
		go func(idx int) {
			sem.Acquire()
			defer func() {
				queries <- true
				sem.Release()
			}()

			query := specs[idx]
			result := query.Result()
			url := query.URL()

			insecureTransport := &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 30 * time.Second,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}

			caCert := query.CACert()
			if len(caCert) > 0 {

				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM([]byte(caCert))

				clientCert, err := tls.X509KeyPair([]byte(query.ClientCert()), []byte(query.ClientKey()))
				if err != nil {
					log.Fatal(err)
				}

				insecureTransport.TLSClientConfig.RootCAs = caCertPool
				insecureTransport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}
			}

			insecureClient := &http.Client{
				Transport: insecureTransport,
				Timeout:   time.Duration(10 * time.Second),
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			if query.IsWebsocket() {
				c, resp, err := websocket.DefaultDialer.Dial(url, query.Headers())
				if query.CheckErr(err) {
					return
				}
				defer c.Close()
				query.AddResponse(resp)
				messages := query["messages"].([]interface{})
				for _, msg := range messages {
					err = c.WriteMessage(websocket.TextMessage, []byte(msg.(string)))
					if query.CheckErr(err) {
						return
					}
				}

				err = c.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if query.CheckErr(err) {
					return
				}

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
					}
					answers = append(answers, string(message))
				}
			} else {
				var req *http.Request

				// Sets grpc-echo POST request.
				//
				// Protocol:
				// 	. 1 byte of zero (not compressed).
				// 	. network order (big-endian) of proto message length.
				// 	. serialized proto message.
				if query.IsGrpc() {
					buf := &bytes.Buffer{}
					if err := binary.Write(buf, binary.BigEndian, uint8(0)); err != nil {
						log.Printf("error when packing first byte: %v", err)
						return
					}

					m := &grpc_echo_pb.EchoRequest{}
					m.Data = "foo"

					pbuf := &proto.Buffer{}
					if err := pbuf.Marshal(m); err != nil {
						log.Printf("error when serealizing the gRPC message: %v", err)
						return
					}

					if err := binary.Write(buf, binary.BigEndian, uint32(len(pbuf.Bytes()))); err != nil {
						log.Printf("error when packing message length: %v", err)
						return
					}

					for i := 0; i < len(pbuf.Bytes()); i++ {
						if err := binary.Write(buf, binary.BigEndian, uint8(pbuf.Bytes()[i])); err != nil {
							log.Printf("error when packing message: %v", err)
							return
						}
					}

					req, err = http.NewRequest("POST", url, buf)
					if query.CheckErr(err) {
						log.Printf("grpc bridge request error: %v", err)
						return
					}

					for k, h := range query.Headers() {
						for _, v := range h {
							log.Printf("setting request header [ %s : %s ]", k, v)
							req.Header.Add(http.CanonicalHeaderKey(k), v)
						}
					}
				} else {
					req, err = http.NewRequest(query.Method(), url, nil)
					req.Header = query.Headers()
					if query.CheckErr(err) {
						return
					}
				}

				host := req.Header.Get("Host")
				if host != "" {
					if query.SNI() {
						if transport.TLSClientConfig == nil {
							transport.TLSClientConfig = &tls.Config{}
						}

						if insecureTransport.TLSClientConfig == nil {
							insecureTransport.TLSClientConfig = &tls.Config{}
						}

						insecureTransport.TLSClientConfig.ServerName = host
						transport.TLSClientConfig.ServerName = host
					}
					req.Host = host
				}

				var cli *http.Client
				if query.Insecure() {
					cli = insecureClient
				} else {
					cli = client
				}

				resp, err := cli.Do(req)
				if query.CheckErr(err) {
					return
				}

				query.AddResponse(resp)
			}
		}(i)
	}

	for i := 0; i < count; i++ {
		<-queries
	}

	bytes, err := json.MarshalIndent(specs, "", "  ")
	if err != nil {
		log.Print(err)
	} else if output == "" {
		fmt.Print(string(bytes))
	} else {
		err = ioutil.WriteFile(output, bytes, 0644)
		if err != nil {
			log.Print(err)
		}
	}
}
