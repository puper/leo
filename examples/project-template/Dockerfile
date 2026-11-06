# syntax=docker/dockerfile:1
FROM golang:1.25.5 AS builder
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
WORKDIR /app
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -mod=readonly -trimpath -ldflags "-s -w -X server/internal/pkg/version.version=${VERSION} -X server/internal/pkg/version.gitCommit=${COMMIT} -X server/internal/pkg/version.buildTime=${BUILD_TIME}" -o /out/server .

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/server /app/server
USER 65532:65532
EXPOSE 8888
ENTRYPOINT ["/app/server", "serve"]
CMD ["--config", "/app/config/config.toml"]
