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
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 appuser
COPY --from=builder /out/webapp /out/migrate /usr/local/bin/
USER appuser
EXPOSE 8080
# CMD, not ENTRYPOINT: fly.toml's release_command replaces it to run `migrate`.
CMD ["/usr/local/bin/webapp"]
