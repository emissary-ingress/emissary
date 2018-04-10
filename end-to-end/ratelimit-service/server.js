'use strict';

const GRPC_PORT = process.env.GRPC_PORT || '5000';
const grpc = require('grpc');
const grpcserver = new grpc.Server();

const path = require('path');
const PROTO_PATH = path.normalize(__dirname + '/ratelimit.proto');
const ratelimitProto = grpc.load(PROTO_PATH).pb.lyft.ratelimit;

grpcserver.addService(ratelimitProto.RateLimitService.service, {
	shouldRateLimit: (call, callback) => {
		let allow = false;
		const rateLimitResponse = new ratelimitProto.RateLimitResponse();
	
		console.log("========>");
		console.log(call.request.domain);
		call.request.descriptors.forEach((descriptor) => {
			descriptor.entries.forEach((entry) => {
				console.log(`  ${entry.key} = ${entry.value}`);
				
				if (entry.key === 'x-ambassador-test-allow' && entry.value === 'true') {
					allow = true;
				}
			});

			const descriptorStatus = new ratelimitProto.RateLimitResponse.DescriptorStatus();
			const rateLimit = new ratelimitProto.RateLimit();
			rateLimit.requests_per_unit = 1000;
			rateLimit.unit = ratelimitProto.RateLimit.Unit.SECOND;
			descriptorStatus.code = ratelimitProto.RateLimitResponse.Code.OK;
			descriptorStatus.current_limit = rateLimit;
			descriptorStatus.limit_remaining = Number.MAX_VALUE;
			rateLimitResponse.statuses.push(descriptorStatus);
		});
		if (allow) {
			rateLimitResponse.overall_code = ratelimitProto.RateLimitResponse.Code.OK;
		} else {
			rateLimitResponse.overall_code = ratelimitProto.RateLimitResponse.Code.OVER_LIMIT;
		}

		console.log("<========");
		console.log(rateLimitResponse);
		return callback(null, rateLimitResponse);
	}
});

grpcserver.bind(`0.0.0.0:${GRPC_PORT}`, grpc.ServerCredentials.createInsecure());
grpcserver.start();
console.log(`Listening on GRPC port ${GRPC_PORT}`);
