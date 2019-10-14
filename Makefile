current_dir = $(shell pwd)

PROJECT = go-YouTokenToMe

PKG_OS = darwin linux

DOCKERFILES = Dockerfile:$(PROJECT)
DOCKER_ORG = "srcd"

# Including ci Makefile
CI_REPOSITORY ?= https://github.com/src-d/ci.git
CI_BRANCH ?= v1
CI_PATH ?= .ci
MAKEFILE := $(CI_PATH)/Makefile.main
$(MAKEFILE):
	git clone --quiet --depth 1 -b $(CI_BRANCH) $(CI_REPOSITORY) $(CI_PATH);
-include $(MAKEFILE)
ifdef TRAVIS_PULL_REQUEST
ifneq ($(TRAVIS_PULL_REQUEST), false)
GOTEST += -tags cipr
$(info Pull Request test mode: $(GOTEST))
endif
endif

fix-style:
	gofmt -s -w .
	goimports -w .

.ONESHELL:
.POSIX:
check-style:
	golint -set_exit_status ./...
	# Run `make fix-style` to fix style errors
	test -z "$$(gofmt -s -d .)"
	test -z "$$(goimports -d .)"
	go vet

install-dev-deps:
	go get -v golang.org/x/lint/golint github.com/mjibson/esc golang.org/x/tools/cmd/goimports


.PHONY: check-style