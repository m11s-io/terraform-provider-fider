default: build

.PHONY: build
build:
	go build ./...

.PHONY: install
install:
	go install .

.PHONY: test
test:
	go test ./... -v -timeout 120s

.PHONY: generate
generate:
	go generate ./...

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: lint
lint:
	golangci-lint run ./...
