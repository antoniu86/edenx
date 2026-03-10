# EdenX Makefile
BINARY   = eden
MODULE   = github.com/edenx/eden
VERSION  = 0.6.0
LDFLAGS  = -ldflags "-X main.Version=$(VERSION) -s -w"

.PHONY: all build install uninstall deb release clean

all: build

## build: Build binary for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## install: Install eden to /usr/local/bin
install: build
	install -Dm755 $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed: /usr/local/bin/$(BINARY)"

## uninstall: Remove eden from /usr/local/bin
uninstall:
	rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstalled"

## deb: Build a .deb package (requires dpkg-deb)
deb: build
	mkdir -p .debpkg/usr/local/bin
	mkdir -p .debpkg/DEBIAN
	cp $(BINARY) .debpkg/usr/local/bin/
	cp debian/control .debpkg/DEBIAN/control
	sed -i "s/^Version:.*/Version: $(VERSION)/" .debpkg/DEBIAN/control
	dpkg-deb --build .debpkg $(BINARY)_$(VERSION)_amd64.deb
	rm -rf .debpkg
	@echo "Built: $(BINARY)_$(VERSION)_amd64.deb"

## release: Build binaries for all supported platforms into dist/
release:
	mkdir -p dist
	GOOS=linux  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64  .
	GOOS=linux  GOARCH=386    go build $(LDFLAGS) -o dist/$(BINARY)-linux-386     .
	GOOS=linux  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64  .
	GOOS=darwin GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .
	@echo "Release binaries in dist/"

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/ .debpkg/
	rm -f *.deb

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
