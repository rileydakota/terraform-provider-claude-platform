default: build

build:
	go build ./...

install:
	go install .

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./internal/provider/ -v -timeout 30m

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
	terraform fmt -recursive examples/

docs:
	cd tools && go generate ./...

generate: fmt docs

.PHONY: default build install test testacc vet lint fmt docs generate
