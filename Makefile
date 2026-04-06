GO ?= go

.PHONY: test
test:
	$(GO) test ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: release
release:
	bash scripts/build-release.sh $(VERSION)

.PHONY: verify-release
verify-release:
	bash scripts/verify-release.sh $(VERSION)
