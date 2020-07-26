// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package kubernetes

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

// Name that identifies the auth method.
const Name = "kubernetes"

// kubernetes token file path.
const defaultPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

type (
	// kubernetes authorization provider request.
	request struct {
		Jwt  string `json:"jwt"`
		Role string `json:"role"`
	}

	// kubernetes authorization provider response.
	response struct {
		Auth struct {
			Token string `json:"client_token"`
			Lease int    `json:"lease_duration"`
		}
	}

	// Renewer renews the Kubernetes token.
	Renewer struct {
		client *api.Client

		address string
		mount   string
		path    string
		role    string
	}
)

// NewRenewer returns a new Kubernetes token provider
// that renews the token on expiration.
func NewRenewer(client *api.Client, address, role, mount string) *Renewer {
	return &Renewer{
		address: address,
		client:  client,
		mount:   mount,
		role:    role,
		path:    defaultPath,
	}
}

// Renew renews the Vault token.
func (r *Renewer) Renew(ctx context.Context) error {
	// create the vault endpoint address.
	endpoint := fmt.Sprintf("%s/v1/auth/%s/login", r.address, r.mount)

	logrus.WithField("path", r.path).
		Debugln("kubernetes: reading account token")

	// reads the jwt token mounted inside the container.
	b, err := ioutil.ReadFile(r.path)
	if err != nil {
		logrus.WithError(err).
			WithField("path", r.path).
			Errorln("kubernetes: cannot read account token")
		return err
	}

	res := &response{}
	req := &request{
		Jwt:  string(b),
		Role: r.role,
	}

	logrus.WithField("endpoint", endpoint).
		Debugln("kubernetes: requesting vault token")

	err = post(endpoint, req, res)
	if err != nil {
		logrus.WithError(err).
			WithField("endpoint", endpoint).
			Errorln("kubernetes: cannot request vault token")
		return err
	}

	r.client.SetToken(res.Auth.Token)
	ttl := time.Duration(res.Auth.Lease) * time.Second

	logrus.WithField("ttl", ttl).
		Debugln("kubernetes: token received")

	return nil
}

// Run performs token renewal at scheduled intervals.
func (r *Renewer) Run(ctx context.Context, renew time.Duration) error {
	if renew == 0 {
		renew = time.Hour
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(renew):
			r.Renew(ctx)
		}
	}
}
