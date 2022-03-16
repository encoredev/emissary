# syntax=docker/dockerfile:1.4
FROM --platform=$BUILDPLATFORM golang:1.18 AS build
WORKDIR /src

# Copy over our mod and sum files, then run go mod download so we can cache the dependancies as three container layers
COPY --link go.mod go.sum ./
COPY --link server/go.mod server/go.sum ./server/
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    go mod download && \
    cd server && go mod download

# Now copy over the rest of the source code
COPY --link . .

# Test the code
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    go vet -v

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    go test -v

# Now build the app for the target OS / architecture
ARG TARGETOS
ARG TARGETARCH
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    cd server && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/emissary .

# Now we build the final image
FROM gcr.io/distroless/static
COPY --from=build /out/emissary /

LABEL org.opencontainers.image.authors="support@encore.dev" \
		org.opencontainers.image.vendor="Encoretivity AB" \
		org.opencontainers.image.description="The Encore Emissary"

ENV EMISSARY_HTTP_PORT=80
ENV EMISSARY_ALLOWED_PROXY_TARGETS="[]"
ENV EMISSARY_AUTH_KEYS="[]"

EXPOSE 80/tcp

USER nonroot:nonroot

ENTRYPOINT ["/emissary"]
