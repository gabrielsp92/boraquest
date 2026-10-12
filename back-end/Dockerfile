# syntax=docker/dockerfile:1
# Production image for the API (deployed to Fly.io, see fly.toml). Keep the
# builder's Go version in step with go.mod.
FROM golang:1.26-alpine AS builder
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${VERSION}" -o /out/webapp ./cmd/webapp

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 appuser
COPY --from=builder /out/webapp /usr/local/bin/webapp
USER appuser
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/webapp"]
