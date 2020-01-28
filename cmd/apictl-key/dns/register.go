package dns

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
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

func init() {
	// Make sure our generator is truly random
	rand.Seed(time.Now().UnixNano())
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
	var registration struct {
		Email string
		Ip    string
	}

	// Decode the registration request:
	//   {"email":"alex@datawire.io","ip":"34.94.127.81"}
	err := decoder.Decode(&registration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: "ping-back" mechanism where we check the ambassador installation is publicly accessible.

	// Generate a random domain name
	domainName := fmt.Sprintf("%s%s", generateRandomName(), c.dnsRegistrationTLD)

	// Start a route53 session
	sess, err := session.NewSession()
	if err != nil {
		c.l.WithError(err).Error("error creating aws route53 session")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.l.Infof(result.String())

	// TODO: Keep track of this DNS and IP registration: save this info in a database somewhere

	// If all is good, return 200OK and the generated domain name in plain text.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(domainName))
}
