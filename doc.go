/*
Copyright 2016 Rohith Jayawardene All rights reserved.
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
	"time"

	"k8s.io/kubernetes/pkg/apis/extensions"
)

var (
	release = "v0.0.1"
	gitsha  = "no gitsha provided"
	version = release + " (git+sha: " + gitsha + ")"
)

const (
	// AnnotationVaultPath is the annontation for pki path
	AnnotationVaultPath = "ingress.vault.io/path"
	// AnnotationVaultTLS indicates the ingress resource is vault enabled
	AnnotationVaultTLS = "ingress.vault.io/tls"
	// AnnotationVaultTTL is the TTL to use on the certificate
	AnnotationVaultTTL = "ingress.vault.io/ttl"
)

// Config is the configuration for the sevice
type Config struct {
	// the vault address
	vaultURL string
	// the vault token
	vaultToken string
	// the namespace we should operate
	namespace string
	// the default ttl
	defaultCertTTL time.Duration
	// the default path
	defaultPath string
	// is the interval between reconcilation
	reconcileTTL time.Duration
	// the verbosity
	verbose bool
}

// ingressResource is a wrapper for extra functionality
type ingressResource struct {
	resource *extensions.Ingress
}

// certificate is a wrapper for a requested cert
type certificate struct {
	ca   string
	cert string
	key  string
}
