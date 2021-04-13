FROM golang:1.16.2 AS builder
ENV GOPROXY=direct GOSUMDB=off
WORKDIR /go/src/project
ADD *.go go.* ./
ADD config config
ADD leaderboard leaderboard
ADD pricing pricing
ADD util util
ADD verifier verifier
RUN env CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o topgun-service .

# # #

FROM ubuntu:20.04
ENTRYPOINT ["/topgun-service"]
RUN \
	apt update && \
	DEBIAN_FRONTEND=noninteractive apt install -y ca-certificates && \
	rm -rf /var/lib/apt/lists
COPY --from=builder /go/src/project/topgun-service /
