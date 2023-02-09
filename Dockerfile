# Build
FROM golang:latest AS build

ENV HOME=/k8ify

WORKDIR ${HOME}

COPY . ${HOME}

RUN make test && go build -v .

# Runtime
FROM docker.io/appuio/oc:v4.12

COPY --from=build k8ify /bin/
