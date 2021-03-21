SHELL=/bin/sh -e
NAME=kev
AUTHOR=appvia
AUTHOR_EMAIL=info@appvia.io
BUILD_TIME=$(shell date '+%s')
CURRENT_TAG=$(shell git tag --points-at HEAD)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GIT_SHA=$(shell git --no-pager describe --always --dirty)
GIT_LAST_TAG_SHA=$(shell git rev-list --tags='v[0.9]*.[0-9]*.[0-9]*' --max-count=1)
GIT_LAST_TAG=$(shell git describe --tags $(GIT_LAST_TAG_SHA))
HARDWARE=$(shell uname -m)
PACKAGES=$(shell go list ./...)
VETARGS ?= -asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -unsafeptr
ifeq ($(USE_GIT_VERSION),true)
	ifeq ($(CURRENT_TAG),)
		VERSION ?= $(GIT_LAST_TAG)-$(GIT_SHA)
	else
		VERSION ?= $(CURRENT_TAG)
	endif
else
	VERSION ?= $(GIT_LAST_TAG)
endif

LFLAGS ?= -X github.com/appvia/kev/pkg/${NAME}.Tag=${GIT_LAST_TAG} -X github.com/appvia/kev/pkg/${NAME}.GitSHA=${GIT_SHA} -X github.com/appvia/kev/pkg/${NAME}.Compiled=${BUILD_TIME} -X github.com/appvia/kev/pkg/${NAME}.Release=${VERSION} -X github.com/appvia/kev/pkg/${NAME}.GitBranch=${GIT_BRANCH}
CLI_PLATFORMS=darwin linux windows
CLI_ARCHITECTURES=amd64 arm64

.PHONY: test e2e authors changelog build release check vet golangci-lint setup-kind

default: build

golang:
	@echo "--> Go Version"
	@go version
	@echo "GOFLAGS: $$GOFLAGS"

build: golang
	@echo "--> Compiling the project ($(VERSION))"
	@mkdir -p bin
	go build -ldflags "${LFLAGS}" -tags=jsoniter -o bin/${NAME} cmd/${NAME}/*.go || exit 1;

package:
	@rm -rf ./release
	@mkdir ./release
	@$(MAKE) package-cli
	cd ./release && sha256sum * > kev.sha256sums

package-cli:
	@echo "--> Compiling CLI static binaries"
	CGO_ENABLED=0 gox -parallel=4 -arch="${CLI_ARCHITECTURES}" -os="${CLI_PLATFORMS}" -ldflags "-w ${LFLAGS}" -output=./release/{{.Dir}}-{{.OS}}-{{.Arch}} ./cmd/${NAME}/

push-release-packages:
	@echo "--> Pushing compiled CLI binaries to draft release (requires github token set in .gitconfig or GITHUB_TOKEN env variable)"
	ghr -replace -draft -n "Release ${VERSION}" "${VERSION}" ./release

release: build
	mkdir -p release
	gzip -c bin/${NAME} > release/${NAME}_${VERSION}_linux_${HARDWARE}.gz
	rm -f release/${NAME}

clean:
	@echo "--> Cleaning up the environment"
	rm -rf ./bin 2>/dev/null
	rm -rf ./release 2>/dev/null

authors:
	@echo "--> Updating the AUTHORS"
	git log --format='%aN <%aE>' | sort -u > AUTHORS

vet:
	@echo "--> Running go vet $(VETARGS) $(PACKAGES)"
	@go vet $(VETARGS) $(PACKAGES)

gofmt:
	@echo "--> Running gofmt check"
	@if gofmt -s -l $$(go list -f '{{.Dir}}' ./...) | grep -q \.go ; then \
		echo "You need to run the make format, we have file unformatted"; \
		gofmt -s -l $$(go list -f '{{.Dir}}' ./...); \
		exit 1; \
	fi

format:
	@echo "--> Running go fmt"
	@gofmt -s -w $$(go list -f '{{.Dir}}' ./...)

bench:
	@echo "--> Running go bench"
	@go test -bench=. -benchmem

verify-licence:
	@echo "--> Verifying the licence headers"
	@hack/verify-licence.sh

coverage:
	@echo "--> Running go coverage"
	@go test -coverprofile cover.out
	@go tool cover -html=cover.out -o cover.html

spelling:
	@echo "--> Checking the spelling"
	@find . -name "*.go" -type f -not -path "./ui/node_modules/*" | xargs go run github.com/client9/misspell/cmd/misspell -error -source=go *.go
	@find . -name "*.md" -type f -not -path "./ui/node_modules/*" | xargs go run github.com/client9/misspell/cmd/misspell -error -source=text *.md

golangci-lint:
	@echo "--> Checking against the golangci-lint"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run --timeout 5m -j 2 ./...

check:
	@echo "--> Running code checkers"
	@$(MAKE) golang
	@$(MAKE) gofmt
	@$(MAKE) golangci-lint
	@$(MAKE) spelling
	@$(MAKE) vet
	@$(MAKE) verify-licences
	@$(MAKE) check-generate-assets

test:
	@echo "--> Running the tests"
	@go test --cover -v $(PACKAGES)


all: test
	@echo "--> Performing all tests"
	@$(MAKE) bench
	@$(MAKE) coverage

e2e:
	@echo "--> Running e2e tests"
	@./e2e/bin/e2e.sh --build-cli true

setup-kind:
	@echo "--> Setting up kind"
	@./hack/e2e/setup-kind.sh

gen-cli-docs:
	@echo "--> Generate CLI reference docs"
	@./hack/doc-gen/cli/generate.sh

verify-cli-docs:
	@echo "--> Verify CLI reference docs"
	@./hack/doc-gen/cli/verify.sh

changelog: release
	git log $(shell git tag | tail -n1)..HEAD --no-merges --format=%B >> changelog
