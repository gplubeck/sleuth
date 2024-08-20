#Makefile for StatSleuth golang project
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test 
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get

SRC_FILES= main.go in_memory_store.go server.go service.go scheduler.go
SRC= $(addprefix src/, $(SRC_FILES))

# Binary name
BINARY_NAME=slueth

.PHONY: all build test clean run

all: build test 

build:
	$(GOBUILD) -o bin/$(BINARY_NAME) $(SRC)

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)

run:
	@make build
	bin/$(BINARY_NAME)
