# FILE IS AUTOMATICALLY MANAGED BY github.com/vegaprotocol/terraform//github
ARG GO_VERSION=1.19
ARG ALPINE_VERSION=3.16
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
RUN mkdir /build
WORKDIR /build
ADD . .
RUN apk add make git gcc g++
RUN make build
FROM alpine:${ALPINE_VERSION}
# USER nonroot:nonroot
# COPY --chown=nonroot:nonroot bin/topgun-service /topgun-service
COPY --from=builder /build/bin/topgun-service /topgun-service
