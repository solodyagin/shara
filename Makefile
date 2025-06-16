.ONESHELL:
.PHONY: windows linux clean all

# Check for required command tools to build or stop immediately
# TOOLS=git go find pwd
# K:=$(foreach exec, $(TOOLS), $(if $(shell which $(exec)), , $(error "No $(exec) in PATH")))

ifeq ($(OS),Windows_NT)
SHELL=C:/Program Files/Git/bin/bash.exe
default: windows
else
SHELL=/usr/bin/bash
default: linux
endif

ROOT_DIR:=$(shell dirname "$(realpath $(firstword $(MAKEFILE_LIST)))")
OUTPUT_DIR=$(ROOT_DIR)/bin
BASENAME=shara

GOOS=linux
GOARCH=amd64
TAGS=-tags "osusergo,netgo,sqlite_omit_load_extension"
LDLAGS=-ldflags "-s -w -extldflags '-static'"
EXECUTABLE=$(BASENAME)-$(GOOS)-$(GOARCH)

windows: GOOS=windows
windows: EXECUTABLE=$(BASENAME)-$(GOOS)-$(GOARCH).exe
windows: --build

linux: GOOS=linux
linux: EXECUTABLE=$(BASENAME)-$(GOOS)-$(GOARCH)
linux: --build

--prepare:
	@cd "$(ROOT_DIR)"
	@cp -n "./configs/example.shara.yaml" "./configs/shara.yaml"

--build: --prepare
	@cd "$(ROOT_DIR)"
	@export GOOS=$(GOOS)
	@export GOARCH=$(GOARCH)
	@export CGO_ENABLED=1
	@go build -trimpath $(TAGS) $(LDLAGS) -o "$(OUTPUT_DIR)/$(EXECUTABLE)" "./cmd/shara"

clean:
	@find $(OUTPUT_DIR) -name '$(BASENAME)[-?][a-zA-Z0-9]*[-?][a-zA-Z0-9]*' -delete

all:
	@make clean
	@make windows
	@make linux
