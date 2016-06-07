FROM alpine:3.4

RUN apk --update add go ca-certificates

COPY . /go/src/github.com/cloudflare/complainer

RUN export GOPATH=/go GO15VENDOREXPERIMENT=1 && \
    go get github.com/cloudflare/complainer/...

USER nobody

ENTRYPOINT ["/go/bin/complainer"]
