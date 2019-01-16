package redis

import (
	pb "github.com/lyft/ratelimit/proto/envoy/service/ratelimit/v2"
	"github.com/lyft/ratelimit/src/config"
	"golang.org/x/net/context"
)

// Interface for a time source.
type TimeSource interface {
	// @return the current unix time in seconds.
	UnixNow() int64
}

// Interface for a rand Source for expiration jitter.
type JitterRandSource interface {
	// @return a non-negative pseudo-random 63-bit integer as an int64.
	Int63() int64
	// @param seed initializes pseudo-random generator to a deterministic state.
	Seed(seed int64)
}

// Interface for interacting with a cache backend for rate limiting.
type RateLimitCache interface {
	// Contact the cache and perform rate limiting for a set of descriptors and limits.
	// @param ctx supplies the request context.
	// @param request supplies the ShouldRateLimit service request.
	// @param limits supplies the list of associated limits. It's possible for a limit to be nil
	//               which means that the associated descriptor does not need to be checked. This
	//               is done for simplicity reasons in the overall service API. The length of this
	//               list must be same as the length of the descriptors list.
	// @return a list of DescriptorStatuses which corresponds to each passed in descriptor/limit pair.
	// 				 Throws RedisError if there was any error talking to the cache.
	DoLimit(
		ctx context.Context,
		request *pb.RateLimitRequest,
		limits []*config.RateLimit) []*pb.RateLimitResponse_DescriptorStatus
}
