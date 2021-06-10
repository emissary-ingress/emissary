package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpc_echo_pb "github.com/datawire/ambassador/v2/pkg/api/kat"
)

// Should we output GRPCWeb debugging?
var debug_grpc_web bool // We set this value in main()   XXX This is a hack

// Limit concurrency

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

// Query and Result manipulation

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

// MinTLSVersion returns the minimun TLS protocol version.
func (q Query) MinTLSVersion() uint16 {
	switch q["minTLSv"].(string) {
	case "v1.0":
		return tls.VersionTLS10
	case "v1.1":
		return tls.VersionTLS11
	case "v1.2":
		return tls.VersionTLS12
	case "v1.3":
		return tls.VersionTLS13
	default:
		return 0
	}
}

// MaxTLSVersion returns the maximum TLS protocol version.
func (q Query) MaxTLSVersion() uint16 {
	switch q["maxTLSv"].(string) {
	case "v1.0":
		return tls.VersionTLS10
	case "v1.1":
		return tls.VersionTLS11
	case "v1.2":
		return tls.VersionTLS12
	case "v1.3":
		return tls.VersionTLS13
	default:
		return 0
	}
}

// CipherSuites returns the list of configured Cipher Suites
func (q Query) CipherSuites() []uint16 {
	val, ok := q["cipherSuites"]
	if !ok {
		return []uint16{}
	}
	cs := []uint16{}
	for _, s := range val.([]interface{}) {
		switch s.(string) {
		// TLS 1.0 - 1.2 cipher suites.
		case "TLS_RSA_WITH_RC4_128_SHA":
			cs = append(cs, tls.TLS_RSA_WITH_RC4_128_SHA)
		case "TLS_RSA_WITH_3DES_EDE_CBC_SHA":
			cs = append(cs, tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA)
		case "TLS_RSA_WITH_AES_128_CBC_SHA":
			cs = append(cs, tls.TLS_RSA_WITH_AES_128_CBC_SHA)
		case "TLS_RSA_WITH_AES_256_CBC_SHA":
			cs = append(cs, tls.TLS_RSA_WITH_AES_256_CBC_SHA)
		case "TLS_RSA_WITH_AES_128_CBC_SHA256":
			cs = append(cs, tls.TLS_RSA_WITH_AES_128_CBC_SHA256)
		case "TLS_RSA_WITH_AES_128_GCM_SHA256":
			cs = append(cs, tls.TLS_RSA_WITH_AES_128_GCM_SHA256)
		case "TLS_RSA_WITH_AES_256_GCM_SHA384":
			cs = append(cs, tls.TLS_RSA_WITH_AES_256_GCM_SHA384)
		case "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":
			cs = append(cs, tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA)
		case "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":
			cs = append(cs, tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA)
		case "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":
			cs = append(cs, tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA)
		case "TLS_ECDHE_RSA_WITH_RC4_128_SHA":
			cs = append(cs, tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA)
		case "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":
			cs = append(cs, tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA)
		case "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":
			cs = append(cs, tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA)
		case "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":
			cs = append(cs, tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA)
		case "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":
			cs = append(cs, tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256)
		case "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":
			cs = append(cs, tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256)
		case "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":
			cs = append(cs, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)
		case "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":
			cs = append(cs, tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256)
		case "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":
			cs = append(cs, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384)
		case "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":
			cs = append(cs, tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384)
		case "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":
			cs = append(cs, tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305)
		case "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":
			cs = append(cs, tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305)

		// TLS 1.3 cipher suites are not tunable
		// TLS_RSA_WITH_RC4_128_SHA
		// TLS_ECDHE_RSA_WITH_RC4_128_SHA
		// TLS_ECDHE_ECDSA_WITH_RC4_128_SHA

		// TLS_FALLBACK_SCSV isn't a standard cipher suite but an indicator
		// that the client is doing version fallback. See RFC 7507.
		case "TLS_FALLBACK_SCSV":
			cs = append(cs, tls.TLS_FALLBACK_SCSV)
		default:
		}
	}
	return cs
}

