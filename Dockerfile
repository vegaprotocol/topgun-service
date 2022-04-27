FROM golang:1.16-alpine AS builder
ENV GOPROXY=direct GOSUMDB=off
WORKDIR /go/src/project
RUN apk add --no-cache ca-certificates git
ADD *.go go.* ./
ADD config config
ADD datastore datastore
ADD leaderboard leaderboard
ADD pricing pricing
ADD util util
ADD verifier verifier
RUN env CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o topgun-service .

# # #

FROM alpine:3.14
ENTRYPOINT ["/topgun-service"]

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/project/topgun-service /
