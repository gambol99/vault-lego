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
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kc_api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// hasSecret checks if the secret exists for the namespace
func (c *controller) hasSecret(name, namespace string) (bool, error) {
	// step: get a list of all secrets
	list, err := c.kc.Secrets(namespace).List(meta_v1.ListOptions{})
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

	secret, err := c.kc.Secrets(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return cert, err
	}

	if secret.Type != kc_api.SecretTypeTLS {
		return cert, fmt.Errorf("invalid secret type, expect: %s but got: %s", kc_api.SecretTypeTLS, secret.Type)
	}
	cert.cert = []byte(secret.Data[kc_api.TLSCertKey])
	cert.key = []byte(secret.Data[kc_api.TLSPrivateKeyKey])

	return cert, nil
}

// addSecret is responsible for add the certificate secret to the namespace
func (c *controller) addSecret(name, namespace string, cert certificate) error {
	var err error

	bundle := fmt.Sprintf("%s\n%s", string(cert.cert), string(cert.ca))

	// step: create the secret
	secret := &kc_api.Secret{
		Type: kc_api.SecretTypeTLS,
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			kc_api.TLSCertKey:       []byte(bundle),
			kc_api.TLSPrivateKeyKey: cert.key,
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

// createKubeClient is responsible for creating a kubernetes clientset
func createKubeClient(path, context string) (*kubernetes.Clientset, error) {
	config, err := createKubeConfig(path, context)
	if err != nil {
		return nil, err
	}

	// creates the clientset
	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// step: test by getting the version
	ksv, err := kc.ServerVersion()
	if err != nil {
		return nil, err
	}
	logrus.WithFields(logrus.Fields{
		"version": ksv.Major + "." + ksv.Minor,
	}).Debug("successfully connected to the api server")

	return kc, nil
}

func createKubeConfig(path, context string) (*rest.Config, error) {
	if path != "" && context != "" {
		// step: load the confiuration file
		kube, err := clientcmd.LoadFromFile(path)
		if err != nil {
			return nil, err
		}
		// step: load the configuration
		config, err := clientcmd.NewDefaultClientConfig(*kube,
			&clientcmd.ConfigOverrides{CurrentContext: context}).ClientConfig()
		if err != nil {
			return nil, err
		}
		return config, nil
	}

	// step: create the client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}
