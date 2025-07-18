.PHONY: test cover clean fmt lint build check vet fmt fmt-check report-check

MODULE_NAME := github.com/restayway/stx

test:
	go test -v -race -coverprofile=coverage.out ./...

cover: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

clean:
	rm -f coverage.out coverage.html
	go clean -cache

fmt:
	go fmt ./...

lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

build:
	go build -v ./...

check: fmt lint test
	@echo "All checks passed!"

bench:
	go test -bench=. -benchmem ./...

deps:
	go mod download
	go mod tidy

update:
	go get -u ./...
	go mod tidy

ci: deps check cover
	@echo "CI pipeline completed successfully!"


fmt:
	go fmt ./...
	gofmt -s -w .

fmt-check:
	@echo "Checking code formatting..."
	@if [ "$$(gofmt -s -d . | wc -l)" -eq 0 ]; then \
		echo "✓ All files are properly formatted"; \
	else \
		echo "✗ Some files need formatting:"; \
		gofmt -s -d .; \
		echo "Run 'make fmt' to fix formatting issues"; \
		exit 1; \
	fi

# Run go vet
vet:
	go vet ./...

# Check for Go Report Card issues
report-check:
	@echo "=== Go Report Card Quality Check ==="
	@echo ""
	@echo "1. Checking gofmt..."
	@make fmt-check
	@echo ""
	@echo "2. Checking go vet..."
	@go vet ./...
	@echo "✓ go vet passed"
	@echo ""
	@echo "3. Checking gocyclo (if available)..."
	@if command -v gocyclo >/dev/null 2>&1; then \
		gocyclo -over 15 .; \
		if [ $$? -eq 0 ]; then echo "✓ gocyclo passed"; fi; \
	else \
		echo "⚠ gocyclo not installed (optional)"; \
	fi
	@echo ""
	@echo "4. Checking ineffassign (if available)..."
	@if command -v ineffassign >/dev/null 2>&1; then \
		ineffassign .; \
		if [ $$? -eq 0 ]; then echo "✓ ineffassign passed"; fi; \
	else \
		echo "⚠ ineffassign not installed (optional)"; \
	fi
	@echo ""
	@echo "5. Checking misspell (if available)..."
	@if command -v misspell >/dev/null 2>&1; then \
		misspell -error .; \
		if [ $$? -eq 0 ]; then echo "✓ misspell passed"; fi; \
	else \
		echo "⚠ misspell not installed (optional)"; \
	fi
	@echo ""
	@echo "✓ Go Report Card check completed!"