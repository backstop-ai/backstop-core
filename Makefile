.PHONY: build test lint coverage ci clean

COVERAGE_THRESHOLD := 90

build:
	go build -o bin/backstop ./cmd/backstop/

test:
	go test -race ./...

lint:
	go tool golangci-lint run ./...

coverage:
	@go test -coverprofile=cover.out ./... > /dev/null 2>&1
	@COVERAGE=$$(go tool cover -func=cover.out | grep '^total:' | awk '{print int($$3)}'); \
	echo "Coverage: $${COVERAGE}%  (threshold: $(COVERAGE_THRESHOLD)%)"; \
	if [ "$$COVERAGE" -lt "$(COVERAGE_THRESHOLD)" ]; then \
		echo "FAIL: coverage below threshold"; \
		exit 1; \
	fi
	@rm -f cover.out

ci: lint test coverage

clean:
	rm -rf bin/ dist/ cover.out
