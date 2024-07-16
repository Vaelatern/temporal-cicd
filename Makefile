.PHONY: build clean

build: temporal-cicd signalgen gitolite-send-event

test:
	go test ./...

clean:
	rm -f temporal-cicd signalgen gitolite-send-event

gitolite-send-event: cmd/gitolite-send-event/*.go internal/*/*.go internal/*/*/*.go
	go build ./cmd/gitolite-send-event

signalgen: cmd/signalgen/*.go internal/*/*.go internal/*/*/*.go
	go build ./cmd/signalgen

temporal-cicd: cmd/temporal-cicd/*.go internal/*/*.go internal/*/*/*.go
	go build ./cmd/temporal-cicd

