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

fmt:
	gofmt -w .
	terraform fmt -recursive examples/

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name claudeplatform

.PHONY: default build install test testacc vet fmt docs
