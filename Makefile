.ONESHELL:
ifeq ($(OS),Windows_NT)
SHELL=C:/Program Files/Git/bin/bash.exe
build: windows
else
SHELL=/usr/bin/bash
build: linux
endif

windows: prepare
	export GOOS=windows
	export GOARCH=amd64
	export CGO_ENABLED=1
	go build -trimpath -a --tags "osusergo,netgo,sqlite_omit_load_extension" -ldflags '-s -w -extldflags "-static"' -o ./dist/shara.exe ./cmd/shara

linux: prepare
	export GOOS=linux
	export GOARCH=amd64
	export CGO_ENABLED=1
	go build -trimpath -a --tags "osusergo,netgo,sqlite_omit_load_extension" -ldflags '-s -w -extldflags "-static"' -o ./dist/shara ./cmd/shara

prepare:
	@rm -rf ./dist/
	@mkdir -p ./dist/configs/
	@cp ./configs/dist.shara.yaml ./dist/configs/shara.yaml

.PHONY: build windows linux
.DEFAULT_GOAL=build
