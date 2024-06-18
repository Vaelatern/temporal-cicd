.PHONY: build

build: temporal-cicd signalgen

signalgen: cmd/signalgen/*.go
	go build ./cmd/signalgen

temporal-cicd: cmd/temporal-cicd/*.go
	go build ./cmd/temporal-cicd

