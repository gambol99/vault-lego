FROM alpine:3.4
MAINTAINER Rohith <gambol99@gmail.com>

RUN apk update && \
    apk add ca-certificates

ADD bin/vault-lego /vault-lego

WORKDIR "/"

ENTRYPOINT [ "/vault-lego" ]
