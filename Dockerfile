FROM alpine:3.12
ENTRYPOINT ["/topgun-service"]
RUN apk add --update --no-cache ca-certificates
ADD topgun-service /
