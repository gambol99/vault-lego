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
	"time"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Author = "Rohith Jayawardene"
	app.Usage = "Requests certificates from vault on behalf of ingress resources"
	app.Version = version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host, h",
			Usage:  "the url for the vault service i.e. https://vault.vault.svc.cluster.local `HOST`",
			EnvVar: "VAULT_ADDR",
		},
		cli.StringFlag{
			Name:   "token, t",
			Usage:  "the vault token to use when requesting a certificate `TOKEN`",
			EnvVar: "VAULT_TOKEN",
		},
		cli.StringFlag{
			Name:   "namespace, n",
			Usage:  "the namespace the service should be looking, by default all",
			EnvVar: "NAMESPACE",
			Value:  "",
		},
		cli.DurationFlag{
			Name:   "default-certificate-ttl",
			Usage:  "the default time-to-live of the certificate (can override by annontation) `TTL`",
			EnvVar: "DEFAULT_CERTIFICATE_TTL",
			Value:  48 * time.Hour,
		},
		cli.StringFlag{
			Name:   "default-path, p",
			Usage:  "the default vault path the pki exists on, e.g. pki/default/issue `PATH`",
			EnvVar: "VAULT_PKI_PATH",
			Value:  "pki/issue/default",
		},
		cli.DurationFlag{
			Name:   "reconcilation-interval",
			Usage:  "the interval between forced reconciliation events `TTL`",
			EnvVar: "RECONCILCATION_INTERVAL",
			Value:  5 * time.Minute,
		},
	}
	// the default action to perform
	app.Action = func(cx *cli.Context) error {
		config := &Config{
			vaultURL:       cx.String("host"),
			vaultToken:     cx.String("token"),
			defaultCertTTL: cx.Duration("default-certificate-ttl"),
			defaultPath:    cx.String("default-path"),
			reconcileTTL:   cx.Duration("reconcilation-interval"),
			verbose:        cx.Bool("verbose"),
		}
		// step: create the controller and run
		if err := newController(config); err != nil {
			fmt.Fprintf(os.Stderr, "[error] %s", err)
			os.Exit(1)
		}
		return nil
	}

	app.Run(os.Args)
}