// ECDHCurves returns the list of configured ECDH CurveIDs
func (q Query) ECDHCurves() []tls.CurveID {
	val, ok := q["ecdhCurves"]
	if !ok {
		return []tls.CurveID{}
	}
	cs := []tls.CurveID{}
	for _, s := range val.([]interface{}) {
		switch s.(string) {
		// TLS 1.0 - 1.2 cipher suites.
		case "CurveP256":
			cs = append(cs, tls.CurveP256)
		case "CurveP384":
			cs = append(cs, tls.CurveP384)
		case "CurveP521":
			cs = append(cs, tls.CurveP521)
		case "X25519":
			cs = append(cs, tls.X25519)
		default:
		}
	}
	return cs
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

// Body returns an io.Reader for the base64 encoded body supplied in
// the query.
func (q Query) Body() io.Reader {
	body, ok := q["body"]
	if ok {
		buf, err := base64.StdEncoding.DecodeString(body.(string))
		if err != nil {
			panic(err)
		}
		return bytes.NewReader(buf)
	} else {
		return nil
	}
}

// GrpcType returns the query's grpc_type field or the empty string.
func (q Query) GrpcType() string {
	val, ok := q["grpc_type"]
	if ok {
		return val.(string)
	}
	return ""
}

// Cookies returns a slice of http.Cookie objects populated with any cookies
// passed in as part of the query.
func (q Query) Cookies() (result []http.Cookie) {
	result = []http.Cookie{}
	cookies, ok := q["cookies"]
	if ok {
		for _, c := range cookies.([]interface{}) {
			cookie := http.Cookie{
				Name:  c.(map[string]interface{})["name"].(string),
				Value: c.(map[string]interface{})["value"].(string),
			}
			result = append(result, cookie)
		}
	}
	return result
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

// DecodeGrpcWebTextBody treats the body as a series of base64-encode chunks. It
// returns the decoded proto and trailers.
func DecodeGrpcWebTextBody(body []byte) ([]byte, http.Header, error) {
	// First, decode all the base64 stuff coming in. An annoyance here
	// is that while the data coming over the wire are encoded in
	// multiple chunks, we can't rely on seeing that framing when
	// decoding: a chunk that's the right length to not need any base-64
	// padding will just run into the next chunk.
	//
	// So we loop to grab all the chunks, but we just serialize it into
	// a single raw byte array.

	var raw []byte

	cycle := 0

	for {
		if debug_grpc_web {
			log.Printf("%v: base64 body '%v'", cycle, body)
		}

		cycle++

		if len(body) <= 0 {
			break
		}

		chunk := make([]byte, base64.StdEncoding.DecodedLen(len(body)))
		n, err := base64.StdEncoding.Decode(chunk, body)

		if err != nil && n <= 0 {
			log.Printf("Failed to process body: %v\n", err)
			return nil, nil, err
		}

		raw = append(raw, chunk[:n]...)

		consumed := base64.StdEncoding.EncodedLen(n)

		body = body[consumed:]
	}

	// Next up, we need to split this into protobuf data and trailers. We
	// do this using grpc-web framing information for this -- each frame
	// consists of one byte of type, four bytes of length, then the data
	// itself.
	//
	// For our use case here, a type of 0 is the protobuf frame, and a type
	// of 0x80 is the trailers.

	trailers := make(http.Header) // the trailers will get saved here
	var proto []byte              // this is what we hand off to protobuf decode

	var frame_start, frame_len uint32
	var frame_type byte
	var frame []byte

	frame_start = 0

	if debug_grpc_web {
		log.Printf("starting frame split, len %v: %v", len(raw), raw)
	}

	for (frame_start + 5) < uint32(len(raw)) {
		frame_type = raw[frame_start]
		frame_len = binary.BigEndian.Uint32(raw[frame_start+1 : frame_start+5])

		frame = raw[frame_start+5 : frame_start+5+frame_len]

		if (frame_type & 128) > 0 {
			// Trailers frame
			if debug_grpc_web {
				log.Printf("  trailers @%v (len %v, type %v) %v - %v", frame_start, frame_len, frame_type, len(frame), frame)
			}

			lines := strings.Split(string(frame), "\n")

			for _, line := range lines {
				split := strings.SplitN(strings.TrimSpace(line), ":", 2)
				if len(split) == 2 {
					key := strings.TrimSpace(split[0])
					value := strings.TrimSpace(split[1])
					trailers.Add(key, value)
				}
			}
		} else {
			// Protobuf frame
			if debug_grpc_web {
				log.Printf("  protobuf @%v (len %v, type %v) %v - %v", frame_start, frame_len, frame_type, len(frame), frame)
			}

			proto = frame
		}

		frame_start += frame_len + 5
	}

	return proto, trailers, nil
}

// AddResponse populates a query's result with data from the query's HTTP
// response object.
//
// This is not called for websockets or real GRPC. It _is_ called for
// GRPC-bridge, GRPC-web, and (of course) HTTP(s).
func (q Query) AddResponse(resp *http.Response) {
	result := q.Result()
	result["status"] = resp.StatusCode
	result["headers"] = resp.Header

	headers := result["headers"].(http.Header)

	if headers != nil {
		// Copy in the client's start date.
		cstart := q["client-start-date"]

		// We'll only have a client-start-date if we're doing plain old HTTP, at
		// present -- so not for WebSockets or gRPC or the like. Don't try to
		// save the start and end dates if we have no start date.
		if cstart != nil {
			headers.Add("Client-Start-Date", q["client-start-date"].(string))

			// Add the client's end date.
			headers.Add("Client-End-Date", time.Now().Format(time.RFC3339Nano))
		}
	}

	if resp.TLS != nil {
		result["tls_version"] = resp.TLS.Version
		result["tls"] = resp.TLS.PeerCertificates
		result["cipher_suite"] = resp.TLS.CipherSuite
	}
	body, err := ioutil.ReadAll(resp.Body)
	if !q.CheckErr(err) {
		log.Printf("%v: %v", q.URL(), resp.Status)
		result["body"] = body
		if q.GrpcType() != "" && len(body) > 5 {
			if q.GrpcType() == "web" {
				// This is the GRPC-web case. Go forth and decode the base64'd
				// GRPC-web body madness.
				decodedBody, trailers, err := DecodeGrpcWebTextBody(body)
				if q.CheckErr(err) {
					log.Printf("Failed to decode grpc-web-text body: %v", err)
					return
				}
				body = decodedBody

				if debug_grpc_web {
					log.Printf("decodedBody '%v'", body)
				}

				for key, values := range trailers {
					for _, value := range values {
						headers.Add(key, value)
					}
				}

			} else {
				// This is the GRPC-bridge case -- throw away the five-byte type/length
				// framing at the start, and just leave the protobuf.
				body = body[5:]
			}

			response := &grpc_echo_pb.EchoResponse{}
			err := proto.Unmarshal(body, response)
			if q.CheckErr(err) {
				log.Printf("Failed to unmarshal proto: %v", err)
				return
			}
			result["text"] = response // q.r.json needs a different format
			return
		}
		var jsonBody interface{}
		err = json.Unmarshal(body, &jsonBody)
		if err == nil {
			result["json"] = jsonBody
		} else {
			result["text"] = string(body)
		}
	}
}

// Request processing

// ExecuteWebsocketQuery handles Websocket queries
func ExecuteWebsocketQuery(query Query) {
	url := query.URL()
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

	result := query.Result()
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
}

// GetGRPCReqBody returns the body of the HTTP request using the
// HTTP/1.1-gRPC bridge format as described in the Envoy docs
// https://www.envoyproxy.io/docs/envoy/v1.9.0/configuration/http_filters/grpc_http1_bridge_filter
func GetGRPCReqBody() (*bytes.Buffer, error) {
	// Protocol:
	// 	. 1 byte of zero (not compressed).
	// 	. network order (big-endian) of proto message length.
	// 	. serialized proto message.
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, uint8(0)); err != nil {
		log.Printf("error when packing first byte: %v", err)
		return nil, err
	}

	m := &grpc_echo_pb.EchoRequest{}
	m.Data = "foo"

	pbuf := &proto.Buffer{}
	if err := pbuf.Marshal(m); err != nil {
		log.Printf("error when serializing the gRPC message: %v", err)
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(pbuf.Bytes()))); err != nil {
		log.Printf("error when packing message length: %v", err)
		return nil, err
	}

	for i := 0; i < len(pbuf.Bytes()); i++ {
		if err := binary.Write(buf, binary.BigEndian, uint8(pbuf.Bytes()[i])); err != nil {
			log.Printf("error when packing message: %v", err)
			return nil, err
		}
	}

	return buf, nil
}

