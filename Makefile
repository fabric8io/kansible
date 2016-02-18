ifndef GOPATH
$(error No GOPATH set)
endif

NAME := kansible
VERSION := $(shell cat version/VERSION)

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
GO_PACKAGES := ansible k8s log
MAIN_GO := kansible.go
KANSIBLE_BIN := $(BIN_DIR)/kansible

VERSION_PREFIX := $(shell git describe --tags --abbrev=0 2>/dev/null)

ifndef VERSION_PREFIX
  VERSION_PREFIX := v0.1.0
endif

VERSION := ${VERSION_PREFIX}+$(shell git rev-parse --short HEAD)

export GO15VENDOREXPERIMENT=1

ifndef VERSION
  VERSION := git-$(shell git rev-parse --short HEAD)
endif

build: $(MAIN_GO)
	go build -o $(KANSIBLE_BIN) -ldflags "-X main.version=${VERSION}" $<

bootstrap:
	go get -u github.com/golang/lint/golint github.com/mitchellh/gox
	glide up

build-all:
	gox -verbose \
	-ldflags "-X main.version=${VERSION}" \
	-os="linux darwin " \
	-arch="amd64 386" \
	-output="$(DIST_DIR)/{{.OS}}-{{.Arch}}/{{.Dir}}" .

clean:
	rm -rf $(DIST_DIR) $(BIN_DIR) release

dist: build-all
	@cd $(DIST_DIR) && \
	find * -type d -exec zip -jr kansible-$(VERSION)-{}.zip {} \; && \
	cd -

install: build
	install -d ${DESTDIR}/usr/local/bin/
	install -m 755 $(KANSIBLE_BIN) ${DESTDIR}/usr/local/bin/kansible

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
	go test -short ./ $(addprefix ./,$(GO_PACKAGES))

test: test-style
	go test -v ./ $(addprefix ./,$(GO_PACKAGES))

test-style:
	@if [ $(shell gofmt -e -l -s *.go $(GO_PACKAGES)) ]; then \
		echo "gofmt check failed:"; gofmt -e -l -s *.go $(GO_PACKAGES); exit 1; \
	fi
	@for i in . $(GO_PACKAGES); do \
		golint $$i; \
	done
	@for i in . $(GO_PACKAGES); do \
		go vet github.com/fabric8io/kansible/$$i; \
	done


release:
	rm -rf build release && mkdir build release
	for os in linux darwin ; do \
		GO15VENDOREXPERIMENT=1 CGO_ENABLED=0 GOOS=$$os ARCH=amd64 go build -o build/$(NAME)-$$os-amd64 $(BUILDFLAGS) -a $(NAME).go ; \
		tar --transform 's|^build/||' --transform 's|-.*||' -czvf release/$(NAME)-$(VERSION)-$$os-amd64.tar.gz build/$(NAME)-$$os-amd64 README.md LICENSE ; \
	done
	GO15VENDOREXPERIMENT=1 CGO_ENABLED=0 GOOS=windows ARCH=amd64 go build -o build/$(NAME)-$(VERSION)-windows-amd64.exe $(BUILDFLAGS) -a $(NAME).go
	zip --junk-paths release/$(NAME)-$(VERSION)-windows-amd64.zip build/$(NAME)-$(VERSION)-windows-amd64.exe README.md LICENSE
	go get -u github.com/progrium/gh-release
	gh-release create fabric8io/$(NAME) $(VERSION) $(BRANCH) $(VERSION)

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
				test-style
