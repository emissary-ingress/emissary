package ratelimit

import (
	"context"

	stats "github.com/lyft/gostats"

	pb_legacy "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v1"
	pb "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"
)

type RateLimitLegacyServiceServer interface {
	pb_legacy.RateLimitServiceServer
}

// legacyService is used to implement ratelimit.proto (https://github.com/lyft/ratelimit/blob/0ded92a2af8261d43096eba4132e45b99a3b8b14/proto/ratelimit/ratelimit.proto)
// the legacyService receives RateLimitRequests, converts the request, and calls the service's ShouldRateLimit method.
type legacyService struct {
	s                          *service
	shouldRateLimitLegacyStats shouldRateLimitLegacyStats
}

type shouldRateLimitLegacyStats struct {
	reqConversionError   stats.Counter
	respConversionError  stats.Counter
	shouldRateLimitError stats.Counter
}

func newShouldRateLimitLegacyStats(scope stats.Scope) shouldRateLimitLegacyStats {
	s := scope.Scope("call.should_rate_limit_legacy")
	return shouldRateLimitLegacyStats{
		reqConversionError:   s.NewCounter("req_conversion_error"),
		respConversionError:  s.NewCounter("resp_conversion_error"),
		shouldRateLimitError: s.NewCounter("should_rate_limit_error"),
	}
}

func (this *legacyService) ShouldRateLimit(
	ctx context.Context,
	request *pb.RateLimitRequest) (finalResponse *pb.RateLimitResponse, finalError error) {

	resp, err := this.s.ShouldRateLimit(ctx, request)
	if err != nil {
		this.shouldRateLimitLegacyStats.shouldRateLimitError.Inc()
		return nil, err
	}

	return resp, nil
}
