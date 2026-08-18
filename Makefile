.PHONY: build test lint coverage ci clean baseline

COVERAGE_THRESHOLD := 90

build:
	go build -o bin/backstop ./cmd/backstop/

test:
	go test -race ./...

# baseline fetches the CI-published baseline artifact into .backstop/baseline.json.
#
# IT IS A PREREQUISITE OF THE RATCHET TESTS (cmd/backstop/bun_ratchet_flip_test.go),
# which read that file and FAIL loudly when it is absent — it is a generated,
# gitignored artifact CI publishes, not committed source, so a fresh clone has none.
#
# IT IS DELIBERATELY NOT A PREREQUISITE OF `test:` OR `ci:`. Wiring it in would make
# every local `make test` shell a GitHub API call; no local test run may hit the
# network by default (ISSUE-176). Run it yourself, once, when you need it.
#
# `build` is required, not decoration: the recipe runs ./bin/backstop, and bin/ is
# untracked and removed by `make clean`.
baseline: build
	./bin/backstop baseline pull

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
