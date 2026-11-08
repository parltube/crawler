FROM golang:1.26-alpine AS build
RUN go install github.com/windtf/wireproxy/cmd/wireproxy@v1.1.3

FROM alpine:3.22
RUN apk add --no-cache curl jq busybox-extras
COPY --from=build /go/bin/wireproxy /usr/local/bin/
COPY entrypoint.sh /usr/local/bin/
COPY rotate.cgi /www/cgi-bin/rotate
RUN chmod +x /usr/local/bin/entrypoint.sh /www/cgi-bin/rotate
ENTRYPOINT ["entrypoint.sh"]
