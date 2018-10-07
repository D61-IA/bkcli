BINARY=bkcli
SRC=cmd/bkcli/*.go
BUILDCMD=go build
VERSION=$$(cat VERSION)

.DEFAULT_GOAL := version

PLATFORMS := windows linux darwin
os = $(word 1, $@)

.PHONY: $(PLATFORMS)
$(PLATFORMS):
	mkdir -p bin/$(os)
	GOOS=$(os) GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o bin/$(os)/$(BINARY) $(SRC)

.PHONY: sync
sync:
	govendor sync

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: clean
clean:
	rm -rf bin/*
