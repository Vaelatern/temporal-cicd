.PHONY: build build-raw clean test

DOCKER?=docker

build-raw: artifacts builder cache kickoff deployer slop-ui

build: docker-build-artifacts docker-build-builder docker-build-cache docker-build-kickoff docker-build-deployer docker-build-slop-ui

docker-build-artifacts:
	$(DOCKER) build -f Dockerfile.artifacts .

docker-build-builder:
	$(DOCKER) build -f Dockerfile.builder .

docker-build-cache:
	$(DOCKER) build -f Dockerfile.cache .

docker-build-kickoff:
	$(DOCKER) build -f Dockerfile.kickoff .

docker-build-deployer:
	$(DOCKER) build -f Dockerfile.deployer .

docker-build-slop-ui:
	$(DOCKER) build -f Dockerfile.slop-ui .

test:
	go test ./...

clean:
	rm -rf artifacts builder cache kickoff deployer slop-ui

artifacts: cmd/artifacts internal/*/*.go internal/*/*/*.go
	go build ./cmd/artifacts

builder: cmd/builder internal/*/*.go internal/*/*/*.go
	go build ./cmd/builder

cache: cmd/cache internal/*/*.go internal/*/*/*.go
	go build ./cmd/cache

kickoff: cmd/kickoff internal/*/*.go internal/*/*/*.go
	go build ./cmd/kickoff

deployer: cmd/deployer internal/*/*.go internal/*/*/*.go
	go build ./cmd/deployer

slop-ui: cmd/slop-ui internal/*/*.go internal/*/*/*.go
	go build ./cmd/slop-ui
