package metriton

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jpillora/backoff"

	"github.com/datawire/ambassador/pkg/metriton"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/lib/licensekeys"
)

const phoneHomeEveryPeriod = 12 * time.Hour

var reporter *metriton.Reporter

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
	if metriton.IsDisabledByUser() {
		fmt.Println("SCOUT_DISABLE, enforcing hard-limits")
		if limiter != nil {
			limiter.SetPhoneHomeHardLimits(true)
		}
		return nil
	}

	if reporter == nil {
		reporter = &metriton.Reporter{
			Application: "aes",
			Version:     version,
			GetInstallID: func(*metriton.Reporter) (string, error) {
				return getenvDefault("AMBASSADOR_CLUSTER_ID",
					getenvDefault("AMBASSADOR_SCOUT_ID",
						"00000000-0000-0000-0000-000000000000")), nil
			},
			BaseMetadata: nil,

			// This can't be an environment variable, or else the user will be
			// able to spoof Metriton responses, bypassing the `hard_limit`
			// response field.
			//
			// Use "https://kubernaut.io/beta/scout" for testing.
			Endpoint: "https://kubernaut.io/scout",
		}
	}

	resp, err := reporter.Report(context.TODO(), prepareData(claims, limiter, component))
	if err != nil {
		if limiter != nil {
			fmt.Println("Metriton call was not a success... allow soft-limit")
			limiter.SetPhoneHomeHardLimits(false)
		}
		return err
	}
	if limiter != nil && resp != nil && resp.HardLimit {
		fmt.Println("Metriton is enforcing hard-limits")
		limiter.SetPhoneHomeHardLimits(true)
		return nil
	}
	return nil
}

func prepareData(claims *licensekeys.LicenseClaimsLatest, limiter *limiter.LimiterImpl, component string) map[string]interface{} {
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

	return map[string]interface{}{
		"id":        customerID,
		"contact":   customerContact,
		"component": component,
		"features":  featuresDataSet,
	}
}

func getenvDefault(key, fallback string) string {
	ret := os.Getenv(key)
	if ret == "" {
		ret = fallback
	}
	return ret
}
