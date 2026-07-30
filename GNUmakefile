default: fmt vet build test

build:
	go build -v ./...

install: build
	go install -v ./...

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

fmt:
	gofmt -s -w -e .

vet:
	go vet ./...

.PHONY: build install test testacc fmt vet
