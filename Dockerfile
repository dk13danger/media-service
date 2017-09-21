# ===== GOLANG BUILDER IMAGE
FROM golang:1.9 as builder

ADD . ${GOPATH}/src/github.com/dk13danger/media-service
WORKDIR ${GOPATH}/src/github.com/dk13danger/media-service

# Install glide
RUN curl https://glide.sh/get | sh

# Install dependencies and build executable file
RUN apt-get update \
 && apt-get install --no-install-recommends -y make sqlite3 \
 && make build

# ===== FINAL IMAGE
FROM debian:jessie
COPY --from=builder /media-service.o /media-service

RUN apt-get update \
 && apt-get install --no-install-recommends -y sqlite3

ENTRYPOINT ["/media-service", "-config", "/etc/media-service/config.yml"]