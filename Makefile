GO ?= /tmp/anarchy-tools/go-root/go/bin/go
IMAGE ?= anarchy-api:dev

.PHONY: test fmt image image-tar

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

image:
	podman build -t $(IMAGE) .

image-tar:
	podman save -o /tmp/anarchy-api-dev.tar $(IMAGE)
