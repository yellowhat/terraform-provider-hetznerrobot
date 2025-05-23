.DEFAULT_GOAL := help
# renovate: datasource=docker depName=docker.io/golang
GOLANG_VER = 1.24.3
# renovate: datasource=github-releases depName=hashicorp/terraform
TERRAFORM_VER = 1.12.1

.PHONY: it
it: ## Start interactive go container
	podman run \
		--interactive \
		--tty \
		--rm \
		--publish 9090:9090 \
		--volume .:/data:z \
		--workdir /data \
		"docker.io/golang:$(GOLANG_VER)"

.PHONY: docs
docs: ## Generate docs
	podman run \
		--interactive \
		--tty \
		--rm \
		--volume .:/data:z \
		--workdir /data \
		--entrypoint bash \
		"docker.io/golang:$(GOLANG_VER)" -c " \
			apt-get update \
		 && apt-get install -y unzip \
		 && curl --output /tmp/t.zip https://releases.hashicorp.com/terraform/$(TERRAFORM_VER)/terraform_$(TERRAFORM_VER)_linux_amd64.zip \
		 && unzip /tmp/t.zip -d /usr/local/bin/ \
		 && cd tools \
		 && go generate ./... \
		"

.PHONY: help
help: ## Makefile Help Page
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[\/\%a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-21s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST) 2>/dev/null
