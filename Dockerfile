FROM fedora:24
LABEL maintainer="gambol99@gmail.com"

ADD bin/vault-lego /vault-lego

WORKDIR "/"

ENTRYPOINT [ "/vault-lego" ]
