.ONESHELL:
ifeq ($(OS),Windows_NT)
SHELL=C:/Program Files/Git/bin/bash.exe
build: windows
else
SHELL=/usr/bin/bash
build: linux
endif

ROOT_DIR := $(shell dirname "$(realpath $(firstword $(MAKEFILE_LIST)))")

windows: prepare
	export GOOS=windows
	export GOARCH=amd64
	export CGO_ENABLED=1
	go build -trimpath -a --tags "osusergo,netgo,sqlite_omit_load_extension" -ldflags '-s -w -extldflags "-static"' -o "${ROOT_DIR}/dist/shara.exe" "${ROOT_DIR}/cmd/shara"

linux: prepare
	export GOOS=linux
	export GOARCH=amd64
	export CGO_ENABLED=1
	go build -trimpath -a --tags "osusergo,netgo,sqlite_omit_load_extension" -ldflags '-s -w -extldflags "-static"' -o "${ROOT_DIR}/dist/shara" "${ROOT_DIR}/cmd/shara"

prepare:
	find "${ROOT_DIR}/dist/" ! -name 'shara.yaml' -type f -exec rm -f {} +

.PHONY: build windows linux
.DEFAULT_GOAL=build
