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
	"fmt"

	"github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

// hasSecret checks if the secret exists for the namespace
func (c *controller) hasSecret(name, namespace string) (bool, error) {
	// step: get a list of all secrets
	list, err := c.kc.Secrets(namespace).List(api.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, x := range list.Items {
		if x.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// getSecret is responsible for retrieving a certificate from the namespace
func (c *controller) getSecret(name, namespace string) (certificate, error) {
	var cert certificate

	logrus.WithFields(logrus.Fields{
		"name":      name,
		"namespace": namespace,
	}).Debug("retrieving the certificate secret from kubernetes")

	secret, err := c.kc.Secrets(namespace).Get(name)
	if err != nil {
		return cert, err
	}

	if secret.Type != api.SecretTypeTLS {
		return cert, fmt.Errorf("invalid secret type, expect: %s but got: %s", api.SecretTypeTLS, secret.Type)
	}
	cert.cert = []byte(secret.Data[api.TLSCertKey])
	cert.key = []byte(secret.Data[api.TLSPrivateKeyKey])

	return cert, nil
}

// addSecret is responsible for add the certificate secret to the namespace
func (c *controller) addSecret(name, namespace string, cert certificate) error {
	var err error
	// step: create the secret
	secret := &api.Secret{
		Type: api.SecretTypeTLS,
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			api.TLSCertKey:       cert.cert,
			api.TLSPrivateKeyKey: cert.key,
		},
	}

	// step: add or update the secret
	found, err := c.hasSecret(name, namespace)
	if err != nil {
		return err
	}
	switch found {
	case false:
		_, err = c.kc.Secrets(namespace).Create(secret)
	default:
		_, err = c.kc.Secrets(namespace).Update(secret)
	}

	return err
}

// createKubeClient is responsible for creating a kubernetes client
func createKubeClient(path, context string) (*client.Client, error) {
	var kc *client.Client

	if path != "" && context != "" {
		// step: load the confiuration file
		kube, err := clientcmd.LoadFromFile(path)
		if err != nil {
			return nil, err
		}
		// step: load the configuration
		config, err := clientcmd.NewDefaultClientConfig(*kube, &clientcmd.ConfigOverrides{CurrentContext: context}).ClientConfig()
		if err != nil {
			return nil, err
		}
		kc, err = client.New(config)
		if err != nil {
			return nil, err
		}
	} else {
		// step: create the client
		ci, err := client.NewInCluster()
		if err != nil {
			return nil, err
		}
		kc = ci
	}
	// step: test by getting the version
	if _, err := kc.ServerVersion(); err != nil {
		return nil, err
	}

	return kc, nil
}
