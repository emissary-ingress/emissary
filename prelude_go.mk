# This is part of `prelude.mk`, split out for organizational purposes.
# !!! NOTHING EAGER IS ALLOWED TO HAPPEN IN THIS FILE !!!

#
# Internal Go language support the rest of prelude.mk (and go-mod.mk)

# Possible values of GOHOSTOS/GOHOSTARCH:
# https://golang.org/doc/install/source#environment
_prelude.HAVE_GO = $(call lazyonce,_prelude.HAVE_GO,$(shell which go 2>/dev/null))
export GOHOSTOS   = $(call lazyonce,GOHOSTOS  ,$(if $(_prelude.HAVE_GO),$(shell go env GOHOSTOS  ),$(shell uname -s | tr A-Z a-z)))
export GOHOSTARCH = $(call lazyonce,GOHOSTARCH,$(if $(_prelude.HAVE_GO),$(shell go env GOHOSTARCH),$(patsubst i%86,386,$(patsubst x86_64,amd64,$(shell uname -m)))))

# All of this funny business with locking can be ditched once we drop
# support for Go 1.11.  (When removing it, be aware that go-mod.mk
# uses `_prelude.go.*` variables).
_prelude.go.GOPATH = $(call lazyonce,$(shell go env GOPATH))
_prelude.go.goversion = $(call lazyonce,_prelude.go.goversion,$(patsubst go%,%,$(filter go1%,$(shell go version))))
_prelude.go.lock = $(if $(filter 1.11 1.11.%,$(_prelude.go.goversion)),$(FLOCK)$(if $@, $(_prelude.go.GOPATH)/pkg/mod ))
