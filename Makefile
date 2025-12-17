
COVER_PKGS := $(shell go list ./... | grep -v /examples)

.PHONY: cover-html
cover-html:
	go test -coverprofile cover.out $(COVER_PKGS)
	go tool cover -html=cover.out

.PHONY: cover-text
cover-text:
	go test -coverprofile cover.out $(COVER_PKGS)
	go tool cover -func=cover.out
