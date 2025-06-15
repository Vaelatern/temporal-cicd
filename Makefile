.PHONY: build clean

build: artifacts builder cache kickoff

test:
	go test ./...

clean:
	rm -rf artifacts builder cache kickoff

artifacts: cmd/artifacts internal/*/*.go internal/*/*/*.go
	go build ./cmd/artifacts

builder: cmd/builder internal/*/*.go internal/*/*/*.go
	go build ./cmd/builder

cache: cmd/cache internal/*/*.go internal/*/*/*.go
	go build ./cmd/cache

kickoff: cmd/kickoff internal/*/*.go internal/*/*/*.go
	go build ./cmd/kickoff

