# Makefile

IMAGE := docker.pkg.github.com/vegaprotocol/topgun-service/topgun-service:latest

.PHONY: default
default: help

.PHONY: docker_pull
docker_pull:
	@docker pull "${IMAGE}"

.PHONY: docker_build
docker_build:
	@docker build -t "${IMAGE}" .

.PHONY: docker_push
docker_push: docker_build
	@docker push "${IMAGE}"

.PHONY: build
build:
	@go build -o topgun-service .

.PHONY: test
test: ## Run tests
	@echo "No tests here."
