BINARY := crawler
PKG     := ./...

.PHONY: build run test lint tidy vet fmt docker clean

build:
	go build -trimpath -o bin/$(BINARY) ./cmd/$(BINARY)

run:
	go run ./cmd/$(BINARY)

test:
	go test -race -cover $(PKG)

vet:
	go vet $(PKG)

fmt:
	go fmt $(PKG)

lint:
	golangci-lint run

tidy:
	go mod tidy

docker:
	docker build -f build/package/Dockerfile -t $(BINARY):latest .

clean:
	rm -rf bin
