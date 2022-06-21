# syntax=docker/dockerfile:1

FROM golang:1.18-alpine as build

WORKDIR /go/src/telice
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/telice

FROM gcr.io/distroless/static-debian11
COPY --from=build /go/src/telice/index.html /go/src/telice/logo.svg /go/bin/telice /
ENTRYPOINT ["/telice"]