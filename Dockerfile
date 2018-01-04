FROM alpine:3.7
LABEL maintainer="catalin.cirstoiu@gmail.com"

ADD bin/vault-lego /vault-lego

WORKDIR "/"

USER daemon
ENTRYPOINT [ "/vault-lego" ]
