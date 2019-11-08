package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
)

func init() {
	var (
		argLicenseKey string
		limitName string
		redisSocketType string
		redisUrl string
	)
	get := &cobra.Command{
		Use:   "get",
		Short: "Get a particular limit value",
	}
	get.Flags().StringVarP(&argLicenseKey, "license-key", "l", "", "the license key")
	get.Flags().StringVarP(&limitName, "limit-name", "n", "", "the limit name to enforce")
	get.Flags().StringVarP(&redisUrl, "redis-url", "u", "", "the redis url to use")
	get.Flags().StringVarP(&redisSocketType, "redis-socket-type", "", "tcp", "the redis socket type to use")
	get.MarkFlagRequired("license-key")
	get.MarkFlagRequired("limit-name")
	get.MarkFlagRequired("redis-url")

	get.RunE = func(cmd *cobra.Command, args []string) error {
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

		maximumCount := limiter.GetLimitValueAtPointInTime(&limit)
		currentUsage := 0

		if limit.Type() == licensekeys.LimitTypeCount {
			climiter, err := limiter.CreateCountLimiter(&limit)
			if err != nil {
				return err
			}
			currentUsage, err = climiter.GetUnderlyingValueAtPointInTime()
			if err != nil {
				return err
			}
		} else {
			return errors.New("Unsure how to set for limit type!")
		}

		fmt.Printf("Current Usage: [%d]\nMaximum Usage: [%d]\n", currentUsage, maximumCount)

		return nil
	}

	argparser.AddCommand(get)
}
