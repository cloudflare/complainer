# Alpine is used instead of a scratch image in order to bring in SSL support and CAs.
FROM alpine:3.4

COPY . /go/src/github.com/cloudflare/complainer

RUN apk --update add go ca-certificates && \
    export GOPATH=/go GO15VENDOREXPERIMENT=1 && \
    go get github.com/cloudflare/complainer/... && \
    apk del go

RUN mkdir /var/log/mesos-complainer
RUN chown nobody:nobody /var/log/mesos-complainer

USER nobody

ENTRYPOINT ["/go/bin/complainer"]
