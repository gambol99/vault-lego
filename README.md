[![Build Status](https://travis-ci.org/gambol99/vault-lego.svg?branch=master)](https://travis-ci.org/gambol99/vault-lego)
[![GoDoc](http://godoc.org/github.com/gambol99/vault-lego?status.png)](http://godoc.org/github.com/gambol99/vault-lego)
[![Docker Repository on Quay](https://quay.io/repository/gambol99/vault-lego/status "Docker Repository on Quay")](https://quay.io/repository/gambol99/vault-lego)
[![GitHub version](https://badge.fury.io/gh/gambol99%2Fvault-lego.svg)](https://badge.fury.io/gh/gambol99%2Fvault-lego)

## **Vault Lego**
----

Vault-Lego provides a utility service similar to [kube-lego](https://github.com/jetstack/kube-lego) responsible for requesting certificates
from a [Vault](https://github.com/hashicorp/vault) PKI backend. The service gathers ingress resources, checks if the are vault enabled and is so, generates certificates for them. The service will also verify certificates which have already been generated have not expired; and renew any which are coming close to expiration.

#### **Usage**
```shell
$ ./vault-lego help
NAME:
   vault-lego - Requests certificates from vault on behalf of ingress resources

USAGE:
   vault-lego [global options] command [command options] [arguments...]

VERSION:
   v0.0.4 (git+sha: no gitsha provided)

AUTHOR:
   Rohith Jayawardene

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --host HOST, -H HOST                 the url for the vault service i.e. https://vault.vault.svc.cluster.local HOST [$VAULT_ADDR]
   --token TOKEN, -t TOKEN              the vault token to use when requesting a certificate TOKEN [$VAULT_TOKEN]
   --default-path PATH, -p PATH         the default vault path the pki exists on, e.g. pki/default/issue PATH (default: "pki/issue/default") [$VAULT_PKI_PATH]
   --kubeconfig PATH                    the path to a kubectl configuration file PATH (default: "/Users/ccirstoiu/.kube/config") [$KUBE_CONFIG]
   --kube-context CONTEXT               the kubernetes context inside the kubeconfig file CONTEXT [$KUBE_CONTEXT]
   --namespace NAMESPACE, -n NAMESPACE  the namespace the service should be looking, by default all NAMESPACE [$KUBE_NAMESPACE]
   --default-ttl TTL                    the default time-to-live of the certificate (can override by annontation) TTL (default: 48h0m0s) [$VAULT_PKI_TTL]
   --minimum-ttl value                  the minimum time-to-live on the certificate, ingress cannot request less then this (default: 24h0m0s) [$VAULT_PKI_MIN_TTL]
   --refresh-ttl value                  refresh certificates when time-to-live goes below this threshold (default: 6h0m0s) [$VAULT_PKI_REFRESH_TTL]
   --json-logging                       whether to enable default json logging format, defaults to true
   --reconcilation-interval TTL         the interval between forced reconciliation events TTL (default: 5m0s) [$RECONCILCATION_INTERVAL]
   --verbose                            switch on verbose logging
   --help, -h                           show help
   --version, -v                        print the version
```

#### **Ingress Annotations**
----
The following annotations are supported on the ingress resources.

|Name                 |type|  description|
|---------------------------|------|-----|
|ingress.vault.io/tls|true or false|indicates if you wish to this ingress resource managed|
|ingress.vault.io/path|string|override the default vault pki backend path|
|ingress.vault.io/ttl|string|override the duration of the certificate e.g 10h|

#### **Kubernetes Setup**

Note, I am assuming you are using [service accounts](http://kubernetes.io/docs/user-guide/service-accounts/) and a [abac policy](http://kubernetes.io/docs/admin/authorization/).

- **Create the service account**
```shell
[jest@starfury ~]$ cat service-account.yml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vault-lego

[jest@starfury ~]$ kubectl --namespace=ingress apply -f service-account.yml
```

- **Update the authorization policy**

Note: your probably was want to be more specific in the below policy, i.e. readonly to ingress resources and read-write on the all namespace secrets, but you get the gist.

```JSON
{"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy", "spec": {"user":"system:serviceaccount:ingress:vault-lego","namespace":"*","resource":"*","apiGroup":"*"}}
```

- **Deploy the service**

```shell
[jest@starfury ~]$ cat deployment.yml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: vault-lego
spec:
  replicas: 1
  template:
    metadata:
      labels:
        name: vault-lego
    spec:
      containers:
      - name: vault-lego
        image: quay.io/gambol99/vault-lego:latest
        resources:
          limits:
            cpu: 100m
            memory: 100Mi
        env:
        - name: VAULT_ADDR
          value: "https://vault.vault.svc.cluster.local:8200"
        - name: VAULT_TOKEN
          valueFrom:
            secretKeyRef:
              name: vault-token
              key: token

[jest@starfury ~]$ kubectl --namespace=ingress apply -f deployment.yml
```

#### **Example Ingress Resource**

```YAML
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    ingress.vault.io/tls: "true"
    ingress.vault.io/ttl: "10h"
    ingress.vault.io/path: platform/pki/default
  name: clair
spec:
  rules:
  - host: site.example.com
    http:
      paths:
      - backend:
          serviceName: site
          servicePort: 443
  tls:
  - hosts:
    - site.example.com
    secretName: tls
```

#### **Example Vault setup**

```bash
vault mount pki
vault mount-tune -max-lease-ttl=8760h pki
vault write pki/root/generate/internal common_name="My Vault ROOT" ttl=8760h
vault write pki/roles/default allow_any_name=true max_ttl="720h"

```
