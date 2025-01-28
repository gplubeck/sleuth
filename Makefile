#Makefile for StatSleuth golang project
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test 
GOCLEAN=$(GOCMD) clean

SRC_FILES= main.go in_memory_store.go server.go service.go scheduler.go version.go
SRC= $(addprefix src/, $(SRC_FILES))

# Binary name
BINARY_NAME=slueth
BUILD_TIME := $(shell date "+%Y-%m-%d %H:%M:%S")

.PHONY: all build test clean run help production

## help: prints this message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## all: runs build and test
all: build test 

## build: builds debug version of slueth and places in bin/
build:
	@echo "files: ${SRC_FILES}"
	$(GOBUILD) -ldflags="-s -X 'main.Version=DEBUG' -X 'main.BuildTime=$(BUILD_TIME)'" -o bin/$(BINARY_NAME) $(SRC)

## test: runs all tests
test:
	$(GOTEST) -v ./...

## clean: removes binary from bin/
clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)

## run: builds and runs debug binary
run:
	@make build
	bin/$(BINARY_NAME)

## production: Makes stripped, production build for linux, amd64
production:
	@echo -n 'Enter Build version number (e.g. 0.0.5): ' && read ans &&\
		GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="-s -X 'main.Version=$$ans' -X 'main.BuildTime=$(BUILD_TIME)'" -o bin/$(BINARY_NAME) $(SRC)
