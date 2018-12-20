package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	pb_struct "github.com/lyft/ratelimit/proto/envoy/api/v2/ratelimit"
	pb "github.com/lyft/ratelimit/proto/envoy/service/ratelimit/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type descriptorValue struct {
	descriptors []*pb_struct.RateLimitDescriptor
}

func (this *descriptorValue) Set(arg string) error {
	parts := strings.Split(arg, ":")
	this.descriptors = make([]*pb_struct.RateLimitDescriptor, len(parts))
	for idx, part := range parts {
		desc, err := this.SetOne(part)
		if err != nil {
			return err
		}
		this.descriptors[idx] = desc
	}
	return nil
}

func (this *descriptorValue) SetOne(arg string) (*pb_struct.RateLimitDescriptor, error) {
	result := &pb_struct.RateLimitDescriptor{}
	pairs := strings.Split(arg, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) != 2 {
			return nil, errors.New("invalid descriptor list")
		}

		result.Entries = append(
			result.Entries, &pb_struct.RateLimitDescriptor_Entry{Key: parts[0], Value: parts[1]})
	}

	return result, nil
}

func (this *descriptorValue) String() string {
	result := make([]string, len(this.descriptors))
	for i, d := range this.descriptors {
		result[i] = d.String()
	}
	return strings.Join(result, ":")
}

func main() {
	dialString := flag.String(
		"dial_string", "localhost:8081", "url of ratelimit server in <host>:<port> form")
	domain := flag.String("domain", "", "rate limit configuration domain to query")
	descriptorValue := descriptorValue{}
	flag.Var(
		&descriptorValue, "descriptors",
		"descriptor list to query in <key>=<value>,<key>=<value>,... form")
	flag.Parse()

	fmt.Printf("dial string: %s\n", *dialString)
	fmt.Printf("domain: %s\n", *domain)
	fmt.Printf("descriptors: %s\n", &descriptorValue)

	conn, err := grpc.Dial(*dialString, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("error connecting: %s\n", err.Error())
		os.Exit(1)
	}

	defer conn.Close()
	c := pb.NewRateLimitServiceClient(conn)
	response, err := c.ShouldRateLimit(
		context.Background(),
		&pb.RateLimitRequest{
			Domain:      *domain,
			Descriptors: descriptorValue.descriptors,
			HitsAddend:  1,
		})
	if err != nil {
		fmt.Printf("request error: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("response: %s\n", response.String())
}
