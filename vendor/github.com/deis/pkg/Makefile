# Short name: Short name, following [a-zA-Z_], used all over the place.
# Some uses for short name:
# - Docker image name
# - Kubernetes service, rc, pod, secret, volume names
SHORT_NAME := pkg

# Enable vendor/ directory support.
export GO15VENDOREXPERIMENT=1

# SemVer with build information is defined in the SemVer 2 spec, but Docker
# doesn't allow +, so we use -.
VERSION := 0.0.1-$(shell date "+%Y%m%d%H%M%S")

# Common flags passed into Go's linker.
LDFLAGS := "-s -X main.version=${VERSION}"

NV_PKGS := $(shell glide nv)

all: build test

# This builds .a files, which will be placed in $GOPATH/pkg
build:
	go build ${NV_PKGS}

test:
	go test ${NV_PKGS}


.PHONY: all build test
