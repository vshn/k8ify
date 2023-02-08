# Build
FROM golang:latest AS build

ENV HOME=/k8ify

WORKDIR ${HOME}

COPY . ${HOME}

RUN go build -v .

# Runtime
FROM docker.io/appuio/oc:v4.11

COPY --from=build k8ify /bin/
