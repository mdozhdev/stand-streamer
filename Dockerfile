FROM golang:1.19-alpine as build

WORKDIR /go/src

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY cmd/stand-streamer/*.go ./

RUN go build -o /go/bin/stand-streamer
FROM debian:bullseye-slim as app

COPY --from=build /go/bin/stand-streamer /usr/bin/stand-streamer

# ENTRYPOINT ["/usr/bin/stand-streamer"]
