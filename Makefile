.PHONY: build test directories

build: directories
	
	go build -o bin/logstore cmd/main.go

directories:
	mkdir -p bin/ 

test:
	go test -v ./...

test-help: build
	bin/logstore -h