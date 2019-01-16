// +build integration

package integration_test

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	pb "github.com/lyft/ratelimit/proto/envoy/service/ratelimit/v2"
	pb_legacy "github.com/lyft/ratelimit/proto/ratelimit"
	"github.com/lyft/ratelimit/src/service_cmd/runner"
	"github.com/lyft/ratelimit/test/common"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func newDescriptorStatus(
	status pb.RateLimitResponse_Code, requestsPerUnit uint32,
	unit pb.RateLimitResponse_RateLimit_Unit, limitRemaining uint32) *pb.RateLimitResponse_DescriptorStatus {

	return &pb.RateLimitResponse_DescriptorStatus{
		Code:           status,
		CurrentLimit:   &pb.RateLimitResponse_RateLimit{RequestsPerUnit: requestsPerUnit, Unit: unit},
		LimitRemaining: limitRemaining,
	}
}

func newDescriptorStatusLegacy(
	status pb_legacy.RateLimitResponse_Code, requestsPerUnit uint32,
	unit pb_legacy.RateLimit_Unit, limitRemaining uint32) *pb_legacy.RateLimitResponse_DescriptorStatus {

	return &pb_legacy.RateLimitResponse_DescriptorStatus{
		Code:           status,
		CurrentLimit:   &pb_legacy.RateLimit{RequestsPerUnit: requestsPerUnit, Unit: unit},
		LimitRemaining: limitRemaining,
	}
}

func TestBasicConfig(t *testing.T) {
	t.Run("WithoutPerSecondRedis", testBasicConfig("8083", "false"))
	t.Run("WithPerSecondRedis", testBasicConfig("8085", "true"))
}

func testBasicConfig(grpcPort, perSecond string) func(*testing.T) {
	return func(t *testing.T) {
		os.Setenv("REDIS_PERSECOND", perSecond)
		os.Setenv("PORT", "8082")
		os.Setenv("GRPC_PORT", grpcPort)
		os.Setenv("DEBUG_PORT", "8084")
		os.Setenv("RUNTIME_ROOT", "runtime/current")
		os.Setenv("RUNTIME_SUBDIRECTORY", "ratelimit")
		os.Setenv("REDIS_PERSECOND_SOCKET_TYPE", "tcp")
		os.Setenv("REDIS_PERSECOND_URL", "localhost:6380")
		os.Setenv("REDIS_SOCKET_TYPE", "tcp")
		os.Setenv("REDIS_URL", "localhost:6379")

		go func() {
			runner.Run()
		}()

		// HACK: Wait for the server to come up. Make a hook that we can wait on.
		time.Sleep(100 * time.Millisecond)

		assert := assert.New(t)
		conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", grpcPort), grpc.WithInsecure())
		assert.NoError(err)
		defer conn.Close()
		c := pb.NewRateLimitServiceClient(conn)

		response, err := c.ShouldRateLimit(
			context.Background(),
			common.NewRateLimitRequest("foo", [][][2]string{{{"hello", "world"}}}, 1))
		assert.Equal(
			&pb.RateLimitResponse{
				OverallCode: pb.RateLimitResponse_OK,
				Statuses:    []*pb.RateLimitResponse_DescriptorStatus{{Code: pb.RateLimitResponse_OK, CurrentLimit: nil, LimitRemaining: 0}}},
			response)
		assert.NoError(err)

		response, err = c.ShouldRateLimit(
			context.Background(),
			common.NewRateLimitRequest("basic", [][][2]string{{{"key1", "foo"}}}, 1))
		assert.Equal(
			&pb.RateLimitResponse{
				OverallCode: pb.RateLimitResponse_OK,
				Statuses: []*pb.RateLimitResponse_DescriptorStatus{
					newDescriptorStatus(pb.RateLimitResponse_OK, 50, pb.RateLimitResponse_RateLimit_SECOND, 49)}},
			response)
		assert.NoError(err)

		// Now come up with a random key, and go over limit for a minute limit which should always work.
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomInt := r.Int()
		for i := 0; i < 25; i++ {
			response, err = c.ShouldRateLimit(
				context.Background(),
				common.NewRateLimitRequest(
					"another", [][][2]string{{{"key2", strconv.Itoa(randomInt)}}}, 1))

			status := pb.RateLimitResponse_OK
			limitRemaining := uint32(20 - (i + 1))
			if i >= 20 {
				status = pb.RateLimitResponse_OVER_LIMIT
				limitRemaining = 0
			}

			assert.Equal(
				&pb.RateLimitResponse{
					OverallCode: status,
					Statuses: []*pb.RateLimitResponse_DescriptorStatus{
						newDescriptorStatus(status, 20, pb.RateLimitResponse_RateLimit_MINUTE, limitRemaining)}},
				response)
			assert.NoError(err)
		}

		// Limit now against 2 keys in the same domain.
		randomInt = r.Int()
		for i := 0; i < 15; i++ {
			response, err = c.ShouldRateLimit(
				context.Background(),
				common.NewRateLimitRequest(
					"another",
					[][][2]string{
						{{"key2", strconv.Itoa(randomInt)}},
						{{"key3", strconv.Itoa(randomInt)}}}, 1))

			status := pb.RateLimitResponse_OK
			limitRemaining1 := uint32(20 - (i + 1))
			limitRemaining2 := uint32(10 - (i + 1))
			if i >= 10 {
				status = pb.RateLimitResponse_OVER_LIMIT
				limitRemaining2 = 0
			}

			assert.Equal(
				&pb.RateLimitResponse{
					OverallCode: status,
					Statuses: []*pb.RateLimitResponse_DescriptorStatus{
						newDescriptorStatus(pb.RateLimitResponse_OK, 20, pb.RateLimitResponse_RateLimit_MINUTE, limitRemaining1),
						newDescriptorStatus(status, 10, pb.RateLimitResponse_RateLimit_HOUR, limitRemaining2)}},
				response)
			assert.NoError(err)
		}
	}
}

