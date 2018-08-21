// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package token

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

func TestRenew(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := ioutil.ReadFile("testdata/renew.json")
		w.Write(out)
	}))
	defer ts.Close()

	client, _ := api.NewClient(&api.Config{
		Address:    ts.URL,
		MaxRetries: 1,
	})
	client.SetToken("8609694a-cdbc-db9b-d345-e782dbb562ed")

	r := NewRenewer(client, time.Minute, time.Minute)
	err := r.refresh()
	if err != nil {
		t.Error(err)
	}
}

func TestRenewErr(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer ts.Close()

	client, _ := api.NewClient(&api.Config{
		Address:    ts.URL,
		MaxRetries: 1,
	})

	r := NewRenewer(client, time.Minute, time.Minute)
	err := r.refresh()
	if err == nil {
		t.Errorf("Want error refreshing token")
	}
}
