// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

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
