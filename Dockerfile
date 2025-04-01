FROM docker.io/golang:1.24.1 AS build

# Create statically linked executable
ARG CGO_ENABLED=0

WORKDIR /app

# Cache deps before building
COPY go.mod go.sum ./
RUN go mod download \
 && go mod verify

COPY internal ./internal
COPY hetznerrobot ./hetznerrobot
COPY main.go ./

RUN go build -o /terraform-provider-hetznerrobot

FROM scratch

COPY --from=build /terraform-provider-hetznerrobot /terraform-provider-hetznerrobot
