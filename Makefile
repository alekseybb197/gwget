PROJECT=$(shell basename "$(PWD)")
APPVERS=0.0.2
GOFLAGS=-ldflags="-w -s" -trimpath -ldflags "-X main.version=${APPVERS}"
GO111MODULE=on

default: build

.PHONY: build
build:
	go build ${GOFLAGS} -o bin/${PROJECT}

.PHONY: run
run:
	go run ${PROJECT}.go

.PHONY: clean
clean:
	go clean
	rm -rf bin

PHONY: fmt
fmt:
	gofumpt -w -s  .

PHONY: lint
lint:
	golangci-lint run -c .golang-ci.yml
