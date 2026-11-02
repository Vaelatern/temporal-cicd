.PHONY: build build-raw clean test upload raw-upload

DOCKER?=docker
SERVICES=artifacts builder cache kickoff deployer slop-ui

build-raw: artifacts builder cache kickoff deployer slop-ui

build: docker-build-artifacts docker-build-builder docker-build-cache docker-build-kickoff docker-build-deployer docker-build-slop-ui

docker-build-artifacts:
	$(DOCKER) build -t artifacts:latest -f Dockerfile.artifacts .

docker-build-builder:
	$(DOCKER) build -t builder:latest -f Dockerfile.builder .

docker-build-cache:
	$(DOCKER) build -t cache:latest -f Dockerfile.cache .

docker-build-kickoff:
	$(DOCKER) build -t kickoff:latest -f Dockerfile.kickoff .

docker-build-deployer:
	$(DOCKER) build -t deployer:latest -f Dockerfile.deployer .

docker-build-slop-ui:
	$(DOCKER) build -t slop-ui:latest -f Dockerfile.slop-ui .

upload:
	for s in $(SERVICES); do \
		$(DOCKER) tag "$$s:latest" "$(UPLOAD_REGISTRY)/$(PROJECT)/$$s:$(TAG)"; \
		$(DOCKER) push "$(UPLOAD_REGISTRY)/$(PROJECT)/$$s:$(TAG)"; \
	done

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

raw-upload:
	for s in $(SERVICES); do \
		oras push "$(UPLOAD_REGISTRY)/$(PROJECT)/artifacts/$$s:$(TAG)" "$$s"; \
	done
