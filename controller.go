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
	"strings"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/vault/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

// controller is the handler for requests and renewals
type controller struct {
	// the config
	config *Config
	// the vault client
	vc *api.Client
	// kubernetes client
	kc *client.Client
	// the current list of ingress resources
	resources atomic.Value
}

// newController creates a new controller for you
func newController(config *Config) error {
	// step: create the vault client
	vc, err := createVaultClient(config.vaultURL, config.vaultToken)
	if err != nil {
		return fmt.Errorf("unable to create vault client, reason %s", err)
	}

	// step: create a kubernetes client
	kc, err := createKubeClient()
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client, reason %s", err)
	}

	// step: create a controller service
	ctr := &controller{
		vc:     vc,
		kc:     kc,
		config: config,
	}
	// step: start the reconcilation process
	ctr.reconcile()

	return nil
}

// reconcile is the main reconcilation method
func (c *controller) reconcile() {
	// step: create a ticker for reconcilation
	ticker := time.NewTicker(c.config.reconcileTTL)
	// step: create a ticker for the certificate renewal
	renewal := time.NewTicker(1 * time.Hour)
	// step: create a watch of the ingress resources
	watcherCh, _ := c.createIngressWatcher()

	for {
		select {
		// a time interval for checking config
		case <-ticker.C:
			c.reconcileIngress()
		// a ingress resource updated or created
		case <-watcherCh:
			c.reconcileIngress()
		case <-renewal.C:
		}
	}
}

// requestCertificate requests a certificate from vault
func (c *controller) requestCertificate(ing *extensions.Ingress, cfg extensions.IngressTLS) (certificate, error) {
	var cert certificate

	// step: make a request for the certificate from vault
	path := c.config.defaultPath
	if override, found := ing.GetAnnotations()[AnnotationVaultPath]; found {
		path = override
	}
	ttl := c.config.defaultCertTTL
	if override, found := ing.GetAnnotations()[AnnotationVaultTTL]; found {
		tm, err := time.ParseDuration(override)
		if err == nil {
			ttl = tm
		}
	}
	// step: add some logging
	logrus.WithFields(logrus.Fields{
		"name":      ing.Name,
		"namespace": ing.Namespace,
		"path":      path,
		"ttl":       ttl.String(),
		"hosts":     strings.Join(cfg.Hosts, ","),
		"secret":    cfg.SecretName,
	}).Error("generating certificate for ingress resource")

	// step: construct the request
	request := map[string]interface{}{
		"common_name": cfg.Hosts[0],
		"format":      "pem",
		"ttl":         ttl,
	}
	if len(cfg.Hosts) > 1 {
		request["alt_name"] = strings.Join(cfg.Hosts[1:], ",")
	}

	// step: make the request to vault for the certificate
	secret, err := c.vc.Logical().Write(path, request)
	if err != nil {
		return cert, err
	}
	// check we have some data
	if secret.Data == nil || len(secret.Data) <= 0 {
		return cert, fmt.Errorf("invalid request from vault request, warning: %s", secret.Warnings)
	}

	cert.ca = secret.Data["issuing_ca"].(string)
	cert.cert = secret.Data["certificate"].(string)
	cert.key = secret.Data["private_key"].(string)

	return cert, err
}

// isValidIngress is responsible for validating the ingress resource
func isValidIngress(ing *extensions.Ingress) error {
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

// createVaultClient is responsible for creating a vault client
func createVaultClient(host, token string) (*api.Client, error) {
	// step: create the vault client
	client, err := api.NewClient(&api.Config{Address: host})
	if err != nil {
		return nil, err
	}
	client.SetToken(token)

	return client, nil
}

// createKubeClient is responsible for creating a kubernetes client
func createKubeClient() (*client.Client, error) {
	// step: create the client
	kc, err := client.NewInCluster()
	if err != nil {
		return nil, err
	}
	// step: test by getting the version
	if _, err := kc.ServerVersion(); err != nil {
		return nil, err
	}

	return kc, nil
}
