# syntax=docker/dockerfile:1
ARG GO_IMAGE=cgr.dev/chainguard/go:latest-dev
ARG STATIC_IMAGE=cgr.dev/chainguard/static:latest
FROM ${GO_IMAGE} AS source
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
FROM source AS test
RUN CGO_ENABLED=0 go test ./... && go vet ./...
FROM test AS build
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /tmp/server ./cmd/server
FROM ${STATIC_IMAGE} AS runtime
COPY --from=build --chown=65532:65532 /tmp/server /app/server
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/server"]
