#!bash

GOLDDIR=${1:-python/tests/gold}

# rm -rf "$GOLDDIR"
mkdir -p "$GOLDDIR"

kubectl() {
    if ! test -f tools/bin/kubectl; then
        make tools/bin/kubectl >&2
    fi
    tools/bin/kubectl "$@"
}

copy_gold () {
	local pod="$1"
	local namespace="${2:-default}"

	if kubectl cp -n $namespace $pod:/tmp/ambassador "$GOLDDIR/${pod}-tmp" >/dev/null; then
		rm -rf "$GOLDDIR/$pod"
		mv "$GOLDDIR/${pod}-tmp" "$GOLDDIR/${pod}"
		# We don't try to compare aconf nor ir configuration when using gold files.
		# They only contribute noise, so remove them here.
		rm -f "$GOLDDIR/${pod}/snapshots/aconf.json"
		rm -f "$GOLDDIR/${pod}/snapshots/ir.json"
		printf "                                                                \r"
		printf "${pod}...\r"
		# echo "$pod: copied"
	else
		printf "                                                                \r"
		printf "${pod}.${namespace}: failed\n"
	fi
}

copy_gold acceptancegrpcbridgetest
copy_gold acceptancegrpctest
copy_gold acceptancegrpcwebtest
copy_gold ambassadoridtest
copy_gold authenticationgrpctest
copy_gold authenticationv2grpctest
copy_gold authenticationheaderrouting
copy_gold authenticationhttpbufferedtest
copy_gold authenticationhttpfailuremodeallowtest
copy_gold authenticationhttppartialbuffertest
copy_gold authenticationtest
copy_gold authenticationtestv1
copy_gold authenticationwebsockettest
copy_gold circuitbreakingtcptest
# copy_gold circuitbreakingtest	# mockery can't quite cope with this -- not sure why
copy_gold clientcertificateauthentication
copy_gold clustertagtest
# copy_gold consultest
copy_gold dogstatsdtest
# copy_gold endpointgrpctest
copy_gold envoylogtest
copy_gold envoylogjsontest
copy_gold globalcircuitbreakingtest
copy_gold globalcorstest
# copy_gold globalloadbalancing
copy_gold gzipminimumconfigtest
copy_gold gzipnotsupportedcontenttypetest
copy_gold gziptest
copy_gold hostcrdwildcards
copy_gold hostcrdcleartext
copy_gold hostcrdrootredirectslashmapping
copy_gold hostcrdtlsconfig
copy_gold hostcrddouble
copy_gold hostcrdclientcertsamenamespace
copy_gold hostcrdrootredirectcongratulations
copy_gold hostcrdrootredirectre2mapping
copy_gold hostcrdseparatetlscontext
copy_gold hostcrdclientcertcrossnamespace
copy_gold hostcrdmanualcontext
copy_gold hostcrdno8080
copy_gold hostcrdsingle
copy_gold hostcrdforcedstar
# copy_gold ingressstatustest1
# copy_gold ingressstatustest2
# copy_gold ingressstatustestacrossnamespaces
# copy_gold ingressstatustestwithannotations
# copy_gold knative0110test
copy_gold linkerdheadermapping
copy_gold listeneridletimeout
# copy_gold loadbalancertest
copy_gold logservicetest
# copy_gold luatest
# copy_gold permappingloadbalancing
copy_gold ratelimitv0test
copy_gold ratelimitv1test
copy_gold ratelimitv1withtlstest
copy_gold ratelimitv2test
copy_gold redirecttests
copy_gold redirecttestsinvalidsecret
copy_gold redirecttestswithproxyproto
copy_gold retrypolicytest
copy_gold saferegexmapping
copy_gold servernametest
copy_gold statsdtest
copy_gold tls
copy_gold tlscontextciphersuites
copy_gold tlscontextprotocolmaxversion
copy_gold tlscontextprotocolminversion
copy_gold tlscontextstest
copy_gold tlscontexttest
# copy_gold tlsingresstest
copy_gold tlsinvalidsecret
copy_gold tlsoriginationsecret
copy_gold tracingexternalauthtest
copy_gold tracingtest
copy_gold tracingtestsampling
copy_gold tracingtestshorttraceid
copy_gold tracingtestzipkinv1
copy_gold tracingtestzipkinv2
copy_gold unsaferegexmapping
copy_gold xfpredirect
copy_gold empty empty-namespace
copy_gold plain plain-namespace
copy_gold tcpmappingtest tcp-namespace

printf "\n"
