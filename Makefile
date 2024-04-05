
PACKAGE := $(shell go list)

default: bin/reachability

bin/reachability: $(shell find . -name '*.go') go.mod go.sum
	go build -ldflags="-X '$(PACKAGE)/cmd.Version=$(shell git describe --tags --always --dirty)'" -o bin/reachability ./main.go

install: bin/reachability
	cp bin/reachability ~/.local/bin/reachability

clean:
	rm -rf bin
