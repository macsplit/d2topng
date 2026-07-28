BINARY := bin/d2topng

.PHONY: build test fmt vet clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/d2topng

test:
	go test ./...

fmt:
	gofmt -l .

vet:
	go vet ./...

clean:
	rm -rf bin
