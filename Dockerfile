# syntax=docker/dockerfile:1.4

FROM golang:1.21-alpine AS build
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/portctl ./cmd/portctl

FROM scratch
COPY --from=build /out/portctl /portctl
ENTRYPOINT ["/portctl"] 