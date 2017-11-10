FROM golang:alpine as builder
RUN apk add --update git
RUN go get github.com/chenhw2/dnspod-http-dns

FROM chenhw2/alpine:base
MAINTAINER CHENHW2 <https://github.com/chenhw2>

# /usr/bin/dnspod-http-dns
COPY --from=builder /go/bin /usr/bin

USER nobody

ENV ARGS="--edns 119.29.29.29"

EXPOSE 5300
EXPOSE 5300/udp

CMD dnspod-http-dns -T -U ${ARGS}
