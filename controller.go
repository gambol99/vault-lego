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
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/vault/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/flowcontrol"
)

// controller is the handler for requests and renewals
type controller struct {
	// the config
	config *Config
	// the vault client
	vc *api.Client
	// kubernetes client
	kc *kubernetes.Clientset
	// the current list of ingress resources
	resources atomic.Value
	// the reconcile rate limiter
	rate flowcontrol.RateLimiter
}

// newController creates a new controller for you
func newController(config *Config) (*controller, error) {
	// step: configure the logger
	logrus.SetLevel(logrus.InfoLevel)
	if config.jsonLogging {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
	if config.verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Infof("starting the vault-lego service, version: %s", version)

	// step: check the service configuration
	if err := isValidConfig(config); err != nil {
		return nil, fmt.Errorf("configuration invalid, %s", err)
	}

	// step: create the vault client
	vc, err := createVaultClient(config.vaultURL, config.vaultToken)
	if err != nil {
		return nil, fmt.Errorf("unable to create vault client, reason %s", err)
	}

	// step: create a kubernetes client
	kc, err := createKubeClient(config.kubeconfig, config.kubecontext)
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client, reason %s", err)
	}

	return &controller{
		vc:     vc,
		kc:     kc,
		config: config,
	}, nil
}

// isValidConfig checks the configurtion options are vali
func isValidConfig(config *Config) error {
	if config.vaultURL == "" {
		return errors.New("no vault host")
	}
	if config.vaultToken == "" {
		return errors.New("no vault token")
	}
	if config.minCertTTL > config.defaultCertTTL {
		return errors.New("minimum certificate ttl cannot be greater then default")
	}

	return nil
}

// reconcile is the main reconcilation method
func (c *controller) reconcile() error {
	// step: create a ticker for reconcilation
	ticker := time.NewTicker(c.config.reconcileTTL)

	// step: create a watch of the ingress resources
	watcherCh, _ := c.createIngressWatcher()

	// step: setup the ratelimiter for ingress
	c.rate = flowcontrol.NewTokenBucketRateLimiter(0.1, 1)

	for {
		select {
		// a time interval for checking config
		case <-ticker.C:
			logrus.Debugf("reconcilation ticker has fired")
			go c.reconcileIngress()
		// a ingress resource updated or created
		case <-watcherCh:
			logrus.Debugf("a change to ingress has occured")
			go c.reconcileIngress()
		}
	}
}

// run is responsible for starting the reconcilation loop
func (c *controller) run() error {
	// step: get an initial list of resurces
	resources, err := c.ingressList()
	if err != nil {
		return err
	}
	c.resources.Store(resources)
	// step: start the background processing
	go c.reconcile()

	return nil
}
