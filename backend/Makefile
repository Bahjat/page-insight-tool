.PHONY: lint test coverage benchmark check run

lint:
	golangci-lint run ./...

test:
	go test ./...

benchmark:
	go test -bench=BenchmarkCheckLinks -benchmem -count=1 -run='^\$$' ./internal/pageinsight/

check: lint test

run:
	export $$(cat .example.env | xargs) && go run ./cmd/api/
