/*
Copyright 2016 Rohith Jayawardene
All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	extensions_v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

// reconcileIngress is responsible for processing the ingress resources
func (c *controller) reconcileIngress() {
	// step: rate limit us
	c.rate.Accept()

	// step: retrieve a list of ingress resources
	list, err := c.ingressList()
	if err != nil {
		return
	}

	// step: check if the ingress resources have changed
	resources := c.resources.Load().(*extensions_v1beta1.IngressList)
	if reflect.DeepEqual(resources, list) {
		logrus.Debugf("nothing to do, the ingress resource have not changed")
		return
	}
	// update the resources
	c.resources.Store(list)

	for _, x := range list.Items {
		// step: check if the ingress resource is vault enabled
		enabled, found := x.GetAnnotations()[AnnotationVaultTLS]
		if !found || (enabled != "true" && enabled != "True") {
			logrus.WithFields(logrus.Fields{
				"name":      x.Name,
				"namespace": x.Namespace,
			}).Debug("skipping ingress resource, not enabled")
			continue
		}

		// step: validate the ingress resource is valid
		if err := isIngressOK(&x); err != nil {
			logrus.WithFields(logrus.Fields{
				"name":      x.Name,
				"namespace": x.Namespace,
				"error":     err.Error(),
			}).Errorf("invalid ingress resource")
			continue
		}

		// step: we iterate the tls configs and check the certificate exists
		for _, tls := range x.Spec.TLS {
			err := func() error {
				// step: check if the secret already exists for this namespace?
				found, err := c.hasSecret(tls.SecretName, x.Namespace)
				if err != nil {
					return err
				}
				if found {
					// check if the certificate is coming up for renewal
					expiring, err := c.checkCertificateExpiring(x.Name, x.Namespace, tls.SecretName)
					if err != nil {
						return err
					}
					// is is close or expired?
					if !expiring {
						logrus.WithFields(logrus.Fields{
							"name":      x.Name,
							"namespace": x.Namespace,
							"hosts":     strings.Join(tls.Hosts, ","),
							"secret":    tls.SecretName,
						}).Debug("certificate not near or expired")

						return nil
					}
					logrus.WithFields(logrus.Fields{
						"name":      x.Name,
						"namespace": x.Namespace,
						"hosts":     strings.Join(tls.Hosts, ","),
						"secret":    tls.SecretName,
					}).Info("certificate is or has expired, attempting to renew")
				}

				// step: make a request for a certificate
				if err := c.makeCertificateRequest(&x, &tls); err != nil {
					return err
				}
				// step: spit out some logging
				logrus.WithFields(logrus.Fields{
					"name":      x.Name,
					"namespace": x.Namespace,
					"hosts":     strings.Join(tls.Hosts, ","),
					"secret":    tls.SecretName,
				}).Info("adding vault certifacte for ingress resource")

				return nil
			}()
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"name":      x.Name,
					"namespace": x.Namespace,
					"secret":    tls.SecretName,
					"error":     err.Error(),
				}).Error("unable to process ingress tls config")
			}
		}
	}
}

// makeCertificateRequest is responsible for making a request to the store for a certificate
func (c *controller) makeCertificateRequest(ingress *extensions_v1beta1.Ingress, tls *extensions_v1beta1.IngressTLS) error {
	// step: make a request for the certificate from vault
	path := c.config.defaultPath
	if override, found := ingress.GetAnnotations()[AnnotationVaultPath]; found {
		path = override
	}
	ttl := c.config.defaultCertTTL
	if override, found := ingress.GetAnnotations()[AnnotationVaultTTL]; found {
		tm, err := time.ParseDuration(override)
		if err == nil {
			ttl = tm
		}

	}
	logrus.WithFields(logrus.Fields{
		"name":      ingress.Name,
		"namespace": ingress.Namespace,
		"path":      path,
		"ttl":       ttl.String(),
		"hosts":     strings.Join(tls.Hosts, ","),
		"secret":    tls.SecretName,
	}).Info("generating certificate for ingress resource")

	// step: we need to request a certificate from vault for this
	cert, err := c.generateCertificate(path, ttl, tls.Hosts)
	if err != nil {
		return err
	}
	// step: inject the certificate into the namespace
	return c.addSecret(tls.SecretName, ingress.Namespace, cert)
}

// checkCertificateExpiring is responsible for checking if a certificate is about to expire
func (c *controller) checkCertificateExpiring(name, namespace, secret string) (bool, error) {
	// step: grab the secret from kubernetes
	cert, err := c.getSecret(secret, namespace)
	if err != nil {
		return false, err
	}

	// step: spit out some logging
	logrus.WithFields(logrus.Fields{
		"name":      name,
		"namespace": namespace,
		"secret":    secret,
	}).Debugf("checking if the certifacte is expiring")

	// step: check if the certificate is expiring
	expired, err := isCertificateExpiring(cert.cert, time.Duration(-6*time.Hour))
	if err != nil {
		return false, err
	}

	return expired, nil
}

// isIngressOK is responsible for validating the ingress resource
func isIngressOK(ing *extensions_v1beta1.Ingress) error {
	if len(ing.Spec.TLS) <= 0 {
		return errors.New("no tls settings")
	}
	for i, x := range ing.Spec.TLS {
		if len(x.Hosts) <= 0 {
			return fmt.Errorf("tls settings for item %d has not hosts", i)
		}
		if x.SecretName == "" {
			return fmt.Errorf("tls settings for item: %d has no secret name defined", i)
		}
	}

	return nil
}
