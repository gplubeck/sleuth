#Makefile for StatSleuth golang project

#PACKAGES = $(shell go list ./...)
PACKAGE_DIR = cmd/
RUN= *.go
PACKAGE = cmd/main.go
BUILD_DIR ?= $(CURDIR)/bins
OUTPUT ?= sleuth 

CGO_ENABLED ?= 0

all: build test install

.PHONY: all build test install

build:
	go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(OUTPUT)  $(PACKAGE)

run:
	go run $(BUILD_FLAGS) $(PACKAGE_DIR)$(RUN)

install:
	go install $(BUILD_DIR)/$(OUTPUT)
