# Makefile
BINARY_NAME=bin/orchestrator

build:
	go build -o $(BINARY_NAME) cmd/orchestrator/main.go

clean:
	rm -rf bin/*

run: build
	./$(BINARY_NAME)