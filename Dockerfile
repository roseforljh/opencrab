ARG REGISTRY_PREFIX=
ARG BUN_IMAGE=oven/bun:latest
ARG GO_IMAGE=golang:alpine
ARG RUNTIME_IMAGE=debian:bookworm-slim
ARG GOPROXY=https://proxy.golang.org,direct

FROM ${REGISTRY_PREFIX}${BUN_IMAGE} AS builder

ARG APP_VERSION=dev

WORKDIR /build
COPY web/package.json .
COPY web/bun.lock .
RUN bun install
COPY ./web .
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=${APP_VERSION} bun run build

FROM ${REGISTRY_PREFIX}${GO_IMAGE} AS builder2
ENV GO111MODULE=on CGO_ENABLED=0

ARG APP_VERSION=dev
ARG GOPROXY

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc
ENV GOPROXY=${GOPROXY}

WORKDIR /build

ADD go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=builder /build/dist ./web/dist
RUN go build -ldflags "-s -w -X github.com/QuantumNous/opencrab/common.Version=${APP_VERSION}" -o opencrab

FROM ${REGISTRY_PREFIX}${RUNTIME_IMAGE}

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/opencrab /
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/opencrab"]
