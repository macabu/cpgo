.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: run
run:
	go run ./cmd/cpgo/main.go -verbose -githubToken=${GITHUB_TOKEN}

.PHONY: test
test:
	go test -race -cover -coverprofile=coverage.out -covermode=atomic ./internal/...
