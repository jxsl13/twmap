.PHONY: check build vet

check: build vet

build:
	go build ./...

vet:
	go vet ./...
