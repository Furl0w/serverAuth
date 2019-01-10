FROM golang:1.10-alpine as builder

COPY *.go /go/src/app/
COPY Gopkg.lock /go/src/app/
COPY Gopkg.toml /go/src/app/
WORKDIR /go/src/app
RUN apk add git
RUN apk add dep
RUN dep ensure -vendor-only
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o serverAuth *.go

FROM scratch

COPY --from=builder /go/src/app/serverAuth /app/serverAuth
CMD ["/app/serverAuth"]