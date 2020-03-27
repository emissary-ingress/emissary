for x in \
	'envoy.cluster.cluster_http___statsdtest_http.upstream_rq_200:310|c' \
	'envoy.cluster.cluster_http___statsdtest_http.upstream_rq_2xx:310|c' \
	'envoy.cluster.cluster_http___statsdtest_http.upstream_rq_time:3|ms' \
	'envoy.cluster.upstream_rq:363|c|#envoy.response_code:200,envoy.cluster_name:cluster_http___dogstatsdtest_http' \
	'envoy.cluster.upstream_rq_xx:363|c|#envoy.response_code_class:2,envoy.cluster_name:cluster_http___dogstatsdtest_http' \
	'envoy.cluster.upstream_rq_time:2|ms|#envoy.cluster_name:cluster_http___dogstatsdtest_http' \
	'envoy.cluster.cluster_http___dogstatsdtest_http.upstream_rq_200:310|c' \
	'envoy.cluster.cluster_http___dogstatsdtest_http.upstream_rq_2xx:310|c' \
	'envoy.cluster.cluster_http___dogstatsdtest_http.upstream_rq_time:3|ms' \
	'envoy.cluster.upstream_rq:363|c|#envoy.response_code:200,envoy.cluster_name:cluster_http___statsdtest_http' \
	'envoy.cluster.upstream_rq_xx:363|c|#envoy.response_code_class:2,envoy.cluster_name:cluster_http___statsdtest_http' \
	'envoy.cluster.upstream_rq_time:2|ms|#envoy.cluster_name:cluster_http___statsdtest_http' \
	; do
	echo "$x" | nc -u -w 1 127.0.0.1 8125
done

