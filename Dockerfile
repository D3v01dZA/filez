FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod main.go index.html ./
RUN go build -o filez .

FROM alpine:3.20
COPY --from=builder /src/filez /usr/local/bin/filez
VOLUME /data
ENV STORAGE_DIR=/data
EXPOSE 8080
ENTRYPOINT ["filez"]
