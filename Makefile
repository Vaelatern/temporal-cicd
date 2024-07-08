.PHONY: build clean

build: temporal-cicd signalgen

test:
	go test ./...

clean:
	rm -f temporal-cicd signalgen

signalgen: cmd/signalgen/*.go internal/*/*.go
	go build ./cmd/signalgen

temporal-cicd: cmd/temporal-cicd/*.go internal/*/*.go
	go build ./cmd/temporal-cicd

