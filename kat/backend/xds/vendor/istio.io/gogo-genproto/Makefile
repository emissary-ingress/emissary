GOOGLEAPIS_SHA = c8c975543a134177cc41b64cbbf10b88fe66aa1d
GOOGLEAPIS_URL = https://raw.githubusercontent.com/googleapis/googleapis/$(GOOGLEAPIS_SHA)

CENSUS_SHA = 7f2434bc10da710debe5c4315ed6d4df454b4024
CENSUS_URL = https://raw.githubusercontent.com/census-instrumentation/opencensus-proto/$(CENSUS_SHA)/src

PROMETHEUS_SHA = 6f3806018612930941127f2a7c6c453ba2c527d2
PROMETHEUS_URL = https://raw.githubusercontent.com/prometheus/client_model/$(PROMETHEUS_SHA)

GOGO_PROTO_PKG := github.com/gogo/protobuf/gogoproto
GOGO_TYPES := github.com/gogo/protobuf/types
GOGO_DESCRIPTOR := github.com/gogo/protobuf/protoc-gen-gogo/descriptor

importmaps := \
	gogoproto/gogo.proto=$(GOGO_PROTO_PKG) \
	google/protobuf/any.proto=$(GOGO_TYPES) \
	google/protobuf/descriptor.proto=$(GOGO_DESCRIPTOR) \
	google/protobuf/duration.proto=$(GOGO_TYPES) \
	google/protobuf/timestamp.proto=$(GOGO_TYPES) \
	google/protobuf/wrappers.proto=$(GOGO_TYPES) \

comma := ,
empty :=
space := $(empty) $(empty)
mapping_with_spaces := $(foreach map,$(importmaps),M$(map),)
MAPPING := $(subst $(space),$(empty),$(mapping_with_spaces))
GOGOSLICK_PLUGIN := --plugin=protoc-gen-gogoslick=gogoslick --gogoslick_out=$(MAPPING)
GOGOFASTER_PLUGIN := --plugin=protoc-gen-gogofaster=gogofaster --gogofaster_out=$(MAPPING)
PROTOC = protoc

googleapis_protos = \
	google/api/http.proto \
	google/api/annotations.proto \
	google/rpc/status.proto \
	google/rpc/code.proto \
	google/rpc/error_details.proto \
	google/type/color.proto \
	google/type/date.proto \
	google/type/dayofweek.proto \
	google/type/latlng.proto \
	google/type/money.proto \
	google/type/postal_address.proto \
	google/type/timeofday.proto \

googleapis_packages = \
	google/api \
	google/rpc \
	google/type \

census_protos = \
	opencensus/proto/stats/v1/stats.proto \
	opencensus/proto/trace/v1/trace.proto \

census_packages = \
	opencensus/proto/stats/v1 \
	opencensus/proto/trace/v1 \

all: build

vendor:
	dep ensure --vendor-only

depend: vendor
	$(foreach var,$(googleapis_packages),mkdir -p googleapis/$(var);)
	$(foreach var,$(census_packages),mkdir -p $(var);)

protoc.version:
	# Record protoc version
	@echo `$(PROTOC) --version` > protoc.version

gogoslick: depend
	@go build -o gogoslick vendor/github.com/gogo/protobuf/protoc-gen-gogoslick/main.go

gogofaster: depend
	@go build -o gogofaster vendor/github.com/gogo/protobuf/protoc-gen-gogofaster/main.go

$(googleapis_protos): %:
	# Download $@ at $(GOOGLEAPIS_SHA)
	@curl -sS $(GOOGLEAPIS_URL)/$@ -o googleapis/$@.tmp
	@sed -e '/^option go_package/d' googleapis/$@.tmp > googleapis/$@
	@rm googleapis/$@.tmp

$(googleapis_packages): %: gogoslick protoc.version $(googleapis_protos)
	# Generate $@
	@$(PROTOC) $(GOGOSLICK_PLUGIN):googleapis -I googleapis googleapis/$@/*.proto

$(census_protos): %:
	# Download $@ at $(CENSUS_SHA)
	@curl -sS $(CENSUS_URL)/$@ -o $@
	@sed -i.tmp '/^option go_package/d' $@
	@rm $@.tmp

$(census_packages): %: gogofaster protoc.version $(census_protos)
	# Generate $@
	@$(PROTOC) $(GOGOFASTER_PLUGIN):. -I . $@/*.proto

prometheus/metrics.proto:
	@mkdir -p prometheus
	@curl -sS $(PROMETHEUS_URL)/metrics.proto -o prometheus/metrics.proto

prometheus/metrics.pb.go: gogofaster protoc.version prometheus/metrics.proto
	# Generate prometheus
	@$(PROTOC) $(GOGOFASTER_PLUGIN):. prometheus/metrics.proto

generate: $(googleapis_packages) $(census_packages) prometheus/metrics.pb.go

format: generate
	# Format code
	@gofmt -l -s -w googleapis

build: format
	# Build code
	@go build ./...

clean:
	@rm gogoslick

.PHONY: all depend format build $(googleapis_protos) $(googleapis_packages) protoc.version clean
