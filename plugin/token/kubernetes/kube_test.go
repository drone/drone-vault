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
	"time"

	"github.com/drone/drone-vault/plugin/token"
	"github.com/google/go-cmp/cmp"
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

	got, err := load(ts.URL, "dev-role", "kubernetes", "testdata/token.jwt")
	if err != nil {
		t.Error(err)
	}

	want := &token.Token{
		Token: "62b858f9-529c-6b26-e0b8-0457b6aacdb4",
		TTL:   time.Duration(2764800000000000),
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestLoad_FileError(t *testing.T) {
	_, err := load("http://localhost", "dev-role", "kubernetes", "testdata/does-not-exist.jwt")
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

	_, err := load(ts.URL, "dev-role", "kubernetes", "testdata/token.jwt")
	if err == nil {
		t.Errorf("Expect request error")
	}
}
