// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

// 2020-01-17 Added approle support https://github.com/fortman

package approle

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

var noContext = context.Background()
var roleId = "c3dedbfe-eadd-56dc-6883-83cf898b3ecc"
var secretId = "4d8ce042-4684-7e8d-dbb3-389bb9a39f7f"
var renewToken = "s.zREhsyJT79kcuGbfrKsbyo0W"
var newToken = "s.OhZm4kQxf6K45Tg0bKNQbTJD"
var ttl, _ = time.ParseDuration("1200s")

func TestMain(m *testing.M) {
	log.SetLevel(log.WarnLevel)
	os.Exit(m.Run())
}

// Renew an existing token
func TestVaultApproleRenew(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/auth/token/renew" {
			t.Errorf("Invalid path, %v", req.URL.Path)
		}
		data, _ := ioutil.ReadFile("testdata/renew_token.json")
		w.Write(data)
	}
	config, listener := testHTTPServer(t, http.HandlerFunc(handler))
	defer listener.Close()

	tlsConfig := api.TLSConfig{Insecure: true}
	config.ConfigureTLS(&tlsConfig)

    client, vaultErr := api.NewClient(config)
    client.SetToken(renewToken)
    if vaultErr != nil {
    	t.Errorf("Can't connect to vault test server\n#{err}")
	}

	r := NewRenewer(client, roleId, secretId, ttl)
	err := r.Renew(noContext)

	if err != nil {
		t.Errorf("ERROR: %v", err)
	}
	if r.client.Token() != renewToken {
		t.Errorf("ERROR: expected token %v, got token %v", renewToken, r.client.Token())
	}

	t.Parallel()
}

// Test first time renew when a token doesn't exist yet
func TestVaultApproleRenewNoToken(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/auth/approle/login" {
			t.Errorf("Invalid path, %v", req.URL.Path)
		}
		data, _ := ioutil.ReadFile("testdata/new_token.json")
		w.Write(data)
	}
	config, listener := testHTTPServer(t, http.HandlerFunc(handler))
	defer listener.Close()

	tlsConfig := api.TLSConfig{Insecure: true}
	config.ConfigureTLS(&tlsConfig)

	client, vaultErr := api.NewClient(config)
	if vaultErr != nil {
		t.Errorf("Can't connect to vault test server\n#{err}")
	}

	r := NewRenewer(client, roleId, secretId, ttl)
	err := r.Renew(noContext)

	if err != nil {
		t.Errorf("ERROR: %v", err)
	}
	if r.client.Token() != newToken {
		t.Errorf("ERROR: expected token %v, got token %v", newToken, r.client.Token())
	}

	t.Parallel()
}

// Generate a new token
func TestVaultApproleNewToken(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/auth/approle/login" {
			t.Errorf("Invalid path, %v", req.URL.Path)
		}
		data, _ := ioutil.ReadFile("testdata/new_token.json")
		w.Write(data)
	}
	config, listener := testHTTPServer(t, http.HandlerFunc(handler))
	defer listener.Close()

	tlsConfig := api.TLSConfig{Insecure: true}
	config.ConfigureTLS(&tlsConfig)

	client, vaultErr := api.NewClient(config)
	if vaultErr != nil {
		t.Errorf("Can't connect to vault test server\n#{err}")
	}

	r := NewRenewer(client, roleId, secretId, ttl)
	err := r.NewToken(noContext)

	if err != nil {
		t.Errorf("ERROR: %v", err)
	}
	if r.client.Token() != newToken {
		t.Errorf("ERROR: expected token %v, got token %v", newToken, r.client.Token())
	}

	t.Parallel()
}

// Test logic when returned TTL is higher than requested
func TestVaultApproleRenewHigherTTL(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		var data []byte
		if req.URL.Path == "/v1/auth/token/renew" {
			data, _ = ioutil.ReadFile("testdata/renew_higher_ttl.json")
		} else if req.URL.Path == "/v1/auth/approle/login" {
			data, _ = ioutil.ReadFile("testdata/new_token.json")
		} else{
			t.Errorf("Invalid path, %v", req.URL.Path)
		}

		w.Write(data)
	}
	config, listener := testHTTPServer(t, http.HandlerFunc(handler))
	defer listener.Close()

	tlsConfig := api.TLSConfig{Insecure: true}
	config.ConfigureTLS(&tlsConfig)

	client, vaultErr := api.NewClient(config)
	client.SetToken(renewToken)
	if vaultErr != nil {
		t.Errorf("Can't connect to vault test server\n#{err}")
	}

	r := NewRenewer(client, roleId, secretId, ttl)
	err := r.Renew(noContext)

	if err != nil {
		t.Errorf("ERROR: %v", err)
	}
	if r.client.Token() != renewToken {
		t.Errorf("ERROR: expected token %v, got token %v", renewToken, r.client.Token())
	}

	t.Parallel()
}

// Test logic when returned TTL is lower than requested
// Expect a new token to be generated
func TestVaultApproleRenewLowerTTL(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		var data []byte
		if req.URL.Path == "/v1/auth/token/renew" {
			data, _ = ioutil.ReadFile("testdata/renew_lower_ttl.json")
		} else if req.URL.Path == "/v1/auth/approle/login" {
			data, _ = ioutil.ReadFile("testdata/new_token.json")
		} else{
			t.Errorf("Invalid path, %v", req.URL.Path)
		}

		w.Write(data)
	}
	config, listener := testHTTPServer(t, http.HandlerFunc(handler))
	defer listener.Close()

	tlsConfig := api.TLSConfig{Insecure: true}
	config.ConfigureTLS(&tlsConfig)

	client, vaultErr := api.NewClient(config)
	client.SetToken(renewToken)
	if vaultErr != nil {
		t.Errorf("Can't connect to vault test server\n#{err}")
	}

	r := NewRenewer(client, roleId, secretId, ttl)
	err := r.Renew(noContext)

	if err != nil {
		t.Errorf("ERROR: %v", err)
	}
	if r.client.Token() != newToken {
		t.Errorf("ERROR: expected token %v, got token %v", newToken, r.client.Token())
	}

	t.Parallel()
}

func testHTTPServer(
	t *testing.T, handler http.Handler) (*api.Config, net.Listener) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	server := &http.Server{Handler: handler}
	go server.Serve(ln)

	config := api.DefaultConfig()
	config.Address = fmt.Sprintf("http://%s", ln.Addr())

	return config, ln
}
