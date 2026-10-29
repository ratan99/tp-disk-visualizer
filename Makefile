BIN    := bin/tpdv
PREFIX ?= $(HOME)/.local
export CGO_ENABLED := 0

.PHONY: build install universal clean vet test screenshots

build:
	go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/tpdv

test:
	go test ./...

vet:
	gofmt -l . && go vet ./...

# Single binary that runs on both Apple Silicon and Intel Macs.
universal:
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/tpdv-arm64 ./cmd/tpdv
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/tpdv-amd64 ./cmd/tpdv
	lipo -create -output $(BIN) bin/tpdv-arm64 bin/tpdv-amd64
	rm bin/tpdv-arm64 bin/tpdv-amd64

# Regenerates docs/assets and docs/demo.html from fake demo data.
screenshots:
	scripts/screenshots.sh

install: build
	install -d $(PREFIX)/bin
	install -m 755 $(BIN) $(PREFIX)/bin/tpdv

clean:
	rm -rf bin
