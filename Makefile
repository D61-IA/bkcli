BINARY=buildkite-cli
SRC=cmd/buildkite-cli/*.go
BUILDCMD=go build
VERSION=$$(cat VERSION)

.DEFAULT_GOAL := version

PLATFORMS := windows linux darwin
os = $(word 1, $@)

.PHONY: $(PLATFORMS)
$(PLATFORMS):
	mkdir -p release/$(os)
	GOOS=$(os) GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o release/$(os)/$(BINARY) $(SRC)

.PHONY: release
	release: windows linux darwin

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: clean
clean:
	rm -rf release/*
