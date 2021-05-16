# Based on the Dockerfile suggested by Codefresh at
# https://codefresh.io/docs/docs/learn-by-example/golang/golang-hello-world/

FROM golang:1-alpine AS build_base
WORKDIR /tmp/dsddns
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o ./out/dsddns .

FROM alpine
COPY --from=build_base /tmp/dsddns/out/dsddns /app/dsddns
CMD ["/app/dsddns", "/etc/dsddns.conf"]