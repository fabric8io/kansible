ifndef GOPATH
$(error No GOPATH set)
endif

NAME := kansible
CURRENT_VERSION := $(shell cat version/VERSION)

GO_VERSION := $(shell go version)
ROOT_PACKAGE := $(shell go list .)
BRANCH  := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
BUILDFLAGS := -ldflags

XBUILDFLAGS := -ldflags \
  " -X $(ROOT_PACKAGE)/version.Version='$(VERSION)'\
		-X $(ROOT_PACKAGE)/version.Revision='$(REV)'\
		-X $(ROOT_PACKAGE)/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/version.GoVersion='$(GO_VERSION)'"

BIN_DIR := bin
DIST_DIR := _dist
GO := GO15VENDOREXPERIMENT=1 go
GO_PACKAGES := $(shell $(GO) list ./... | grep -v /vendor/)
SRCS := $(shell find -type d -not -wholename './.git*' -not -wholename './vendor*' -not -wholename '.')
MAIN_GO := kansible.go
KANSIBLE_BIN := $(BIN_DIR)/kansible

VERSION := $(CURRENT_VERSION)+$(shell git rev-parse --short HEAD)

LINTERS := --disable-all --enable=vet --enable=golint --enable=errcheck --enable=ineffassign --enable=interfacer --enable=goimports --enable=gofmt

build: $(MAIN_GO)
	$(GO) build -o $(KANSIBLE_BIN) -ldflags "-X main.version=$(VERSION)" $<

bootstrap:
	$(GO) get -u github.com/golang/lint/golint github.com/mitchellh/gox github.com/alecthomas/gometalinter
	GO15VENDOREXPERIMENT=1 glide up

build-all:
	gox -verbose \
	-ldflags "-X main.version=$(VERSION)" \
	-os="linux darwin windows" \
	-arch="amd64 386" \
	-output="$(DIST_DIR)/{{.OS}}-{{.Arch}}/{{.Dir}}" .

clean:
	rm -rf $(DIST_DIR) $(BIN_DIR) release

dist: build-all
	@cd $(DIST_DIR) && \
	find * -type d -exec zip -jr kansible-$(VERSION)-{}.zip {} \; && \
	cd -

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
	@export TMP=$(shell mktemp -d) && cp -r vendor $${TMP}/src && GOPATH=$${TMP}:$${GOPATH} gometalinter --vendor --deadline=60s $(LINTERS) ./... || true

docker-scratch:
	gox -verbose -ldflags "-X main.version=$(VERSION)" -os="linux" -arch="amd64" \
	   -output="bin/kansible-docker" .
	docker build -f Dockerfile.scratch -t "fabric8/kansible:scratch" .

release:
	rm -rf build release && mkdir build release
	for os in linux darwin ; do \
		GOOS=$$os ARCH=amd64 $(GO) build -o build/$(NAME)-$$os-amd64 $(BUILDFLAGS) -a $(NAME).go ; \
		tar --transform 's|^build/||' --transform 's|-.*||' -czvf release/$(NAME)-$(CURRET_VERSION)-$$os-amd64.tar.gz build/$(NAME)-$$os-amd64 README.md LICENSE ; \
	done
	CGO_ENABLED=0 GOOS=windows ARCH=amd64 $(GO) build -o build/$(NAME)-$(CURRENT_VERSION)-windows-amd64.exe $(BUILDFLAGS) -a $(NAME).go
	zip --junk-paths release/$(NAME)-$(VERSION)-windows-amd64.zip build/$(NAME)-$(CURRENT_VERSION)-windows-amd64.exe README.md LICENSE
	$(GO) get -u github.com/progrium/gh-release
	gh-release create fabric8io/$(NAME) $(CURRENT_VERSION) $(BRANCH) $(CURRENT_VERSION)

.PHONY: release clean

.PHONY: bootstrap \
				build \
				build-all \
				clean \
				dist \
				install \
				prep-bintray-json \
				quicktest \
				release \
				test \
				test-charts \
				lint
