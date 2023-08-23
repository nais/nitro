BUILDTIME = $(shell date "+%s")
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git rev-parse --short HEAD)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/nais/onprem/nitro/pkg/version.Revision=$(LAST_COMMIT) -X github.com/nais/onprem/nitro/pkg/version.Date=$(DATE) -X github.com/nais/onprem/nitro/pkg/version.BuildUnixTime=$(BUILDTIME)

.PHONY: all install nitro-release-linux

all: install release-all

release-all: release-linux

install:
	go build -o ../nitro-cluster -ldflags="-s -w $(LDFLAGS)" "cmd/provision/main.go"
	go build -o ../nitro-runner -ldflags="-s -w $(LDFLAGS)" "cmd/provision/runner/main.go"

release-linux:
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=1 \
	go build -o nitro-linux -ldflags="-s -w $(LDFLAGS)" "cmd/provision/main.go"
