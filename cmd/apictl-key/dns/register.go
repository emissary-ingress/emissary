package dns

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

type dnsclient struct {
	l                  *logrus.Logger
	hostedZoneId       string
	dnsRegistrationTLD string
}

type registration struct {
	Email string
	Ip    string
}

var privateIPBlocks []*net.IPNet

func init() {
	// Make sure our generator is truly random
	rand.Seed(time.Now().UnixNano())

	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func NewController(l *logrus.Logger, hostedZoneId string, dnsRegistrationTLD string) http.Handler {
	return &dnsclient{
		l:                  l,
		hostedZoneId:       hostedZoneId,
		dnsRegistrationTLD: dnsRegistrationTLD,
	}
}

func (c *dnsclient) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	// Decode the registration request:
	//   {"email":"alex@datawire.io","ip":"34.94.127.81"}
	var registration registration
	err := decoder.Decode(&registration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check the IP is public and is serving an AES install
	err = c.verifyIPIsReady(registration.Ip)
	if err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	}

	// Do register a DNS entry
	domainName, err := c.doRegister(registration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Keep track of this DNS and IP registration: save this info in a database somewhere

	// If all is good, return 200OK and the generated domain name in plain text.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(domainName))
}

func (c *dnsclient) verifyIPIsReady(ip string) error {
	if !c.isPublicIP(ip) {
		return fmt.Errorf("ip address is not public")
	}

	var transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).DialContext,
	}
	var client = &http.Client{
		// 3s timeout
		Timeout:   3 * time.Second,
		Transport: transport,
		// Don't follow redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// GET http://{IP}/.well-known/acme-challenge/
	//  --> should return 404
	//  --> should return header "server: envoy"
	response, err := client.Get(fmt.Sprintf("http://%s/.well-known/acme-challenge/", ip))
	if err != nil {
		return err
	}
	if response.StatusCode != 404 || response.Header.Get("server") != "envoy" {
		return fmt.Errorf("failed to validate the target ip is running AES")
	}
	return nil
}

func (c *dnsclient) isPublicIP(ipString string) bool {
	ip := net.ParseIP(ipString)
	if ip == nil {
		// it's not even an IP!
		return false
	}
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		// local interfaces
		return false
	}
	for _, block := range privateIPBlocks {
		// private ip ranges
		if block.Contains(ip) {
			return false
		}
	}
	return true
}

func (c *dnsclient) doRegister(registration registration) (string, error) {
	// Generate a random domain name
	domainName := fmt.Sprintf("%s%s", generateRandomName(), c.dnsRegistrationTLD)

	// Start a route53 session
	sess, err := session.NewSession()
	if err != nil {
		c.l.WithError(err).Error("error creating aws route53 session")
		return "", err
	}
	r53 := route53.New(sess)

	// Create a route53 record set, associating the IP with random domain name
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"), // Create!, don't update...
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(domainName),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(registration.Ip),
							},
						},
						TTL:  aws.Int64(60),
						Type: aws.String("A"),
					},
					// TODO: Save a TXT record as well?
				},
			},
		},
		HostedZoneId: aws.String(c.hostedZoneId),
	}

	// Save the route53 records
	result, err := r53.ChangeResourceRecordSets(input)
	if err != nil {
		c.l.WithError(err).Error("error creating dns entry")
		return "", err
	}
	c.l.Infof(result.String())

	return domainName, nil
}
