GO := go
DOCKER := docker
TAG?=$(shell git rev-parse HEAD)
REGISTRY?=probablynot
IMAGE=eks-image-updater
GOOS=linux

all: build

build:
	@echo ">> building using docker"
	@$(DOCKER) build --platform linux/amd64 -t ${REGISTRY}/${IMAGE}:${TAG} -f Dockerfile .
	@$(DOCKER) tag ${REGISTRY}/${IMAGE}:${TAG} ${REGISTRY}/${IMAGE}:latest

push:
	docker push ${REGISTRY}/${IMAGE}:${TAG}
	docker push ${REGISTRY}/${IMAGE}:latest

bin:
	@echo ">> building local binary"
	CGO_ENABLED=0 GOOS=${GOOS} go build -a -ldflags '-extldflags "-static"' -o eks-image-updater

.PHONY: all build push bin
