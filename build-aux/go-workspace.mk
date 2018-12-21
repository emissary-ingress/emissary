
GOPATH=$(CURDIR)/.go-workspace
GOBIN=$(CURDIR)
GO = GOPATH=$(GOPATH) GOBIN=$(GOBIN) go

IMAGE_GOBIN=$(CURDIR)/image
IMAGE_GO = CGO_ENABLED=0 GOOS=linux GOPATH=$(GOPATH) GOBIN=$(IMAGE_GOBIN) go

build: $(bins)
.PHONY: build

vendor::
ifneq ($(wildcard glide.yaml),)
vendor:: glide.yaml $(wildcard glide.lock)
	rm -rf $@
	glide install
endif

$(bins): %: FORCE vendor
	$(GO) install $(pkg)/cmd/$@

$(bins:%=image/%): %: FORCE vendor
	$(IMAGE_GO) build -o ${@} $(pkg)/cmd/${@:image/%=%}

# .NOTPARALLEL is important, as having multiple `go install`s going at
# once can corrupt `$(GOPATH)/pkg`.  Setting .NOTPARALLEL is simpler
# than mucking with multi-target pattern rules.
.NOTPARALLEL:
