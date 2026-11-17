# syntax=docker/dockerfile:1

FROM golang:1.22.12-alpine AS build
WORKDIR /src
RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /catalogo ./cmd/catalogo

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build --link /catalogo /catalogo
USER nonroot:nonroot
EXPOSE 8091
ENTRYPOINT ["/catalogo"]
