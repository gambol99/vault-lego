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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
)

const (
	pkiCommonName  = "common_name"
	pkiFormat      = "format"
	pkiTTL         = "ttl"
	pkiHosts       = "alt_names"
	pkiCertificate = "certificate"
	pkiPrivateKey  = "private_key"
	pkiIssuer      = "issuing_ca"
)

// generateCertificate requests a certificate from vault
func (c *controller) generateCertificate(path string, ttl time.Duration, hosts []string) (certificate, error) {
	var cert certificate
	// step: make the request to vault for the certificate
	secret, err := c.vc.Logical().Write(path, getVaultCertificateRequest(hosts, ttl))
	if err != nil {
		return cert, err
	}

	// check we have some data
	if secret.Data == nil || len(secret.Data) <= 0 {
		return cert, fmt.Errorf("invalid request from vault request, warning: %s", secret.Warnings)
	}

	cert.ca = []byte(secret.Data["issuing_ca"].(string))
	cert.cert = []byte(secret.Data["certificate"].(string))
	cert.key = []byte(secret.Data["private_key"].(string))

	return cert, err
}

// isCertificateExpiring parses and checks if the certificate is about to expire
func isCertificateExpiring(content []byte, threshold time.Duration) (bool, error) {
	// decode the pem content
	block, _ := pem.Decode(content)
	if block == nil {
		return false, errors.New("unable to parse the pem block")
	}
	// decode the certificate
	crt, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, err
	}
	// are we after the expiration date?
	if time.Now().After(crt.NotAfter) {
		return true, nil
	}
	if time.Now().After(crt.NotAfter.Add(threshold)) {
		return true, nil
	}

	return false, nil
}

// createVaultClient is responsible for creating a vault client
func createVaultClient(host, token string) (*api.Client, error) {
	// the http client
	hc := &http.Client{
		Timeout: time.Duration(30) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// step: create the vault client
	client, err := api.NewClient(&api.Config{Address: host, HttpClient: hc})
	if err != nil {
		return nil, err
	}
	// step: set the client token
	client.SetToken(token)

	return client, nil
}

// getVaultCertificateRequest is responsible for builting a request
func getVaultCertificateRequest(hosts []string, ttl time.Duration) map[string]interface{} {
	request := map[string]interface{}{
		pkiCommonName: hosts[0],
		pkiTTL:        fmt.Sprintf("%fh", ttl.Hours()),
	}
	if len(hosts) > 1 {
		request[pkiHosts] = strings.Join(hosts[1:], ",")
	}

	return request
}
