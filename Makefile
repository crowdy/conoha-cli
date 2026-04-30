BINARY := conoha
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/crowdy/conoha-cli/cmd.version=$(VERSION)"

.PHONY: build test test-e2e lint clean install banner-version

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

# Update banner.svg version pill. Usage: make banner-version V=v0.7.2
banner-version:
	@if [ -z "$(V)" ]; then echo "usage: make banner-version V=vX.Y.Z" >&2; exit 2; fi
	@case "$(V)" in v[0-9]*.[0-9]*.[0-9]*) ;; *) echo "V must look like vX.Y.Z (got: $(V))" >&2; exit 2 ;; esac
	@sed -i.bak -E 's|(class="version">)v[0-9]+\.[0-9]+\.[0-9]+(</text>)|\1$(V)\2|' banner.svg && rm banner.svg.bak
	@echo "banner.svg updated to $(V)"
