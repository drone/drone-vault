// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

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
		logrus.Debugf("vault: token rereshing disabled")
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
