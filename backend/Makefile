.PHONY: lint test check run

lint:
	golangci-lint run ./...

test:
	go test ./...

check: lint test

run:
	export $$(cat .example.env | xargs) && go run ./cmd/api/