// CallRealGRPC handles real gRPC queries, i.e. queries that use the normal gRPC
// generated code and the normal HTTP/2-based transport.
func CallRealGRPC(query Query) {
	qURL, err := url.Parse(query.URL())
	if query.CheckErr(err) {
		log.Printf("grpc url parse failed: %v", err)
		return
	}

	const requiredPath = "/echo.EchoService/Echo"
	if qURL.Path != requiredPath {
		query.Result()["error"] = fmt.Sprintf("GRPC path %s is not %s", qURL.Path, requiredPath)
		return
	}

	dialHost := qURL.Host
	if !strings.Contains(dialHost, ":") {
		// There is no port number in the URL, but grpc.Dial wants host:port.
		if qURL.Scheme == "https" {
			dialHost = dialHost + ":443"
		} else {
			dialHost = dialHost + ":80"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Dial runs in the background and thus always appears to succeed. If you
	// pass grpc.WithBlock() to make it wait for a connection, failures just hit
	// the deadline rather than returning a useful error like "no such host" or
	// "connection refused" or whatever. Perhaps they are considered "transient"
	// and there's some retry logic we need to turn off. Anyhow, we don't pass
	// grpc.WithBlock(), instead letting the error happen at the request below.
	// This makes useful error messages visible in most cases.
	var dialOptions []grpc.DialOption
	if qURL.Scheme != "https" {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}
	conn, err := grpc.DialContext(ctx, dialHost, dialOptions...)
	if query.CheckErr(err) {
		log.Printf("grpc dial failed: %v", err)
		return
	}
	defer conn.Close()

	client := grpc_echo_pb.NewEchoServiceClient(conn)
	request := &grpc_echo_pb.EchoRequest{Data: "real gRPC"}

	// Prepare outgoing headers, which are passed via Context
	md := metadata.MD{}
	headers, ok := query["headers"]
	if ok {
		for key, val := range headers.(map[string]interface{}) {
			md.Set(key, val.(string))
		}
	}
	ctx = metadata.NewOutgoingContext(ctx, md)

	response, err := client.Echo(ctx, request)
	stat, ok := status.FromError(err)
	if !ok { // err is not nil and not a grpc Status
		query.CheckErr(err)
		log.Printf("grpc echo request failed: %v", err)
		return
	}

	// It's hard to tell the difference between a failed connection and a
	// successful connection that set an error code. We'll use the
	// heuristic that DNS errors and Connection Refused both appear to
	// return code 14 (Code.Unavailable).
	grpcCode := int(stat.Code())
	if grpcCode == 14 {
		query.CheckErr(err)
		log.Printf("grpc echo request connection failed: %v", err)
		return
	}

	// Now process the response and synthesize the requisite result values.
	// Note: Don't set result.body to anything that cannot be decoded as base64,
	// or the kat harness will fail.
	resHeader := make(http.Header)
	resHeader.Add("Grpc-Status", fmt.Sprint(grpcCode))
	resHeader.Add("Grpc-Message", stat.Message())

	result := query.Result()
	result["headers"] = resHeader
	result["body"] = ""
	result["status"] = 200
	if err == nil {
		result["text"] = response // q.r.json needs a different format
	}

	// Stuff that's not available:
	// - query.result.status (the HTTP status -- synthesized as 200)
	// - query.result.headers (the HTTP response headers -- we're just putting
	//   in grpc-status and grpc-message as the former is required by the
	//   tests and the latter can be handy)
	// - query.result.body (the raw HTTP body)
	// - query.result.json or query.result.text (the parsed HTTP body -- we're
	//   emitting the full EchoResponse object in the text field)
}

// ExecuteQuery constructs the appropriate request, executes it, and records the
// response and related information in query.result.
func ExecuteQuery(query Query) {
	// Websocket stuff is handled elsewhere
	if query.IsWebsocket() {
		ExecuteWebsocketQuery(query)
		return
	}

	// Real gRPC is handled elsewhere
	if query.GrpcType() == "real" {
		CallRealGRPC(query)
		return
	}

	// Prepare an http.Transport with customized TLS settings.
	transport := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{},
	}
	if query.Insecure() {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}
	if caCert := query.CACert(); len(caCert) > 0 {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caCert))
		transport.TLSClientConfig.RootCAs = caCertPool
	}
	if query.ClientCert() != "" || query.ClientKey() != "" {
		clientCert, err := tls.X509KeyPair([]byte(query.ClientCert()), []byte(query.ClientKey()))
		if err != nil {
			log.Fatal(err)
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}
	}
	if query.MinTLSVersion() != 0 {
		transport.TLSClientConfig.MinVersion = query.MinTLSVersion()
	}
	if query.MaxTLSVersion() != 0 {
		transport.TLSClientConfig.MaxVersion = query.MaxTLSVersion()
	}
	if len(query.CipherSuites()) > 0 {
		transport.TLSClientConfig.CipherSuites = query.CipherSuites()
	}
	if len(query.ECDHCurves()) > 0 {
		transport.TLSClientConfig.CurvePreferences = query.ECDHCurves()
	}

	// Prepare the HTTP request
	var body io.Reader
	method := query.Method()
	if query.GrpcType() != "" {
		// Perform special handling for gRPC-bridge and gRPC-web
		buf, err := GetGRPCReqBody()
		if query.CheckErr(err) {
			log.Printf("gRPC buffer error: %v", err)
			return
		}
		if query.GrpcType() == "web" {
			result := make([]byte, base64.StdEncoding.EncodedLen(buf.Len()))
			base64.StdEncoding.Encode(result, buf.Bytes())
			buf = bytes.NewBuffer(result)
		}
		body = buf
		method = "POST"
	} else {
		body = query.Body()
	}
	req, err := http.NewRequest(method, query.URL(), body)
	if query.CheckErr(err) {
		log.Printf("request error: %v", err)
		return
	}
	req.Header = query.Headers()
	for _, cookie := range query.Cookies() {
		req.AddCookie(&cookie)
	}

	// Save the client's start date.
	query["client-start-date"] = time.Now().Format(time.RFC3339Nano)

	// Handle host and SNI
	host := req.Header.Get("Host")
	if host != "" {
		if query.SNI() {
			transport.TLSClientConfig.ServerName = host
		}
		req.Host = host
	}

	// Perform the request and save the results.
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(10 * time.Second),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if query.CheckErr(err) {
		return
	}
	query.AddResponse(resp)
}

