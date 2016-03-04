ifndef GOPATH
$(error No GOPATH set)
endif

NAME := kansible
VERSION := $(shell cat version/VERSION)
REVISION=$(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
BRANCH=$(shell git rev-parse --abbrev-ref HEAD 2> /dev/null || echo 'unknown')
HOST=$(shell hostname -f)
BUILD_DATE=$(shell date +%Y%m%d-%H:%M:%S)
GO_VERSION=$(shell go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')

ROOT_PACKAGE := $(shell go list .)

BUILDFLAGS := -ldflags \
  " -X $(ROOT_PACKAGE)/version.Version='$(VERSION)'\
		-X $(ROOT_PACKAGE)/version.Revision='$(REVISION)'\
		-X $(ROOT_PACKAGE)/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/version.BuildUser='${USER}@$(HOST)'\
		-X $(ROOT_PACKAGE)/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/version.GoVersion='$(GO_VERSION)'"

BIN_DIR := bin
DIST_DIR := _dist
GO := GO15VENDOREXPERIMENT=1 go
GO_PACKAGES := $(shell $(GO) list ./... | grep -v /vendor/)
SRCS := $(shell find . -path ./vendor -prune -o -name '*.go')
MAIN_GO := kansible.go
KANSIBLE_BIN := $(BIN_DIR)/kansible

LINTERS := --disable-all --enable=vet --enable=golint --enable=errcheck --enable=ineffassign --enable=interfacer --enable=goimports --enable=gofmt

build: $(MAIN_GO)
	$(GO) build -o $(KANSIBLE_BIN) $(BUILDFLAGS) $<

bootstrap:
	$(GO) get -u github.com/golang/lint/golint github.com/mitchellh/gox github.com/alecthomas/gometalinter github.com/fabric8io/gobump
	gometalinter --install --update

update-vendor:
	GO15VENDOREXPERIMENT=1 glide up --update-vendored

build-all:
	gox -verbose \
	$(BUILDFLAGS) \
	-os="linux darwin freebsd netbsd openbsd solaris windows" \
	-arch="amd64 386" \
	-output="$(DIST_DIR)/{{.OS}}-{{.Arch}}/{{.Dir}}" .

clean:
	rm -rf $(DIST_DIR) $(BIN_DIR) release

install: build
	install -d $(DESTDIR)/usr/local/bin/
	install -m 755 $(KANSIBLE_BIN) $(DESTDIR)/usr/local/bin/kansible

prep-bintray-json:
# TRAVIS_TAG is set to the tag name if the build is a tag
ifdef TRAVIS_TAG
	@jq '.version.name |= "$(VERSION)"' _scripts/ci/bintray-template.json | \
		jq '.package.repo |= "kansible"' > _scripts/ci/bintray-ci.json
else
	@jq '.version.name |= "$(VERSION)"' _scripts/ci/bintray-template.json \
		> _scripts/ci/bintray-ci.json
endif

quicktest:
	$(GO) test -short $(GO_PACKAGES)

test:
	$(GO) test -v $(GO_PACKAGES)

lint:
	@echo "Linting does not currently fail the build but is likely to do so in future - fix stuff you see, when you see it please"
	@export TMP=$(shell mktemp -d) && cp -r vendor $${TMP}/src && GOPATH=$${TMP}:$${GOPATH} GO15VENDOREXPERIMENT=1 gometalinter --vendor --deadline=60s $(LINTERS) ./... || rm -rf $${TMP}} || true

docker-scratch:
	gox -verbose $(BUILDFLAGS) -os="linux" -arch="amd64" \
	   -output="bin/kansible-docker" .
	docker build -f Dockerfile.scratch -t "fabric8/kansible:scratch" .

release: build-all
	rm -rf build release && mkdir build release
	for os in linux darwin freebsd netbsd openbsd solaris ; do \
		for arch in amd64 386 ; do \
			tar --transform "s|^$(DIST_DIR)/$$os-$$arch/||" -czf release/$(NAME)-$(VERSION)-$$os-$$arch.tar.gz $(DIST_DIR)/$$os-$$arch/$(NAME) README.md LICENSE ; \
		done ; \
	done
	for arch in amd64 386 ; do \
		zip -q --junk-paths release/$(NAME)-$(VERSION)-windows-$$arch.zip $(DIST_DIR)/windows-$$arch/$(NAME).exe README.md LICENSE ; \
	done ; \
	$(GO) get -u github.com/progrium/gh-release
	gh-release create fabric8io/$(NAME) $(VERSION) $(BRANCH) $(VERSION)

bump:
	gobump minor -f version/VERSION

bump-patch:
	gobump patch -f version/VERSION

.PHONY: release clean

.PHONY: bootstrap \
				build \
				build-all \
				clean \
				install \
				prep-bintray-json \
				quicktest \
				release \
				test \
				test-charts \
				lint \
				bump \
				bump-patch \
				update-vendor
