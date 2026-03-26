GO ?= /tmp/anarchy-tools/go-root/go/bin/go

.PHONY: test fmt

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...
