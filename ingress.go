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
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	extensions_v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// createIngressWatcher is responsible for creating a watcher on ingress resources
func (c *controller) createIngressWatcher() (chan struct{}, chan struct{}) {
	// step: the reconcile period
	resyncPeriod := time.Duration(60 * time.Second)
	// step: create the update channel
	updateCh := make(chan struct{}, 10)
	// the stop channel
	stopCh := make(chan struct{}, 0)
	// step: create the handler for and update the channel on events
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			go func() { updateCh <- struct{}{} }()
		},
		DeleteFunc: func(obj interface{}) {
			go func() { updateCh <- struct{}{} }()
		},
		UpdateFunc: func(old, cur interface{}) {
			go func() { updateCh <- struct{}{} }()
		},
	}

	// step: create the kubernetes informer
	_, controller := cache.NewInformer(
		&cache.ListWatch{
			ListFunc:  ingressListFunc(c.kc, c.config.namespace),
			WatchFunc: ingressWatchFunc(c.kc, c.config.namespace),
		}, &extensions_v1beta1.Ingress{}, resyncPeriod, handler,
	)

	// step: start the controller
	go controller.Run(stopCh)

	return updateCh, stopCh
}

// ingressList returns a list of ingress resources
func (c *controller) ingressList() (*extensions_v1beta1.IngressList, error) {
	return c.kc.Extensions().Ingresses(c.config.namespace).List(meta_v1.ListOptions{})
}

// ingressListFunc is responsible for listing ingress resources
func ingressListFunc(c *kubernetes.Clientset, ns string) func(meta_v1.ListOptions) (runtime.Object, error) {
	return func(opts meta_v1.ListOptions) (runtime.Object, error) {
		return c.Extensions().Ingresses(ns).List(opts)
	}
}

// ingressWatchFunc is responsible for watching ingress resources
func ingressWatchFunc(c *kubernetes.Clientset, ns string) func(options meta_v1.ListOptions) (watch.Interface, error) {
	return func(options meta_v1.ListOptions) (watch.Interface, error) {
		return c.Extensions().Ingresses(ns).Watch(options)
	}
}
