GO      ?= go
GOBIN   ?= $(shell $(GO) env GOPATH)/bin
BINARY  := cchef
DIST    := dist

# Pinned so local and CI runs agree (avoids surprise failures on tool bumps).
GOLANGCI_VERSION := v2.12.2
GOSEC_VERSION := v2.27.1
GOVULNCHECK_VERSION := v1.5.0
GOCYCLO_VERSION := v0.6.0

# Cyclomatic-complexity threshold reported by `make complexity` (Go Report
# Card's gocyclo check flags functions above 15).
GOCYCLO_OVER := 15

.DEFAULT_GOAL := all
.PHONY: all build clean complexity cover fix fix-check fmt fmt-check install-tools lint sast sbom sbom-audit sbom-scan sec test vet vuln

## all: check formatting/modernization, vet, test, build, lint, and security (mirrors CI)
all: fmt-check fix-check vet test build lint sec

## build: compile the cchef binary into dist/
build:
	@mkdir -p $(DIST)
	$(GO) build -o $(DIST)/$(BINARY) .

## clean: remove build artifacts
clean:
	rm -rf $(DIST) coverage.out

## complexity: report functions above the cyclomatic-complexity threshold
## (Go Report Card's gocyclo check, run with maintained tooling; excludes
## tests). Informational — prints offenders and a count, never fails the build.
complexity:
	@out=$$($(GO) run github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION) \
		-over $(GOCYCLO_OVER) -ignore '_test\.go$$' internal/ cmd/); \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		echo; \
		echo "$$(echo "$$out" | wc -l | tr -d ' ') function(s) over complexity $(GOCYCLO_OVER)."; \
	else \
		echo "No functions over complexity $(GOCYCLO_OVER)."; \
	fi

## cover: run tests, write a cross-package coverage profile, and print the total
cover:
	$(GO) test -coverpkg=./... -covermode=atomic -coverprofile=coverage.out ./...
	@$(GO) tool cover -func=coverage.out | tail -1

## fix: apply go fix modernizations, iterating to a fixed point (bounded)
fix:
	@for i in 1 2 3 4 5; do \
		$(GO) fix ./... || { echo "make fix: 'go fix' failed (syntax error?) — aborting"; exit 1; }; \
		[ -z "$$($(GO) fix -diff ./... 2>/dev/null)" ] && exit 0; \
	done; \
	echo "make fix: did not converge after 5 iterations"; exit 1

## fix-check: fail if go fix would modernize any code (mirrors CI; run `make fix`)
fix-check:
	@diff=$$($(GO) fix -diff ./... 2>/dev/null); \
	if [ -n "$$diff" ]; then \
		echo "go fix would modernize code; run 'make fix':"; \
		echo "$$diff"; \
		exit 1; \
	fi

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

## sast: run gosec static analysis. By-design findings (weak-crypto ports,
## bounded byte/bit conversions, CLI file args) carry justified `// #nosec`
## annotations; -track-suppressions prints them for audit. G104 is excluded as
## errcheck (in golangci-lint) already governs unchecked errors.
sast:
	$(GO) run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) \
		-exclude=G104 -nosec-require-rules -nosec-require-justification \
		-track-suppressions ./...

## sbom: generate a CycloneDX SBOM for the module
sbom:
	@mkdir -p $(DIST)/sbom
	$(GOBIN)/cyclonedx-gomod mod -json -output $(DIST)/sbom/$(BINARY)-sbom.json

## sbom-audit: generate then scan the SBOM
sbom-audit: sbom sbom-scan

## sbom-scan: scan the generated SBOM for vulnerabilities
sbom-scan:
	$(GOBIN)/grype sbom:$(DIST)/sbom/$(BINARY)-sbom.json --output table --fail-on high

## sec: run all source-level security checks (gosec SAST + govulncheck vuln scan)
sec: sast vuln

## test: run all unit tests
test:
	$(GO) test ./...

## vet: run go vet static checks
vet:
	$(GO) vet ./...

## vuln: scan dependencies and the Go stdlib for known, reachable vulnerabilities
vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...
