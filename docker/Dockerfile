FROM golang:alpine AS builder

WORKDIR /go/src/github.com/stellargraph/bkcli/
COPY . /go/src/github.com/stellargraph/bkcli/

RUN apk add --no-cache --update make git \
      && go get -u github.com/kardianos/govendor \
      && /go/bin/govendor sync \
      && make linux

FROM alpine:3.8

COPY --from=builder /go/src/github.com/stellargraph/bkcli/bin/linux /app/

WORKDIR /app
ENTRYPOINT ["/app/bkcli"]
