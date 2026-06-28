GO      ?= go
GOBIN   ?= $(shell $(GO) env GOPATH)/bin
BINARY  := cchef
DIST    := dist

.DEFAULT_GOAL := all
.PHONY: all build clean cover fmt install-tools lint sbom sbom-audit sbom-scan test vet

## all: format, vet, test, and build
all: fmt vet test build

## build: compile the cchef binary into dist/
build:
	@mkdir -p $(DIST)
	$(GO) build -o $(DIST)/$(BINARY) .

## clean: remove build artifacts
clean:
	rm -rf $(DIST) coverage.out

## cover: run tests and write a cross-package coverage profile
cover:
	$(GO) test -coverpkg=./... -covermode=atomic -coverprofile=coverage.out ./...

## fmt: format all Go source
fmt:
	$(GO) fmt ./...

## install-tools: install lint + SBOM tooling
install-tools:
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	$(GO) install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
	curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b $(GOBIN)

## lint: run golangci-lint (see install-tools)
lint:
	$(GOBIN)/golangci-lint run

## sbom: generate a CycloneDX SBOM for the module
sbom:
	@mkdir -p $(DIST)/sbom
	$(GOBIN)/cyclonedx-gomod mod -json -output $(DIST)/sbom/$(BINARY)-sbom.json

## sbom-audit: generate then scan the SBOM
sbom-audit: sbom sbom-scan

## sbom-scan: scan the generated SBOM for vulnerabilities
sbom-scan:
	$(GOBIN)/grype sbom:$(DIST)/sbom/$(BINARY)-sbom.json --output table --fail-on high

## test: run all unit tests
test:
	$(GO) test ./...

## vet: run go vet static checks
vet:
	$(GO) vet ./...
