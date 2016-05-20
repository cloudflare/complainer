FROM alpine:3.3

COPY . /go/src/github.com/cloudflare/complainer

RUN apk --update add go ca-certificates && \
    export GOPATH=/go GO15VENDOREXPERIMENT=1 && \
    go get github.com/cloudflare/complainer/... && \
    apk del go

ENTRYPOINT ["/go/bin/complainer"]
