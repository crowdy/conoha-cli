BINARY := conoha
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/crowdy/conoha-cli/cmd.version=$(VERSION)"

.PHONY: build test test-e2e lint clean install

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

test:
	go test ./... -v

# DinD E2E harness (tests/e2e/run.sh). Requires docker.
test-e2e:
	bash tests/e2e/run.sh

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
