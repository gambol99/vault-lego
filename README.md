## **Vault Lego**
----

Vault-Lego provides a utility service similar to [kube-lego](https://github.com/jetstack/kube-lego) responsible for requesting certificates
from a [Vault](https://github.com/hashicorp/vault) PKI backend. The service gathers ingress resources, checks if the are vault enabled and is so, generates certificates for them. The service will also verify certificates which have already been generated have not expired; and renew any which are coming close to expiration.

#### **Usage**
```shell
[jest@starfury vault-lego]$ bin/vault-lego help
NAME:
   vault-lego - Requests certificates from vault on behalf of ingress resources

USAGE:
   vault-lego [global options] command [command options] [arguments...]

VERSION:
   v0.0.1 (git+sha: d7f8409-dirty)

AUTHOR(S):
   Rohith Jayawardene

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --host HOST, -H HOST                 the url for the vault service i.e. https://vault.vault.svc.cluster.local HOST [$VAULT_ADDR]
   --token TOKEN, -t TOKEN              the vault token to use when requesting a certificate TOKEN [$VAULT_TOKEN]
   --default-path PATH, -p PATH         the default vault path the pki exists on, e.g. pki/default/issue PATH (default: "pki/issue/default") [$VAULT_PKI_PATH]
   --kubeconfig PATH                    the path to a kubectl configuration file PATH (default: "/home/jest/.kube/config") [$KUBE_CONFIG]
   --kube-context CONTEXT               the kubernetes context inside the kubeconfig file CONTEXT [$KUBE_CONTEXT]
   --namespace NAMESPACE, -n NAMESPACE  the namespace the service should be looking, by default all NAMESPACE [$KUBE_NAMESPACE]
   --default-ttl TTL                    the default time-to-live of the certificate (can override by annontation) TTL (default: 48h0m0s) [$VAULT_PKI_TTL]
   --minimum-ttl value                  the minimum time-to-live on the certificate, ingress cannot request less then this (default: 24h0m0s) [$VAUILT_PKI_MIN_TTL]
   --json-logging                       whether to enable default json logging format, defaults to true
   --reconcilation-interval TTL         the interval between forced reconciliation events TTL (default: 5m0s) [$RECONCILCATION_INTERVAL]
   --verbose                            switch on verbose logging
   --help, -h                           show help
   --version, -v                        print the version
```

#### ** Ingress Annotations**
----
The following annotations are supported on the ingress resources.

|Name                 |type|  description|
|---------------------------|------|
|ingress.vault.io/tls|true or false|indicates if you wish to this ingress resource managed|
|ingress.kubernetes.io/path|string|override the default vault pki backend path|
|ingress.kubernetes.io/ttl|string|override the duration of the certificate e.g 10h|

#### **Example***

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
