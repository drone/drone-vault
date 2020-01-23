// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/drone/drone-go/plugin/secret"
	"github.com/drone/drone-vault/plugin"
	"github.com/drone/drone-vault/plugin/token"
	"github.com/drone/drone-vault/plugin/token/kubernetes"
	"github.com/drone/drone-vault/plugin/token/approle"

	"github.com/hashicorp/vault/api"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	_ "github.com/joho/godotenv/autoload"
)

// additional vault environment variables that are
// used by the vault client.
var envs = []string{
	"VAULT_ADDR",
	"VAULT_CACERT",
	"VAULT_CAPATH",
	"VAULT_CLIENT_CERT",
	"VAULT_SKIP_VERIFY",
	"VAULT_MAX_RETRIES",
	"VAULT_TOKEN",
	"VAULT_TLS_SERVER_NAME",
}

type config struct {
	Address            string        `envconfig:"DRONE_BIND"`
	Debug              bool          `envconfig:"DRONE_DEBUG"`
	Secret             string        `envconfig:"DRONE_SECRET"`
	VaultAddr          string        `envconfig:"VAULT_ADDR"`
	VaultRenew         time.Duration `envconfig:"VAULT_TOKEN_RENEWAL"`
	VaultTTL           time.Duration `envconfig:"VAULT_TOKEN_TTL"`
	VaultAuthType      string        `envconfig:"VAULT_AUTH_TYPE"`
	VaultAuthMount     string        `envconfig:"VAULT_AUTH_MOUNT_POINT"`
	VaultApproleID     string        `envconfig:"VAULT_APPROLE_ID"`
	VaultApproleSecret string        `envconfig:"VAULT_APPROLE_SECRET"`
	VaultKubeRole      string        `envconfig:"VAULT_KUBERNETES_ROLE"`
}

func main() {
	spec := new(config)
	err := envconfig.Process("", spec)
	if err != nil {
		logrus.Fatal(err)
	}

	if spec.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if spec.Secret == "" {
		logrus.Fatalln("missing secret key")
	}
	if spec.VaultAddr == "" {
		logrus.Warnln("missing vault address")
	}
	if spec.Address == "" {
		spec.Address = ":3000"
	}

	// creates the vault client from the VAULT_*
	// environment variables.
	client, err := api.NewClient(nil)
	if err != nil {
		logrus.Fatalln(err)
	}

	// global context
	ctx := context.Background()

	http.Handle("/", secret.Handler(
		spec.Secret,
		plugin.New(client),
		logrus.StandardLogger(),
	))

	var g errgroup.Group

	// the token can be fetched at runtime if an auth
	// provider is configured. otherwise, the user must
	// specify a VAULT_TOKEN.
	if spec.VaultAuthType == kubernetes.Name {
		renewer := kubernetes.NewRenewer(
			client,
			spec.VaultAddr,
			spec.VaultKubeRole,
			spec.VaultAuthMount,
		)
		err := renewer.Renew(ctx)
		if err != nil {
			logrus.Fatalln(err)
		}

		// the vault token needs to be periodically
		// refreshed and the kubernetes token has a
		// max age of 32 days.
		g.Go(func() error {
			return renewer.Run(ctx, spec.VaultRenew)
		})
	} else if spec.VaultAuthType == approle.Name {
		renewer := approle.NewRenewer(
			client,
			spec.VaultApproleID,
			spec.VaultApproleSecret,
			spec.VaultTTL,
		)
		err := renewer.Renew(ctx)
		if err != nil {
			logrus.Fatalln(err)
		}

		// the vault token needs to be periodically refreshed
		g.Go(func() error {
			return renewer.Run(ctx, spec.VaultRenew)
		})
	} else {
		g.Go(func() error {
			return token.NewRenewer(
				client, spec.VaultTTL, spec.VaultRenew).Run(ctx)
		})
	}

	g.Go(func() error {
		logrus.Infof("server listening on address %s", spec.Address)
		return http.ListenAndServe(spec.Address, nil)
	})

	if err := g.Wait(); err != nil {
		logrus.Fatal(err)
	}
}
