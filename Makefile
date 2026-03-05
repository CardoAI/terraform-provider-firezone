default: install

build:
	go build -o terraform-provider-firezone

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/example/firezone/0.1.0/darwin_arm64
	cp terraform-provider-firezone ~/.terraform.d/plugins/registry.terraform.io/example/firezone/0.1.0/darwin_arm64/

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./... -v -timeout 120m

fmt:
	go fmt ./...

lint:
	golangci-lint run

.PHONY: build install test testacc fmt lint
