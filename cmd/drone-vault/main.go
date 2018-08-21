// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/drone/drone-go/plugin/secret"
	"github.com/drone/drone-vault/plugin"
	"github.com/drone/drone-vault/plugin/token"
	"github.com/drone/drone-vault/plugin/token/kubernetes"

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
	Debug          bool          `envconfig:"DEBUG"`
	Address        string        `envconfig:"SERVER_ADDRESS"`
	Secret         string        `envconfig:"SECRET_KEY"`
	VaultAddr      string        `envconfig:"VAULT_ADDR"`
	VaultRenew     time.Duration `envconfig:"VAULT_TOKEN_RENEWAL"`
	VaultTTL       time.Duration `envconfig:"VAULT_TOKEN_TTL"`
	VaultAuthType  string        `envconfig:"VAULT_AUTH_TYPE"`
	VaultAuthMount string        `envconfig:"VAULT_AUTH_MOUNT_POINT"`
	VaultKubeRole  string        `envconfig:"VAULT_KUBERNETES_ROLE"`
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

	// the token can be fetched at runtime if an auth
	// provider is configured. otherwise, the user must
	// specify a VAULT_TOKEN.
	if spec.VaultAuthType == kubernetes.Name {
		token, err := kubernetes.Load(
			spec.VaultAddr,
			spec.VaultKubeRole,
			spec.VaultAuthMount,
		)
		if err != nil {
			logrus.Fatalln(err)
		}
		client.SetToken(token.Token)
		spec.VaultTTL = token.TTL
	}

	http.Handle("/", secret.Handler(
		spec.Secret,
		plugin.New(client),
		logrus.StandardLogger(),
	))

	var g errgroup.Group
	g.Go(func() error {
		ctx := context.Background()
		return token.NewRenewer(
			client, spec.VaultTTL, spec.VaultRenew).Run(ctx)
	})

	g.Go(func() error {
		logrus.Infof("server listening on address %s", spec.Address)
		return http.ListenAndServe(spec.Address, nil)
	})

	if err := g.Wait(); err != nil {
		logrus.Fatal(err)
	}
}
