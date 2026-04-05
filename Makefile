VERSION := 0.1.0
BINARY := td-cli
GOFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build clean install

build:
	go build $(GOFLAGS) -o $(BINARY).exe ./cmd/td-cli/

build-all:
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -o dist/$(BINARY)-windows-amd64.exe ./cmd/td-cli/
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o dist/$(BINARY)-darwin-amd64 ./cmd/td-cli/
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o dist/$(BINARY)-darwin-arm64 ./cmd/td-cli/
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/td-cli/

install: build
	cp $(BINARY).exe $(GOPATH)/bin/$(BINARY).exe

clean:
	rm -f $(BINARY).exe
	rm -rf dist/
