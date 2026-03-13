.PHONY: build test lint clean

build:
	go build -o bin/backstop ./cmd/backstop/

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ dist/
