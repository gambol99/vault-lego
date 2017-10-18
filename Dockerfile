FROM fedora:24
LABEL maintainer="catalin.cirstoiu@gmail.com"

ADD bin/vault-lego /vault-lego

WORKDIR "/"

ENTRYPOINT [ "/vault-lego" ]
