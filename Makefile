.PHONY: default
default: fmt test lint

.PHONY: fmt
fmt:
	gofmt -d -e -s .

.PHONY: test
test:
	go test -v -race ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: lint-docker
lint-docker:
	docker run -t --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.50.1 golangci-lint run