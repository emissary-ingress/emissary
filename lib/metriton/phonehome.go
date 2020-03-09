package metriton

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jpillora/backoff"

	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/lib/licensekeys"
)

type MetritonResponse struct {
	IsHardLimit bool `json:"hard_limit"`
}

const phoneHomeEveryPeriod = 12 * time.Hour

// THIS CAN'T BE AN ENVIRONMENT VARIABLE, OR METRITON MIGHT BE HIJACKED
// Use "https://kubernaut.io/beta/scout" for testing purposes without polluting production data.
const metritonEndpoint = "https://kubernaut.io/scout"

func PhoneHomeEveryday(claims *licensekeys.LicenseClaimsLatest, limiter *limiter.LimiterImpl, application, version string) {
	// Phone home every X period
	phoneHomeTicker := time.NewTicker(phoneHomeEveryPeriod)
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
				fmt.Printf("Metriton error after %d attempts: %v\n", int(b.Attempt()), err)
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
			maxUsageValue := limiter.GetFeatureMaxUsageValue(&limit)
			featuresDataSet = append(featuresDataSet, map[string]interface{}{
				"name":      limitName,
				"usage":     usageValue,
				"max_usage": maxUsageValue,
				"limit":     limitValue,
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

	installID, err := uuid.Parse(
		getenvDefault("AMBASSADOR_CLUSTER_ID",
			getenvDefault("AMBASSADOR_SCOUT_ID", "00000000-0000-0000-0000-000000000000")))
	if err != nil {
		panic(err)
	}

	return map[string]interface{}{
		"application": "aes",
		"install_id":  installID,
		"version":     version,
		"metadata": map[string]interface{}{
			"id":        customerID,
			"contact":   customerContact,
			"component": component,
			"features":  featuresDataSet,
		},
	}
}

func getenvDefault(key, fallback string) string {
	ret := os.Getenv(key)
	if ret == "" {
		ret = fallback
	}
	return ret
}
