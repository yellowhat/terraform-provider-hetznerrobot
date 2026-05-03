# renovate: datasource=docker depName=docker.io/golang

GOLANG_VER := "1.26.2"

# renovate: datasource=github-releases depName=hashicorp/terraform

TERRAFORM_VER := "1.12.1"

set shell := ["bash", "-o", "errexit", "-o", "nounset", "-o", "pipefail", "-c"]

@_:
    just --list --unsorted

# Start interactive go container
env:
    podman run \
        --interactive \
        --tty \
        --rm \
        --publish 9090:9090 \
        --volume .:/data:z \
        --workdir /data \
        "docker.io/golang:{{ GOLANG_VER }}"

# Generate docs
docs:
    podman run \
        --interactive \
        --tty \
        --rm \
        --volume .:/data:z \
        --workdir /data \
        --entrypoint bash \
        "docker.io/golang:{{ GOLANG_VER }}" -c """ \
            apt-get update \
         && apt-get install --yes unzip \
         && curl --output /tmp/t.zip https://releases.hashicorp.com/terraform/{{ TERRAFORM_VER }}/terraform_{{ TERRAFORM_VER }}_linux_amd64.zip \
         && unzip /tmp/t.zip -d /usr/local/bin/ \
         && cd tools \
         && go generate ./... \
        """
