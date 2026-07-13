VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint docs clean install

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o flawless .

test:
	go vet ./...
	go test ./...

lint:
	go vet ./...

install:
	go install -trimpath -ldflags '$(LDFLAGS)' .

docs:
	cd docs && npm install --no-fund --no-audit && npm run build

clean:
	rm -f flawless
	rm -rf docs/dist docs/node_modules
