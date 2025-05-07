# Build
FROM docker.io/library/golang:1.23 AS build

WORKDIR /src
COPY . .
RUN make test && make build

FROM ghcr.io/vshn/appcat-cli:latest AS appcat-cli
# Runtime
FROM docker.io/appuio/oc:v4.18

COPY --from=appcat-cli /bin/appcat-cli /bin/appcat-cli
COPY --from=build /src/k8ify /bin/k8ify
