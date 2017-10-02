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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Author = author
	app.Usage = "Requests certificates from vault on behalf of ingress resources"
	app.Version = version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host, H",
			Usage:  "the url for the vault service i.e. https://vault.vault.svc.cluster.local `HOST`",
			EnvVar: "VAULT_ADDR",
		},
		cli.StringFlag{
			Name:   "token, t",
			Usage:  "the vault token to use when requesting a certificate `TOKEN`",
			EnvVar: "VAULT_TOKEN",
		},
		cli.StringFlag{
			Name:   "default-path, p",
			Usage:  "the default vault path the pki exists on, e.g. pki/default/issue `PATH`",
			EnvVar: "VAULT_PKI_PATH",
			Value:  "pki/issue/default",
		},
		cli.StringFlag{
			Name:   "kubeconfig",
			Usage:  "the path to a kubectl configuration file `PATH`",
			Value:  os.Getenv("HOME") + "/.kube/config",
			EnvVar: "KUBE_CONFIG",
		},
		cli.StringFlag{
			Name:   "kube-context",
			Usage:  "the kubernetes context inside the kubeconfig file `CONTEXT`",
			EnvVar: "KUBE_CONTEXT",
		},
		cli.StringFlag{
			Name:   "namespace, n",
			Usage:  "the namespace the service should be looking, by default all `NAMESPACE`",
			EnvVar: "KUBE_NAMESPACE",
		},
		cli.DurationFlag{
			Name:   "default-ttl",
			Usage:  "the default time-to-live of the certificate (can override by annontation) `TTL`",
			EnvVar: "VAULT_PKI_TTL",
			Value:  48 * time.Hour,
		},
		cli.DurationFlag{
			Name:   "minimum-ttl",
			Usage:  "the minimum time-to-live on the certificate, ingress cannot request less then this",
			EnvVar: "VAULT_PKI_MIN_TTL",
			Value:  24 * time.Hour,
		},
		cli.DurationFlag{
			Name:   "refresh-ttl",
			Usage:  "refresh certificates when time-to-live goes below this threshold",
			EnvVar: "VAULT_PKI_REFRESH_TTL",
			Value:  6 * time.Hour,
		},
		cli.BoolTFlag{
			Name:  "json-logging",
			Usage: "whether to enable default json logging format, defaults to true",
		},
		cli.DurationFlag{
			Name:   "reconcilation-interval",
			Usage:  "the interval between forced reconciliation events `TTL`",
			EnvVar: "RECONCILCATION_INTERVAL",
			Value:  5 * time.Minute,
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "switch on verbose logging",
		},
	}
	// the default action to perform
	app.Action = func(cx *cli.Context) error {
		config := &Config{
			vaultURL:       cx.String("host"),
			vaultToken:     cx.String("token"),
			defaultCertTTL: cx.Duration("default-ttl"),
			minCertTTL:     cx.Duration("minimum-ttl"),
			refreshCertTTL: cx.Duration("refresh-ttl"),
			defaultPath:    cx.String("default-path"),
			kubeconfig:     cx.String("kubeconfig"),
			kubecontext:    cx.String("kube-context"),
			jsonLogging:    cx.Bool("json-logging"),
			reconcileTTL:   cx.Duration("reconcilation-interval"),
			verbose:        cx.Bool("verbose"),
		}
		// step: create the controller and run
		service, err := newController(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[error] %s", err)
			os.Exit(1)
		}
		// start the service
		if err := service.run(); err != nil {
			fmt.Fprintf(os.Stderr, "[error] %s", err)
			os.Exit(1)
		}

		signalChannel := make(chan os.Signal)
		signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-signalChannel

		return nil
	}

	app.Run(os.Args)
}
