package dns

import (
	"encoding/json"
	"fmt"
	"github.com/datawire/apro/cmd/apictl-key/datasource"
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
	datasource         datasource.Datasource
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

func NewController(l *logrus.Logger, hostedZoneId string, dnsRegistrationTLD string, datasource datasource.Datasource) http.Handler {
	return &dnsclient{
		l:                  l,
		hostedZoneId:       hostedZoneId,
		dnsRegistrationTLD: dnsRegistrationTLD,
		datasource:         datasource,
	}
}

func (c *dnsclient) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteIp, _, _ := net.SplitHostPort(r.RemoteAddr)

	decoder := json.NewDecoder(r.Body)

	// Decode the registration request:
	//   {"email":"alex@datawire.io","ip":"34.94.127.81"}
	var registration registration
	if err := decoder.Decode(&registration); err != nil {
		c.l.WithError(err).Warn("failed to parse http request")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check the Email is present. We don't actually check it's valid tho
	if registration.Email == "" {
		c.l.Warn("email is missing from registration request")
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	// Check the IP is public and is serving an AES install
	if err := c.verifyIPIsReady(registration.Ip); err != nil {
		c.l.WithError(err).Warn("failed to verify ip is in ready state")
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	}

	var domainName string
	attempt := 1
	const maxAttempts = 5
	for {
		// Generate a random domain name
		domainName = fmt.Sprintf("%s%s", generateRandomName(), c.dnsRegistrationTLD)

		// Validate it is not already registered
		exists, err := c.datasource.DomainExists(domainName)
		if err != nil {
			c.l.WithError(err).Error("failed to verify the domain was not already registered")
			http.Error(w, "domain name registration failed", http.StatusInternalServerError)
			return
		}
		if !exists {
			break
		} else if attempt == maxAttempts {
			c.l.Errorf("failed to generate a unique and unused domain name after %d attempts", attempt)
			http.Error(w, "domain name registration failed", http.StatusInternalServerError)
			return
		} else {
			attempt++
		}
	}

	// Do register a DNS entry
	if err := c.doRegister(domainName, registration.Ip); err != nil {
		c.l.WithError(err).Error("failed to register the dns record")
		http.Error(w, "domain name registration failed", http.StatusInternalServerError)
		return
	}

	// Save the registration information in database
	if err := c.datasource.AddDomain(domainName, registration.Ip, registration.Email, remoteIp); err != nil {
		c.l.WithError(err).Errorf("failed to persists the domain registration request; a dns record '%s' was registered and must be cleaned up manually", domainName)
		http.Error(w, "domain name registration failed", http.StatusInternalServerError)
		return
	}

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

	attempt := 1
	const maxAttempts = 5
	for {
		// GET http://{IP}/.well-known/acme-challenge/
		//  --> should return 404
		//  --> should return header "server: envoy"
		response, err := client.Get(fmt.Sprintf("http://%s/.well-known/acme-challenge/", ip))
		defer response.Body.Close()
		if err == nil && (response.StatusCode != 404 || response.Header.Get("server") != "envoy") {
			err = fmt.Errorf("failed to validate the target ip is running AES")
		}
		if err != nil {
			c.l.WithError(err).Warnf("error while attempting to validate the target IP %d/%d", attempt, maxAttempts)
			// Retry a few times... it's a new installation of AES and initialization might not be complete
			if attempt == maxAttempts {
				return err
			}
			attempt++
			// Don't sleep; we need to make sure we can handle the original HTTP request in <30 seconds
		} else {
			return nil
		}
	}
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

func (c *dnsclient) doRegister(domainName string, ip string) error {
	// Start a route53 session
	sess, err := session.NewSession()
	if err != nil {
		c.l.WithError(err).Error("error creating aws route53 session")
		return err
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
								Value: aws.String(ip),
							},
						},
						TTL:  aws.Int64(5),
						Type: aws.String("A"),
					},
				},
				{
					Action: aws.String("CREATE"), // Create!, don't update...
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(fmt.Sprintf("*.%s", domainName)), // Saving a second wildcard record, helping bust dns caches
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(ip),
							},
						},
						TTL:  aws.Int64(5),
						Type: aws.String("A"),
					},
				},
			},
		},
		HostedZoneId: aws.String(c.hostedZoneId),
	}

	// Save the route53 records
	result, err := r53.ChangeResourceRecordSets(input)
	if err != nil {
		c.l.WithError(err).Error("error creating dns entry")
		return err
	}
	c.l.Infof(result.String())

	return nil
}
