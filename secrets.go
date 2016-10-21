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
	"encoding/base64"
	"errors"
	"fmt"

	"k8s.io/kubernetes/pkg/api"
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

	secret, err := c.kc.Secrets(namespace).Get(name)
	if err != nil {
		return cert, err
	}

	if secret.Type != api.SecretTypeTLS {
		return cert, errors.New("invalid secret type, not tls")
	}

	for _, x := range []string{api.TLSCertKey, api.TLSPrivateKeyKey} {
		decoded, err := base64.StdEncoding.DecodeString(string(secret.Data[x]))
		if err != nil {
			return cert, fmt.Errorf("unable to base64 decode %s, reason: %s", x, err)
		}
		switch x {
		case api.TLSCertKey:
			cert.cert = string(decoded)
		default:
			cert.key = string(decoded)
		}
	}

	return cert, nil
}

// addSecret is responsible for add the certificate secret to the namespace
func (c *controller) addSecret(name, namespace string, cert certificate) error {
	// step: add the secret to the namespace
	_, err := c.kc.Secrets(namespace).Create(&api.Secret{
		Type: api.SecretTypeTLS,
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			api.TLSCertKey:       []byte(base64.StdEncoding.EncodeToString([]byte(cert.cert))),
			api.TLSPrivateKeyKey: []byte(base64.StdEncoding.EncodeToString([]byte(cert.key))),
		},
	})

	return err
}
