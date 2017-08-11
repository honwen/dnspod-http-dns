FROM chenhw2/alpine:base
MAINTAINER CHENHW2 <https://github.com/chenhw2>

ARG VER=20170811
ARG URL=https://github.com/chenhw2/dnspod-http-dns/releases/download/v$VER/dnspod-http-dns_linux-amd64-$VER.tar.gz

RUN mkdir -p /usr/bin \
    && cd /usr/bin \
    && wget -qO- ${URL} | tar xz \
    && mv dnspod-http-dns_* dnspod-http-dns

USER nobody

ENV ARGS="--edns 119.29.29.29"

EXPOSE 5300
EXPOSE 5300/udp

CMD dnspod-http-dns -T -U ${ARGS}
