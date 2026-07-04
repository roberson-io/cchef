GO      ?= go
GOBIN   ?= $(shell $(GO) env GOPATH)/bin
BINARY  := cchef
DIST    := dist

# Pinned so local and CI lint runs agree (avoids surprise failures on tool bumps).
GOLANGCI_VERSION := v2.12.2

.DEFAULT_GOAL := all
.PHONY: all build clean cover fmt fmt-check install-tools lint sbom sbom-audit sbom-scan test vet

## all: check formatting, vet, test, build, and lint (mirrors CI)
all: fmt-check vet test build lint

## build: compile the cchef binary into dist/
build:
	@mkdir -p $(DIST)
	$(GO) build -o $(DIST)/$(BINARY) .

## clean: remove build artifacts
clean:
	rm -rf $(DIST) coverage.out

## cover: run tests, write a cross-package coverage profile, and print the total
cover:
	$(GO) test -coverpkg=./... -covermode=atomic -coverprofile=coverage.out ./...
	@$(GO) tool cover -func=coverage.out | tail -1

## fmt: format all Go source (gofmt + import grouping, per .golangci.yml)
fmt:
	$(GOBIN)/golangci-lint fmt

## fmt-check: fail if any Go source is not gofmt-clean (mirrors CI)
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files are not gofmt-clean:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

## install-tools: install lint + SBOM tooling
install-tools:
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	$(GO) install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
	curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b $(GOBIN)

## lint: enforce the Google Go Style Guide subset via golangci-lint (.golangci.yml)
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
