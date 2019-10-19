// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package token

import (
	"context"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

// Renewer implements token renewal.
type Renewer struct {
	client *api.Client
	ttl    time.Duration
	renew  time.Duration
}

// NewRenewer returns a new token renewer.
func NewRenewer(client *api.Client, ttl, renew time.Duration) *Renewer {
	return &Renewer{
		client: client,
		ttl:    ttl,
		renew:  renew,
	}
}

// Run performs token renewal at scheduled intervals.
func (r *Renewer) Run(ctx context.Context) error {
	if r.renew == 0 || r.ttl == 0 {
		logrus.Debugf("vault: token refreshing disabled")
		return nil
	}

	logrus.Infof("vault: token renewal enabled: %v interval", r.renew)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.renew):
			r.refresh()
		}
	}
}

func (r *Renewer) refresh() error {
	incr := int(r.ttl / time.Second)

	logrus.Debugf("vault: refreshing token: increment %v", r.ttl)
	_, err := r.client.Auth().Token().RenewSelf(incr)
	if err != nil {
		logrus.Errorf("vault: refreshing token failed: %s", err)
		return err
	}
	logrus.Debugf("vault: refreshing token succeeded")
	return nil
}