type Args struct {
	input  string
	output string
}

func parseArgs(rawArgs ...string) Args {
	var args Args
	flagset := flag.NewFlagSet("kat-client", flag.ExitOnError)
	flagset.StringVar(&args.input, "input", "", "input filename")
	flagset.StringVar(&args.output, "output", "", "output filename")
	flagset.Parse(rawArgs)
	return args
}

func main() {
	debug_grpc_web = false

	rlimit()

	args := parseArgs(os.Args[1:]...)

	var data []byte
	var err error

	// Read input file
	if args.input == "" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(args.input)
	}
	if err != nil {
		panic(err)
	}

	// Parse input file
	var specs []Query
	err = json.Unmarshal(data, &specs)
	if err != nil {
		panic(err)
	}

	// Prep semaphore to limit concurrency
	limitStr := os.Getenv("KAT_QUERY_LIMIT")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 25
	}
	sem := NewSemaphore(limit)

	// Launch queries concurrently
	count := len(specs)
	queries := make(chan bool)
	for i := 0; i < count; i++ {
		go func(idx int) {
			sem.Acquire()
			defer func() {
				queries <- true
				sem.Release()
			}()
			ExecuteQuery(specs[idx])
		}(i)
	}

	// Wait for all the answers
	for i := 0; i < count; i++ {
		<-queries
	}

	// Generate the output file
	bytes, err := json.MarshalIndent(specs, "", "  ")
	if err != nil {
		log.Print(err)
	} else if args.output == "" {
		fmt.Print(string(bytes))
	} else {
		err = ioutil.WriteFile(args.output, bytes, 0644)
		if err != nil {
			log.Print(err)
		}
	}
}
