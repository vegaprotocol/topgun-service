# Makefile

IMAGE := docker.pkg.github.com/vegaprotocol/topgun-service/topgun-service:latest
CONTAINER := topgun-service

.PHONY: default
default: help

.PHONY: docker_pull
docker_pull: ## Pull docker image from github registry
	@docker pull "${IMAGE}"

.PHONY: docker_build
docker_build: ## Build local docker image
	@docker build -t "${IMAGE}" .

.PHONY: docker_push
docker_push: docker_build ## Push docker image to github image registry
	@docker push "${IMAGE}"

#.PHONY: docker_run
#docker_run: ## Run docker image
#	@docker run -d --name=${CONTAINER} -p 8333:8333 -v "$${PWD}/includelist.csv:/includelist.csv" "${IMAGE}" -addr=0.0.0.0:8333 -endpoint=https://veganode.example.com/query -includelist=...

.PHONY: docker_stop
docker_stop: ## Stop docker container
	@docker ps -q --filter name="${CONTAINER}" | xargs -r docker stop

.PHONY: docker_rm
docker_rm: ## Remove docker container
	@docker ps -qa --filter name="${CONTAINER}" | xargs -r docker rm

.PHONY: docker_stoprm
docker_stoprm: | docker_stop docker_rm ## Stop and remove docker container

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build (dynamic) binary
	@go build -o topgun-service .

.PHONY: build-static
build-static: # Build static binary
	@env CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o topgun-service .

.PHONY: test
test: ## Run tests
	@echo "No tests here."
