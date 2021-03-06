FROM golang:1.10-alpine as builder

RUN apk --update add ca-certificates
COPY Gopkg.lock /go/src/serverAuth/
COPY Gopkg.toml /go/src/serverAuth/
WORKDIR /go/src/serverAuth
RUN apk add git
RUN apk add dep
RUN dep ensure -vendor-only
COPY app/ /go/src/serverAuth/app/
COPY tools/ /go/src/serverAuth/tools/
WORKDIR /go/src/serverAuth/app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o serverAuth

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/src/serverAuth/app/serverAuth /app/serverAuth
CMD ["/app/serverAuth"]