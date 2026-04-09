.ONESHELL:
.PHONY: build build_linux build_windows clean all

ROOT_DIR:=$(shell dirname "$(realpath $(firstword $(MAKEFILE_LIST)))")
OUTPUT_DIR:=$(ROOT_DIR)/bin
BASENAME:=shara

GOOS=linux
GOARCH=amd64
CGO_ENABLED=0
TAGS=
LDFLAGS=-ldflags "-s -w"
EXT=
OUTPUT=$(OUTPUT_DIR)/$(BASENAME)_$(GOOS)-$(GOARCH)$(EXT)

TARGET=build_linux

ifeq ($(OS),Windows_NT)
	SHELL="$(PROGRAMFILES)/Git/bin/bash.exe"
	TARGET=build_windows
else
	SHELL=/usr/bin/bash
endif

build: $(TARGET)

build_linux: --compile

build_windows: GOOS=windows
build_windows: EXT=.exe
build_windows: --compile

--compile:
	@cd "$(ROOT_DIR)"
	@cp -n "./configs/example.shara.yaml" "./configs/shara.yaml"
	@export GOOS=$(GOOS)
	@export GOARCH=$(GOARCH)
	@export CGO_ENABLED=$(CGO_ENABLED)
	@go build -trimpath $(TAGS) $(LDFLAGS) -o "$(OUTPUT)" "./cmd/shara"

clean:
	@find $(OUTPUT_DIR) -type f -name '*' -delete

all:
	@make clean
	@make linux
	@make windows
