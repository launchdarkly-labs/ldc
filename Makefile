GOLANGCI_VERSION=v1.12.5

SHELL=/bin/bash

test: lint
	go test ./...

lint:
	./bin/golangci-lint run ./...

init:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s $(GOLANGCI_VERSION)

RELEASE_CMD=curl -sL https://git.io/goreleaser | bash -s -- --rm-dist

publish:
	$(RELEASE_CMD)

publish-snapshot:
	$(RELEASE_CMD)

release:
	$(RELEASE_CMD) --skip-publish --skip-validate

release-snapshot:
	$(RELEASE_CMD) --skip-publish --skip-validate --snapshot

integration-test:
    echo "Not implemented yet"

.PHONY: docker init lint publish release test test-centos test-debian test-docker test-all test-docker-conf test-docker-standalone
