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
	"reflect"

	"github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

// reconcileIngress is responsible for processing the ingress resources
func (c *controller) reconcileIngress() {
	// step: retrieve a list of ingress resources
	list, err := c.ingressList()
	if err != nil {
		return
	}

	// step: check if the ingress resources have changed
	resources := c.resources.Load().(*extensions.IngressList)
	if reflect.DeepEqual(resources, list) {
		// nothing has changed we can continue
		return
	}

	// step: iterate the list
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

		// step: validate the ingress resource has be processed by us
		if err := isValidIngress(&x); err != nil {
			logrus.WithFields(logrus.Fields{
				"name":      x.Name,
				"namespace": x.Namespace,
				"error":     err.Error(),
			}).Errorf("invalid ingress resource")
			continue
		}

		// step: we iterate the tls configs and check the certificate exist
		for index, j := range x.Spec.TLS {
			message, err := func() (string, error) {
				if found, err := c.hasSecret(j.SecretName, x.Namespace); err != nil {
					return "unable to check for kubernetes secret", err
				} else if found {
					// check if the certificate is coming up for renewal

					return "", nil
				}

				// step: we need to request a certificate from vault for this
				cert, err := c.requestCertificate(&x, j)
				if err != nil {
					return "unable to generate certificate for resource", err
				}
				// step: inject the certificate into the namespace
				if err := c.addSecret(j.SecretName, x.Namespace, cert); err != nil {
					return "unable to add kubernetes secret", err
				}
				return "", nil
			}()
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"name":      x.Name,
					"namespace": x.Namespace,
					"index":     index,
					"error":     err.Error(),
				}).Error(message)
			}
		}
	}
}
