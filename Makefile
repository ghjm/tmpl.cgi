PROGRAM := tmpl.cgi
SHELL := /bin/bash

GO_FILES := $(shell find . -name '*.go' ! -name '*_test.go' ! -name '*_gen.go')
PROGRAM_DEPS := Makefile go.mod go.sum $(GO_FILES)

$(PROGRAM): $(PROGRAM_DEPS)
	go build -o $(PROGRAM) main.go

.PHONY: lint
lint:
	@golangci-lint run

.PHONY: test
test:
	@go test ./... -count=1

.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: check-fmt
check-fmt:
	@if [[ $$(gofmt -l .) ]]; then echo Code needs to be formatted; exit 1; fi

.PHONY: clean
clean:
	rm -f $(PROGRAM)

