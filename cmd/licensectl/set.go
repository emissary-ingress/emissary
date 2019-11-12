package main

import (
	"fmt"
	"strings"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/lib/licensekeys"
)

func init() {
	var (
		argLicenseKey   string
		limitName       string
		redisSocketType string
		redisUrl        string
		value           string
	)
	set := &cobra.Command{
		Use:   "set",
		Short: "Set a particular limit value",
	}
	set.Flags().StringVarP(&argLicenseKey, "license-key", "l", "", "the license key")
	set.Flags().StringVarP(&limitName, "limit-name", "n", "", "the limit name to enforce")
	set.Flags().StringVarP(&redisUrl, "redis-url", "u", "", "the redis url to use")
	set.Flags().StringVarP(&redisSocketType, "redis-socket-type", "", "tcp", "the redis socket type to use")
	set.Flags().StringVarP(&value, "value", "v", "", "the value to pass to the limiter set function")
	set.MarkFlagRequired("license-key")
	set.MarkFlagRequired("limit-name")
	set.MarkFlagRequired("redis-url")
	set.MarkFlagRequired("value")

	set.RunE = func(cmd *cobra.Command, args []string) error {
		license, err := licensekeys.ParseKey(argLicenseKey)
		if err != nil {
			return err
		}

		limit, ok := licensekeys.ParseLimit(limitName)
		if !ok {
			return errors.New("Unknown limit: " + limitName)
		}

		redisPool, redisPoolErr := pool.New(redisSocketType, redisUrl, 1)
		if redisPoolErr != nil {
			return redisPoolErr
		}
		if redisPool == nil {
			return errors.New("nil redis-pool!")
		}

		limiter := limiter.NewLimiterImpl()
		limiter.SetClaims(license)
		limiter.SetRedisPool(redisPool)

		if limit.Type() == licensekeys.LimitTypeCount {
			values := strings.Split(value, ",")
			climiter, err := limiter.CreateCountLimiter(&limit)
			if err != nil {
				return err
			}

			for _, value := range values {
				err = climiter.IncrementUsage(value)
				if err != nil {
					return err
				}
			}
			fmt.Println("Added values!")
		} else {
			return errors.New("Unsure how to set for limit type!")
		}

		return nil
	}

	argparser.AddCommand(set)
}
