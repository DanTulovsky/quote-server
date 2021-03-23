

#build stage
FROM golang:alpine AS builder

# install git and ca certs for debugger
RUN apk update && apk add --no-cache git ca-certificates make build-base

COPY . $GOPATH/src/github.com/DanTulovsky/quote-server
WORKDIR $GOPATH/src/github.com/DanTulovsky/quote-server

ADD . /go/src/github.com/DanTulovsky/quote-server

# fetch deps
RUN go get -d -v ./...

# optimized binary
# RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/run
RUN go build -tags musl -o /go/bin/run_http cmd/http/run.go
RUN go install -tags musl -v cmd/http/run.go

# final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates bash curl
COPY --from=builder /go/bin/run_http /go/bin/run_http
# COPY --from=builder /go/src/github.com/DanTulovsky/web-static/data /data/

# run this command automatically
# can do: docker run -it --rm --entrypoint=/bin/ash image...
ENTRYPOINT ["/go/bin/run_http"]

RUN mkdir -p /logs

# this is just documentation really
EXPOSE 8080
