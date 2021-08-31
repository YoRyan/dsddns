FROM golang:1-alpine AS build
WORKDIR /src
COPY . .
RUN go mod download
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -a -installsuffix cgo -o ./out/dsddns .

FROM scratch
COPY --from=build /src/out/dsddns /main
USER 10001
ENTRYPOINT ["/main"]
CMD ["/etc/dsddns.conf"]