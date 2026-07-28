BINARY := bin/d2topng
SERVER_BINARY := bin/d2topng-server

.PHONY: build build-server test fmt vet clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/d2topng

build-server:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(SERVER_BINARY) ./cmd/d2topng-server

test:
	go test ./...

fmt:
	gofmt -l .

vet:
	go vet ./...

clean:
	rm -rf bin
