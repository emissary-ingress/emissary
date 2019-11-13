package ratelimit_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	stats "github.com/lyft/gostats"

	pb "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"

	"github.com/lyft/ratelimit/src/config"
	"github.com/lyft/ratelimit/src/redis"
	ratelimit "github.com/lyft/ratelimit/src/service"
	"github.com/lyft/ratelimit/test/common"

	mock_limiter "github.com/datawire/apro/cmd/amb-sidecar/limiter/mocks"
)

func convertRatelimits(ratelimits []*config.RateLimit) ([]*pb.RateLimitResponse_RateLimit, error) {
	if ratelimits == nil {
		return nil, nil
	}

	ret := make([]*pb.RateLimitResponse_RateLimit, 0)
	for _, rl := range ratelimits {
		if rl == nil {
			ret = append(ret, nil)
			continue
		}
		ret = append(ret, rl.Limit)
	}

	return ret, nil
}

func TestServiceLegacy(test *testing.T) {
	t := commonSetup(test)
	defer t.controller.Finish()
	service := t.setupBasicService()

	// First request, config should be loaded.
	req := common.NewRateLimitRequestLegacy("test-domain", [][][2]string{{{"hello", "world"}}}, 1)
	t.config.EXPECT().GetLimit(context.Background(), "test-domain", req.Descriptors[0]).Return(nil)
	t.cache.EXPECT().DoLimit(context.Background(), req, []*config.RateLimit{nil}).Return(
		[]*pb.RateLimitResponse_DescriptorStatus{{Code: pb.RateLimitResponse_OK, CurrentLimit: nil, LimitRemaining: 0}})

	response, err := service.GetLegacyService().ShouldRateLimit(context.Background(), req)
	t.assert.Equal(
		&pb.RateLimitResponse{
			OverallCode: pb.RateLimitResponse_OK,
			Statuses:    []*pb.RateLimitResponse_DescriptorStatus{{Code: pb.RateLimitResponse_OK, CurrentLimit: nil, LimitRemaining: 0}}},
		response)
	t.assert.Nil(err)

	// Force a config reload.
	barrier := newBarrier()
	t.configLoader.EXPECT().Load(
		// configs
		[]config.RateLimitConfigToLoad{
			{Name: "config.basic_config", FileBytes: "fake_yaml"},
		},
		// stats scope
		gomock.Any(),
	).Do(func([]config.RateLimitConfigToLoad, stats.Scope) {
		barrier.signal()
	}).Return(t.config)
	t.runtimeUpdateCallback <- 1
	barrier.wait()

	// Different request.
	req = common.NewRateLimitRequestLegacy(
		"different-domain", [][][2]string{{{"foo", "bar"}}, {{"hello", "world"}}}, 1)

	limits := []*config.RateLimit{
		config.NewRateLimit(10, pb.RateLimitResponse_RateLimit_MINUTE, "key", t.statStore),
		nil}
	legacyLimits, err := convertRatelimits(limits)
	if err != nil {
		t.assert.FailNow(err.Error())
	}

	t.config.EXPECT().GetLimit(context.Background(), "different-domain", req.Descriptors[0]).Return(limits[0])
	t.config.EXPECT().GetLimit(context.Background(), "different-domain", req.Descriptors[1]).Return(limits[1])
	t.cache.EXPECT().DoLimit(context.Background(), req, limits).Return(
		[]*pb.RateLimitResponse_DescriptorStatus{{Code: pb.RateLimitResponse_OVER_LIMIT, CurrentLimit: limits[0].Limit, LimitRemaining: 0},
			{Code: pb.RateLimitResponse_OK, CurrentLimit: nil, LimitRemaining: 0}})
	response, err = service.GetLegacyService().ShouldRateLimit(context.Background(), req)
	t.assert.Equal(
		&pb.RateLimitResponse{
			OverallCode: pb.RateLimitResponse_OVER_LIMIT,
			Statuses: []*pb.RateLimitResponse_DescriptorStatus{
				{Code: pb.RateLimitResponse_OVER_LIMIT, CurrentLimit: legacyLimits[0], LimitRemaining: 0},
				{Code: pb.RateLimitResponse_OK, CurrentLimit: nil, LimitRemaining: 0},
			}},
		response)
	t.assert.Nil(err)

	// Config load failure.
	t.configLoader.EXPECT().Load(
		// configs
		[]config.RateLimitConfigToLoad{
			{Name: "config.basic_config", FileBytes: "fake_yaml"},
		},
		// stats scope
		gomock.Any(),
	).Do(func([]config.RateLimitConfigToLoad, stats.Scope) {
		barrier.signal()
		panic(config.RateLimitConfigError("load error"))
	})
	t.runtimeUpdateCallback <- 1
	barrier.wait()

	// Config should still be valid. Also make sure order does not affect results.
	limits = []*config.RateLimit{
		nil,
		config.NewRateLimit(10, pb.RateLimitResponse_RateLimit_MINUTE, "key", t.statStore)}
	legacyLimits, err = convertRatelimits(limits)
	if err != nil {
		t.assert.FailNow(err.Error())
	}

	t.config.EXPECT().GetLimit(context.Background(), "different-domain", req.Descriptors[0]).Return(limits[0])
	t.config.EXPECT().GetLimit(context.Background(), "different-domain", req.Descriptors[1]).Return(limits[1])
	t.cache.EXPECT().DoLimit(context.Background(), req, limits).Return(
		[]*pb.RateLimitResponse_DescriptorStatus{{Code: pb.RateLimitResponse_OK, CurrentLimit: nil, LimitRemaining: 0},
			{Code: pb.RateLimitResponse_OVER_LIMIT, CurrentLimit: limits[1].Limit, LimitRemaining: 0}})
	response, err = service.GetLegacyService().ShouldRateLimit(context.Background(), req)
	t.assert.Equal(
		&pb.RateLimitResponse{
			OverallCode: pb.RateLimitResponse_OVER_LIMIT,
			Statuses: []*pb.RateLimitResponse_DescriptorStatus{
				{Code: pb.RateLimitResponse_OK, CurrentLimit: nil, LimitRemaining: 0},
				{Code: pb.RateLimitResponse_OVER_LIMIT, CurrentLimit: legacyLimits[1], LimitRemaining: 0},
			}},
		response)
	t.assert.Nil(err)

	t.assert.EqualValues(2, t.statStore.NewCounter("config_load_success").Value())
	t.assert.EqualValues(1, t.statStore.NewCounter("config_load_error").Value())
}

