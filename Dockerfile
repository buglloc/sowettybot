FROM golang:1.20.6 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/sowettybot ./cmd/sowettybot

FROM debian:bookworm-backports

COPY --from=build /go/bin/sowettybot /usr/sbin/local
ADD example/config.yaml /etc/sowettybot/config.yaml

WORKDIR /var/run

ENTRYPOINT ["/usr/sbin/local/sowettybot start --config /etc/sowettybot/config.yaml"]
