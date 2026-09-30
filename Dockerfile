# syntax=docker/dockerfile:1
# ---- build ----
# Cross-compiles on the build machine's native arch (fast multi-arch builds).
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH=amd64
# Version shown in the app (set by CI).
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Pure-Go SQLite driver: no CGO, fully static binary.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags="-s -w -X github.com/zachcurry13/recipebank/internal/version.Version=${VERSION}" \
    -o /out/recipebank ./cmd/recipebank

# ---- runtime ----
FROM alpine:3.22
LABEL org.opencontainers.image.source="https://github.com/ZachCurry13/recipebank" \
      org.opencontainers.image.description="RecipeBank: self-hosted family recipe bank with allergy checks"
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S -g 568 recipebank && adduser -S -u 568 -G recipebank recipebank \
 && mkdir -p /data && chown recipebank:recipebank /data
COPY --from=build /out/recipebank /usr/local/bin/recipebank
USER 568:568
ENV RECIPEBANK_ADDR=:8080 \
    RECIPEBANK_DATA_DIR=/data
VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/recipebank"]
