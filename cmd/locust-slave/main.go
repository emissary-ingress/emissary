package main

import (
	"context"
	"fmt"
	"time"

	"github.com/myzhan/boomer"
	"google.golang.org/grpc"

	envoyAuthV2_new "github.com/datawire/ambassador/go/apis/envoy/service/auth/v2"
	envoyAuthV2 "github.com/datawire/ambassador/go/apis/envoy/service/auth/v2alpha"

	pb_struct "github.com/datawire/ambassador/go/apis/envoy/api/v2/ratelimit"
	pb "github.com/datawire/ambassador/go/apis/envoy/service/ratelimit/v1"
	pb_new "github.com/datawire/ambassador/go/apis/envoy/service/ratelimit/v2"
)

var rlsConn *grpc.ClientConn

func doRls() error {
	var err error

	if rlsConn == nil {
		rlsConn, err = grpc.DialContext(context.Background(), "127.0.0.1:8081", grpc.WithInsecure())
		if err != nil {
			return fmt.Errorf("grpc dial failed: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	client := pb.NewRateLimitServiceClient(rlsConn)
	req := new(pb_new.RateLimitRequest)
	req.Domain = "envoy"
	entry := pb_struct.RateLimitDescriptor_Entry{Key: "client_id", Value: "foo"}
	entries := []*pb_struct.RateLimitDescriptor_Entry{&entry}
	req.Descriptors = []*pb_struct.RateLimitDescriptor{{Entries: entries}}

	_, err = client.ShouldRateLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("grpc request failed: %v", err)
	}

	// FIXME: Assert something about the response?

	return nil
}

var authConn *grpc.ClientConn

func doAuth() error {
	var err error

	if authConn == nil {
		authConn, err = grpc.DialContext(context.Background(), "127.0.0.1:3000", grpc.WithInsecure())
		if err != nil {
			return fmt.Errorf("grpc dial failed: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	client := envoyAuthV2.NewAuthorizationClient(authConn)
	req := new(envoyAuthV2_new.CheckRequest)

	_, err = client.Check(ctx, req)
	if err != nil {
		return fmt.Errorf("grpc request failed: %v", err)
	}

	// FIXME: Assert something about the response?

	return nil
}

func wrap(name string, fn func() error) func() {
	return func() {
		start := time.Now()
		err := fn()
		responseTime := time.Since(start).Nanoseconds() / int64(time.Millisecond)

		if err != nil {
			boomer.RecordFailure("grpc", name, responseTime, err.Error())
			//fmt.Println(err)
		} else {
			boomer.RecordSuccess("grpc", name, responseTime, 10) // FIXME 10?
			//fmt.Println("Success")
		}
	}
}

func main() {
	task1 := &boomer.Task{
		Name:   "rls",
		Weight: 10,
		Fn:     wrap("rls", doRls),
	}
	task2 := &boomer.Task{
		Name:   "auth",
		Weight: 10,
		Fn:     wrap("auth", doAuth),
	}

	boomer.Run(task1, task2)
}
