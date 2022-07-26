// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"encoding/json"
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/secret"
	"github.com/hashicorp/vault/api"

	"github.com/google/go-cmp/cmp"
)

var noContext = context.Background()

// Use the following snippet to spin up a local vault
// server for integration testing:
//
//    docker run --cap-add=IPC_LOCK -e 'VAULT_DEV_ROOT_TOKEN_ID=dummy' -p 8200:8200 vault
//    export VAULT_ADDR=http://127.0.0.1:8200
//    export VAULT_TOKEN=dummy

func TestPlugin(t *testing.T) {
	fileSecret, _:= ioutil.ReadFile("testdata/secret.json") 
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fileSecret)
	}))
	defer ts.Close()

	client, _ := api.NewClient(&api.Config{
		Address:    ts.URL,
		MaxRetries: 1,
	})

	plugin := New(client)

	// convert testdata/secret.json into mock return value
  // used for asterisk selector test.
	var jsonSecret map[string]map[string]interface{}
	json.Unmarshal(fileSecret, &jsonSecret)
	jsonSecretDataArr, _ := json.Marshal(filterStringData(jsonSecret["data"]))
	jsonSecretData := bytes.NewBuffer(jsonSecretDataArr).String()

	var tests = []struct {
		Request secret.Request
		Want    drone.Secret
	}{
		{
			secret.Request{
				Path: "secret/docker",
				Name: "username",
				Build: drone.Build{
					Event:  "push",
					Target: "master",
				},
				Repo: drone.Repo{
					Slug: "octocat/hello-world",
				},
			},
			drone.Secret{
				Name: "username",
				Data: "david",
				Pull: true,
				Fork: true,
			},
		},
		{
			secret.Request{
				Path: "secret/docker",
				Name: "password",
				Build: drone.Build{
					Event:  "push",
					Target: "master",
				},
				Repo: drone.Repo{
					Slug: "octocat/hello-world",
				},
			},
			drone.Secret{
				Name: "password",
				Data: "BnQw&XDWgaEeT9XGTT29",
				Pull: true,
				Fork: true,
			},
		},
		{
			secret.Request{
				Path: "secret/docker",
				Name: "*",
				Build: drone.Build{
					Event:  "push",
					Target: "master",
				},
				Repo: drone.Repo{
					Slug: "octocat/hello-world",
				},
			},
			drone.Secret{
				Name: "*",
				Data: jsonSecretData,
				Pull: true,
				Fork: true,
			},
		},
	}

	for _, tc := range tests {
		got, err := plugin.Find (noContext, &tc.Request)
		if err != nil {
			t.Error(err)
			return
		}

		if diff := cmp.Diff(got, &tc.Want); diff != "" {
			t.Errorf(diff)
			return
		}
	}
}

func TestPlugin_FilterBranches(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := ioutil.ReadFile("testdata/secret.json")
		w.Write(out)
	}))
	defer ts.Close()

	client, _ := api.NewClient(&api.Config{
		Address:    ts.URL,
		MaxRetries: 1,
	})

	req := &secret.Request{
		Path: "secret/docker",
		Name: "username",
		Build: drone.Build{
			Event:  "push",
			Target: "development",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect error")
		return
	}
	if want, got := err.Error(), "access denied: branch does not match"; got != want {
		t.Errorf("Want error %q, got %q", want, got)
	}
}

func TestPlugin_FilterRepo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := ioutil.ReadFile("testdata/secret.json")
		w.Write(out)
	}))
	defer ts.Close()

	client, _ := api.NewClient(&api.Config{
		Address:    ts.URL,
		MaxRetries: 1,
	})

	req := &secret.Request{
		Path: "secret/docker",
		Name: "username",
		Build: drone.Build{
			Event:  "push",
			Target: "master",
		},
		Repo: drone.Repo{
			Slug: "spaceghost/hello-world",
		},
	}
	plugin := New(client)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect error")
		return
	}
	if want, got := err.Error(), "access denied: repository does not match"; got != want {
		t.Errorf("Want error %q, got %q", want, got)
	}
}

func TestPlugin_FilterEvent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := ioutil.ReadFile("testdata/secret.json")
		w.Write(out)
	}))
	defer ts.Close()

	client, _ := api.NewClient(&api.Config{
		Address:    ts.URL,
		MaxRetries: 1,
	})

	req := &secret.Request{
		Path: "secret/docker",
		Name: "username",
		Build: drone.Build{
			Event:  "pull_request",
			Target: "master",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect error")
		return
	}
	if want, got := err.Error(), "access denied: event does not match"; got != want {
		t.Errorf("Want error %q, got %q", want, got)
	}
}

func TestPlugin_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := ioutil.ReadFile("testdata/not_found.json")
		w.WriteHeader(404)
		w.Write(out)
	}))
	defer ts.Close()

	client, _ := api.NewClient(&api.Config{
		Address:    ts.URL,
		MaxRetries: 1,
	})

	req := &secret.Request{
		Path: "secret/docker",
		Name: "username",
		Build: drone.Build{
			Event:  "pull_request",
			Target: "master",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect error")
		return
	}
	if want, got := err.Error(), "secret not found"; got != want {
		t.Errorf("Want error %q, got %q", want, got)
		return
	}
}

func TestPlugin_KeyNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := ioutil.ReadFile("testdata/secret.json")
		w.WriteHeader(200)
		w.Write(out)
	}))
	defer ts.Close()

	client, _ := api.NewClient(&api.Config{
		Address:    ts.URL,
		MaxRetries: 1,
	})

	req := &secret.Request{
		Path: "secret/docker",
		Name: "token",
		Build: drone.Build{
			Event:  "push",
			Target: "master",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect error")
		return
	}
	if got, want := err.Error(), "secret key not found"; got != want {
		t.Errorf("Want error %q, got %q", want, got)
		return
	}
}