func TestBasicConfigLegacy(t *testing.T) {
	os.Setenv("PORT", "8082")
	os.Setenv("GRPC_PORT", "8083")
	os.Setenv("DEBUG_PORT", "8084")
	os.Setenv("RUNTIME_ROOT", "runtime/current")
	os.Setenv("RUNTIME_SUBDIRECTORY", "ratelimit")

	go func() {
		runner.Run()
	}()

	// HACK: Wait for the server to come up. Make a hook that we can wait on.
	time.Sleep(100 * time.Millisecond)

	assert := assert.New(t)
	conn, err := grpc.Dial("localhost:8083", grpc.WithInsecure())
	assert.NoError(err)
	defer conn.Close()
	c := pb_legacy.NewRateLimitServiceClient(conn)

	response, err := c.ShouldRateLimit(
		context.Background(),
		common.NewRateLimitRequestLegacy("foo", [][][2]string{{{"hello", "world"}}}, 1))
	assert.Equal(
		&pb_legacy.RateLimitResponse{
			OverallCode: pb_legacy.RateLimitResponse_OK,
			Statuses:    []*pb_legacy.RateLimitResponse_DescriptorStatus{{Code: pb_legacy.RateLimitResponse_OK, CurrentLimit: nil, LimitRemaining: 0}}},
		response)
	assert.NoError(err)

	response, err = c.ShouldRateLimit(
		context.Background(),
		common.NewRateLimitRequestLegacy("basic_legacy", [][][2]string{{{"key1", "foo"}}}, 1))
	assert.Equal(
		&pb_legacy.RateLimitResponse{
			OverallCode: pb_legacy.RateLimitResponse_OK,
			Statuses: []*pb_legacy.RateLimitResponse_DescriptorStatus{
				newDescriptorStatusLegacy(pb_legacy.RateLimitResponse_OK, 50, pb_legacy.RateLimit_SECOND, 49)}},
		response)
	assert.NoError(err)

	// Now come up with a random key, and go over limit for a minute limit which should always work.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomInt := r.Int()
	for i := 0; i < 25; i++ {
		response, err = c.ShouldRateLimit(
			context.Background(),
			common.NewRateLimitRequestLegacy(
				"another", [][][2]string{{{"key2", strconv.Itoa(randomInt)}}}, 1))

		status := pb_legacy.RateLimitResponse_OK
		limitRemaining := uint32(20 - (i + 1))
		if i >= 20 {
			status = pb_legacy.RateLimitResponse_OVER_LIMIT
			limitRemaining = 0
		}

		assert.Equal(
			&pb_legacy.RateLimitResponse{
				OverallCode: status,
				Statuses: []*pb_legacy.RateLimitResponse_DescriptorStatus{
					newDescriptorStatusLegacy(status, 20, pb_legacy.RateLimit_MINUTE, limitRemaining)}},
			response)
		assert.NoError(err)
	}

	// Limit now against 2 keys in the same domain.
	randomInt = r.Int()
	for i := 0; i < 15; i++ {
		response, err = c.ShouldRateLimit(
			context.Background(),
			common.NewRateLimitRequestLegacy(
				"another_legacy",
				[][][2]string{
					{{"key2", strconv.Itoa(randomInt)}},
					{{"key3", strconv.Itoa(randomInt)}}}, 1))

		status := pb_legacy.RateLimitResponse_OK
		limitRemaining1 := uint32(20 - (i + 1))
		limitRemaining2 := uint32(10 - (i + 1))
		if i >= 10 {
			status = pb_legacy.RateLimitResponse_OVER_LIMIT
			limitRemaining2 = 0
		}

		assert.Equal(
			&pb_legacy.RateLimitResponse{
				OverallCode: status,
				Statuses: []*pb_legacy.RateLimitResponse_DescriptorStatus{
					newDescriptorStatusLegacy(pb_legacy.RateLimitResponse_OK, 20, pb_legacy.RateLimit_MINUTE, limitRemaining1),
					newDescriptorStatusLegacy(status, 10, pb_legacy.RateLimit_HOUR, limitRemaining2)}},
			response)
		assert.NoError(err)
	}
}