func TestEmptyDomainLegacy(test *testing.T) {
	t := commonSetup(test)
	defer t.controller.Finish()
	service := t.setupBasicService()

	request := common.NewRateLimitRequestLegacy("", [][][2]string{{{"hello", "world"}}}, 1)
	response, err := service.GetLegacyService().ShouldRateLimit(context.Background(), request)
	t.assert.Nil(response)
	t.assert.Equal("rate limit domain must not be empty", err.Error())
	t.assert.EqualValues(1, t.statStore.NewCounter("call.should_rate_limit.service_error").Value())
	t.assert.EqualValues(1, t.statStore.NewCounter("call.should_rate_limit_legacy.should_rate_limit_error").Value())
}

func TestEmptyDescriptorsLegacy(test *testing.T) {
	t := commonSetup(test)
	defer t.controller.Finish()
	service := t.setupBasicService()

	request := common.NewRateLimitRequestLegacy("test-domain", [][][2]string{}, 1)
	response, err := service.GetLegacyService().ShouldRateLimit(context.Background(), request)
	t.assert.Nil(response)
	t.assert.Equal("rate limit descriptor list must not be empty", err.Error())
	t.assert.EqualValues(1, t.statStore.NewCounter("call.should_rate_limit.service_error").Value())
	t.assert.EqualValues(1, t.statStore.NewCounter("call.should_rate_limit_legacy.should_rate_limit_error").Value())
}

func TestCacheErrorLegacy(test *testing.T) {
	t := commonSetup(test)
	defer t.controller.Finish()
	service := t.setupBasicService()

	req := common.NewRateLimitRequestLegacy("different-domain", [][][2]string{{{"foo", "bar"}}}, 1)
	limits := []*config.RateLimit{config.NewRateLimit(10, pb.RateLimitResponse_RateLimit_MINUTE, "key", t.statStore)}
	t.config.EXPECT().GetLimit(context.Background(), "different-domain", req.Descriptors[0]).Return(limits[0])
	t.cache.EXPECT().DoLimit(context.Background(), req, limits).Do(
		func(context.Context, *pb.RateLimitRequest, []*config.RateLimit) {
			panic(redis.RedisError("cache error"))
		})

	response, err := service.GetLegacyService().ShouldRateLimit(context.Background(), req)
	t.assert.Nil(response)
	t.assert.Equal("cache error", err.Error())
	t.assert.EqualValues(1, t.statStore.NewCounter("call.should_rate_limit.redis_error").Value())
	t.assert.EqualValues(1, t.statStore.NewCounter("call.should_rate_limit_legacy.should_rate_limit_error").Value())
}

func TestInitialLoadErrorLegacy(test *testing.T) {
	t := commonSetup(test)
	defer t.controller.Finish()

	t.runtime.EXPECT().AddUpdateCallback(gomock.Any()).Do(
		func(callback chan<- int) { t.runtimeUpdateCallback = callback })
	t.runtime.EXPECT().Snapshot().Return(t.snapshot).MinTimes(1)
	t.snapshot.EXPECT().Keys().Return([]string{"foo", "config.basic_config"}).MinTimes(1)
	t.snapshot.EXPECT().Get("config.basic_config").Return("fake_yaml").MinTimes(1)
	t.configLoader.EXPECT().Load(
		// configs
		[]config.RateLimitConfigToLoad{
			{Name: "config.basic_config", FileBytes: "fake_yaml"},
		},
		// stats scope
		gomock.Any(),
	).Do(func([]config.RateLimitConfigToLoad, stats.Scope) {
		panic(config.RateLimitConfigError("load error"))
	})
	service := ratelimit.NewService(t.runtime, t.cache, t.configLoader, t.statStore, mock_limiter.NewMockLimiter())

	request := common.NewRateLimitRequestLegacy("test-domain", [][][2]string{{{"hello", "world"}}}, 1)
	response, err := service.GetLegacyService().ShouldRateLimit(context.Background(), request)
	t.assert.Nil(response)
	t.assert.Equal("no rate limit configuration loaded", err.Error())
	t.assert.EqualValues(1, t.statStore.NewCounter("call.should_rate_limit.service_error").Value())
	t.assert.EqualValues(1, t.statStore.NewCounter("call.should_rate_limit_legacy.should_rate_limit_error").Value())

}
