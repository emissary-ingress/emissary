package common

import (
	pb_struct "github.com/datawire/ambassador/pkg/api/envoy/api/v2/ratelimit"
	pb "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"
)

func NewRateLimitRequest(domain string, descriptors [][][2]string, hitsAddend uint32) *pb.RateLimitRequest {
	request := &pb.RateLimitRequest{}
	request.Domain = domain
	for _, descriptor := range descriptors {
		newDescriptor := &pb_struct.RateLimitDescriptor{}
		for _, entry := range descriptor {
			newDescriptor.Entries = append(
				newDescriptor.Entries,
				&pb_struct.RateLimitDescriptor_Entry{Key: entry[0], Value: entry[1]})
		}
		request.Descriptors = append(request.Descriptors, newDescriptor)
	}
	request.HitsAddend = hitsAddend
	return request
}

func NewRateLimitRequestLegacy(domain string, descriptors [][][2]string, hitsAddend uint32) *pb.RateLimitRequest {
	request := &pb.RateLimitRequest{}
	request.Domain = domain
	for _, descriptor := range descriptors {
		newDescriptor := &pb_struct.RateLimitDescriptor{}
		for _, entry := range descriptor {
			newDescriptor.Entries = append(
				newDescriptor.Entries,
				&pb_struct.RateLimitDescriptor_Entry{Key: entry[0], Value: entry[1]})
		}
		request.Descriptors = append(request.Descriptors, newDescriptor)
	}
	request.HitsAddend = hitsAddend
	return request
}
