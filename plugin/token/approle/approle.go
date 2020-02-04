// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

// 2020-01-17 Added approle support https://github.com/fortman

package approle

import (
	"context"
	"strconv"
	"time"
	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

// Name that identifies the auth method.
const Name = "approle"

type (
	// Renewer renews the Kubernetes token.
	Renewer struct {
		client *api.Client
		roleId   string
		secretId string
		ttl      int
	}
)

// NewRenewer returns a new Kubernetes token provider
// that renews the token on expiration.
func NewRenewer(client *api.Client, roleId string, secretId string, ttl time.Duration) *Renewer {
	return &Renewer{
		client:   client,
		roleId:   roleId,
		secretId: secretId,
		ttl: int(ttl.Seconds()),
	}
}

// Renew renews the Vault token.
func (r *Renewer) Renew(ctx context.Context) error {
	// create the vault endpoint address.
	path := "auth/token/renew-self"


	if r.client.Token() == "" {
		logrus.Infoln("vault approle: no existing token, fetching one")
		return r.NewToken(ctx)
	}

	logrus.Debugln("vault approle: renewing token")

	// Renew

	resp, err := r.client.Logical().Write(path,
		map[string]interface{}{
			"increment": strconv.Itoa(r.ttl),
		})
	if err != nil {
		logrus.WithError(err).Errorln("vault approle: token could not be renewed")
		return r.NewToken(ctx)
	}

	if resp.Auth.LeaseDuration < r.ttl {
		logrus.Infoln("vault approle: token could not be renewed for desired ttl")
		logrus.Infoln("vault approle: will request new token")
		return r.NewToken(ctx)
	}

	if resp == nil {
		logrus.Errorln("expected a response for login")
		return nil
	}
	if resp.Auth == nil {
		logrus.Errorln("expected auth object from response")
		return nil
	}
	if resp.Auth.ClientToken == "" {
		logrus.Errorln("expected a client token")
		return nil
	}

	r.client.SetToken(resp.Auth.ClientToken)
	ttl := time.Duration(resp.Auth.LeaseDuration) * time.Second

	logrus.WithField("ttl", ttl).
	 	Debugln("vault approle: existing token valid")

	return nil
}

func (r *Renewer) NewToken(ctx context.Context) error {
	path := "auth/approle/login"

	logrus.Debugln("vault approle: generating new token")

	resp, err := r.client.Logical().Write(path,
		map[string]interface{}{
			"role_id":   r.roleId,
			"secret_id": r.secretId,
		})
	if err != nil {
		logrus.Fatalln(err)
	}

	if resp == nil {
		logrus.Errorln("expected a response for login")
		return nil
	}
	if resp.Auth == nil {
		logrus.Errorln("expected auth object from response")
		return nil
	}
	if resp.Auth.ClientToken == "" {
		logrus.Errorln("expected a client token")
		return nil
	}

	r.client.SetToken(resp.Auth.ClientToken)
	ttl := time.Duration(resp.Auth.LeaseDuration) * time.Second
	logrus.WithField("ttl", ttl).
		Debugln("vault approle: token received")

	return nil
}

// Run performs token renewal at scheduled intervals.
func (r *Renewer) Run(ctx context.Context, renew time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(renew):
			r.Renew(ctx)
		}
	}
}
