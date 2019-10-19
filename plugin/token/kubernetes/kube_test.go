// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package kubernetes

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/vault/api"
)

var noContext = context.Background()

func TestLoad(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/kubernetes/login" {
			t.Errorf("Invalid path")
		}
		data, _ := ioutil.ReadFile("testdata/token.json")
		w.Write(data)
	}))
	defer ts.Close()

	client, _ := api.NewClient(nil)

	r := NewRenewer(client, ts.URL, "dev-role", "kubernetes")
	r.path = "testdata/token.jwt"
	err := r.Renew(noContext)
	if err != nil {
		t.Error(err)
	}

	want := "62b858f9-529c-6b26-e0b8-0457b6aacdb4"
	got := client.Token()
	if got != want {
		t.Errorf("Want token %s, got %s", want, got)
	}
}

func TestLoad_FileError(t *testing.T) {
	r := NewRenewer(nil, "http://localhost", "dev-role", "kubernetes")
	r.path = "testdata/does-not-exist.jwt"
	err := r.Renew(noContext)
	if _, ok := err.(*os.PathError); !ok {
		t.Errorf("Expect PathError got %v", err)
	}
}

func TestLoad_RequestError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/kubernetes/login" {
			t.Errorf("Invalid path")
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	r := NewRenewer(nil, ts.URL, "dev-role", "kubernetes")
	r.path = "testdata/token.jwt"
	err := r.Renew(noContext)
	if err == nil {
		t.Errorf("Expect request error")
	}
}
