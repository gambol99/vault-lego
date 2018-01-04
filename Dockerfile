FROM alpine:3.7
LABEL maintainer="catalin.cirstoiu@gmail.com"

ADD bin/vault-lego /vault-lego

WORKDIR "/"

# user daemon
USER 2:2
ENTRYPOINT [ "/vault-lego" ]
