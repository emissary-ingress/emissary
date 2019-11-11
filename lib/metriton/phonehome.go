package metriton

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/jpillora/backoff"
	"net/http"
	"os"
	"time"

	"github.com/datawire/apro/lib/licensekeys"

	"github.com/google/uuid"
)

type MetritonResponse struct {
	IsHardLimit bool `json:"hard_limit"`
}

func PhoneHomeEveryday(claims *licensekeys.LicenseClaimsLatest, limiter *limiter.LimiterImpl, application, version string) {
	// Phone home every 12 hours
	phoneHomeTicker := time.NewTicker(12 * time.Hour)
	for range phoneHomeTicker.C {
		go PhoneHome(claims, limiter, application, version)
	}
}

func PhoneHome(claims *licensekeys.LicenseClaimsLatest, limiter *limiter.LimiterImpl, application, version string) {
	// We might be really excited to phone home, but let the limiters startup process attempt to initialize the Redis connection...
	if limiter != nil {
		time.Sleep(15 * time.Second)
	}
	fmt.Println("Calling Metriton")

	b := &backoff.Backoff{
		Min:    5 * time.Minute,
		Max:    8 * time.Hour,
		Jitter: true,
		Factor: 2,
	}
	for {
		err := phoneHome(claims, limiter, application, version)
		if err != nil {
			d := b.Duration()
			if b.Attempt() >= 8 {
				fmt.Printf("Metriton error after %d attemps: %v\n", int(b.Attempt()), err)
				b.Reset()
				break
			}
			fmt.Printf("Metriton error, retrying in %s: %v\n", d, err)
			time.Sleep(d)
			continue
		}
		b.Reset()
		break
	}
}

func phoneHome(claims *licensekeys.LicenseClaimsLatest, limiter *limiter.LimiterImpl, component, version string) error {
	data := prepareData(claims, limiter, component, version)

	if os.Getenv("SCOUT_DISABLE") != "" {
		fmt.Println("SCOUT_DISABLE, enforcing hard-limits")
		if limiter != nil {
			limiter.SetPhoneHomeHardLimits(true)
		}
		return nil
	}

	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}
	metritonEndpoint := "https://kubernaut.io/scout" // THIS CAN'T BE AN ENVIRONMENT VARIABLE, OR METRITON MIGHT BE HIJACKED
	resp, err := http.Post(metritonEndpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		if limiter != nil {
			fmt.Println("Metriton call was not a success... allow soft-limit")
			limiter.SetPhoneHomeHardLimits(false)
		}
		return err
	}
	defer resp.Body.Close()

	metritonResponse := MetritonResponse{}
	err = json.NewDecoder(resp.Body).Decode(&metritonResponse)
	if err != nil {
		if limiter != nil {
			fmt.Println("Metriton call was not a success... allow soft-limit")
			limiter.SetPhoneHomeHardLimits(false)
		}
		return err
	}
	if limiter != nil && metritonResponse.IsHardLimit {
		fmt.Println("Metriton is enforcing hard-limits")
		limiter.SetPhoneHomeHardLimits(true)
		return nil
	}
	return nil
}

func prepareData(claims *licensekeys.LicenseClaimsLatest, limiter *limiter.LimiterImpl, component, version string) map[string]interface{} {
	activeClaims := claims
	if limiter != nil && limiter.GetClaims() != nil {
		activeClaims = limiter.GetClaims()
	}

	// Make sure we log a message while calculating license usage, even if we don't actually call Metriton
	featuresDataSet := []map[string]interface{}{}
	for _, limitName := range licensekeys.ListKnownLimits() {
		limit, ok := licensekeys.ParseLimit(limitName)
		if ok && limiter != nil {
			limitValue := limiter.GetLimitValueAtPointInTime(&limit)
			usageValue := limiter.GetFeatureUsageValueAtPointInTime(&limit)
			featuresDataSet = append(featuresDataSet, map[string]interface{}{
				"name":  limitName,
				"usage": usageValue,
				"limit": limitValue,
			})
			if usageValue >= limitValue {
				fmt.Printf("You've reached the usage limits for your licensed feature %s usage (%d) limit (%d). Contact Datawire for a license key to remove limits https://www.getambassador.io/contact/\n",
					limitName, usageValue, limitValue)
			}
		}
	}
	customerID := ""
	if activeClaims != nil {
		customerID = activeClaims.CustomerID
	}
	customerContact := ""
	if activeClaims != nil {
		customerContact = activeClaims.CustomerEmail
	}
	namespace, err := uuid.Parse("a4b394d6-02f4-11e9-87ca-f8344185863f")
	if err != nil {
		panic(err)
	}

	return map[string]interface{}{
		"application": "aes",
		"install_id":  uuid.NewSHA1(namespace, []byte(customerID)).String(),
		"version":     version,
		"metadata": map[string]interface{}{
			"id":        customerID,
			"contact":   customerContact,
			"component": component,
			"features":  featuresDataSet,
		},
	}
}
