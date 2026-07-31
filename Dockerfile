# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

ARG NODE_IMAGE=node:20-alpine
ARG GO_IMAGE=golang:1.24.1-alpine
ARG RUNTIME_IMAGE=alpine:3.22

FROM --platform=$BUILDPLATFORM ${NODE_IMAGE} AS web-builder
WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

COPY frontend/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS go-builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY *.go ./
COPY internal/ ./internal/
COPY frontend/*.go ./frontend/
COPY --from=web-builder /src/frontend/dist/ ./frontend/dist/

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /out/yseren .

FROM --platform=$BUILDPLATFORM ${RUNTIME_IMAGE} AS runtime-files
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 yseren \
    && adduser -S -D -H -u 10001 -G yseren yseren \
    && mkdir -p /yseren-root/app /yseren-root/config /yseren-root/media

FROM ${RUNTIME_IMAGE} AS runtime
ARG VERSION=dev
ARG VCS_REF=unknown

LABEL org.opencontainers.image.title="YSeren Headless" \
      org.opencontainers.image.description="Zero-copy trusted-LAN media access through a browser" \
      org.opencontainers.image.source="https://github.com/Seacolour/YSeren" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}"

COPY --from=runtime-files /etc/passwd /etc/passwd
COPY --from=runtime-files /etc/group /etc/group
COPY --from=runtime-files /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=runtime-files --chown=10001:10001 /yseren-root/ /

COPY --from=go-builder --chown=10001:10001 /out/yseren /app/yseren
COPY --chown=10001:10001 docker/yseren.yaml /config/yseren.yaml

USER yseren:yseren
WORKDIR /app

EXPOSE 1479
STOPSIGNAL SIGTERM

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:1479/api/status || exit 1

ENTRYPOINT ["/app/yseren"]
CMD ["-config", "/config/yseren.yaml"]
