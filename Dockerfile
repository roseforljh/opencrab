FROM golang:1.25 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY .env.example ./.env.example

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /opencrab-api ./cmd/api

FROM debian:bookworm-slim
WORKDIR /app

COPY --from=builder /opencrab-api /usr/local/bin/opencrab-api
COPY .env.example /app/.env.example

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& mkdir -p /app/runtime

EXPOSE 8080

CMD ["/usr/local/bin/opencrab-api"]
