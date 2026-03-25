ARG REGISTRY_PREFIX=
ARG BUN_IMAGE=oven/bun:1.3.8
ARG GO_IMAGE=golang:1.25.1-alpine
ARG RUNTIME_IMAGE=debian:bookworm-slim
ARG GOPROXY=https://proxy.golang.org,direct
ARG NPM_REGISTRY=

FROM ${REGISTRY_PREFIX}${BUN_IMAGE} AS web-builder
ARG APP_VERSION=dev
ARG NPM_REGISTRY
WORKDIR /build

COPY web/package.json web/bun.lock ./
RUN if [ -n "$NPM_REGISTRY" ]; then \
      npm_config_registry="$NPM_REGISTRY" bun install --frozen-lockfile; \
    else \
      bun install --frozen-lockfile; \
    fi

COPY web/ ./
RUN DISABLE_ESLINT_PLUGIN=true VITE_REACT_APP_VERSION=${APP_VERSION} bun run build

FROM ${REGISTRY_PREFIX}${GO_IMAGE} AS go-builder
ENV GO111MODULE=on \
    CGO_ENABLED=0

ARG APP_VERSION=dev
ARG GOPROXY
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    GOPROXY=${GOPROXY}

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY common ./common
COPY constant ./constant
COPY controller ./controller
COPY dto ./dto
COPY i18n ./i18n
COPY logger ./logger
COPY middleware ./middleware
COPY model ./model
COPY oauth ./oauth
COPY pkg/cachex ./pkg/cachex
COPY relay ./relay
COPY router ./router
COPY service ./service
COPY setting ./setting
COPY types ./types
COPY main.go ./
COPY --from=web-builder /build/dist ./web/dist

RUN go build \
    -trimpath \
    -ldflags "-s -w -X github.com/roseforljh/opencrab/common.Version=${APP_VERSION}" \
    -o /opencrab

FROM ${REGISTRY_PREFIX}${RUNTIME_IMAGE}
ENV TZ=UTC

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=go-builder /opencrab /opencrab

EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/opencrab"]
