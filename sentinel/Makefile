.PHONY: build run test fmt vet clean

build:
	go build -o bin/sentinel ./cmd/sentinel

run:
	go run ./cmd/sentinel

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf bin/
