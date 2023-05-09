// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client, false)
	got, err := plugin.Find(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}

	want := &drone.Secret{
		Name: "username",
		Data: "david",
		Pull: true,
		Fork: true,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
		return
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
	plugin := New(client, false)
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
	plugin := New(client, false)
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
	plugin := New(client, false)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect error")
		return
	}
	if want, got := err.Error(), "access denied: event does not match"; got != want {
		t.Errorf("Want error %q, got %q", want, got)
	}
}

func TestPlugin_FilterForks(t *testing.T) {
	cases := []struct {
		name                       string
		disallowForksSecretSetting string
		disallowForksGlobalSetting bool
		expectedError              string
		isFork                     bool
	}{
		{
			name:                       "disallow forks, secret setting is true",
			disallowForksSecretSetting: "true",
			disallowForksGlobalSetting: false,
			expectedError:              "access denied: forks are not allowed",
			isFork:                     true,
		},
		{
			name:                       "disallow forks, global setting is true",
			disallowForksSecretSetting: "",
			disallowForksGlobalSetting: true,
			expectedError:              "access denied: forks are not allowed",
			isFork:                     true,
		},
		{
			name:                       "disallow forks, secret setting is false",
			disallowForksSecretSetting: "false",
			disallowForksGlobalSetting: false,
			expectedError:              "",
			isFork:                     true,
		},
		{
			name:                       "disallow forks, secret setting is not set",
			disallowForksSecretSetting: "",
			disallowForksGlobalSetting: false,
			expectedError:              "",
			isFork:                     true,
		},
		{
			name:                       "disallow forks, secret setting enabled but not a fork",
			disallowForksSecretSetting: "true",
			disallowForksGlobalSetting: false,
			expectedError:              "",
			isFork:                     false,
		},
		{
			name:                       "disallow forks, global setting enabled but not a fork",
			disallowForksSecretSetting: "",
			disallowForksGlobalSetting: true,
			expectedError:              "",
			isFork:                     false,
		},
		{
			name:                       "allow forks, secret setting is false (override global setting)",
			disallowForksSecretSetting: "false",
			disallowForksGlobalSetting: true,
			expectedError:              "",
			isFork:                     true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				payloadBefore, _ := ioutil.ReadFile("testdata/secret.json")
				payload := make(map[string]interface{})
				json.Unmarshal(payloadBefore, &payload)
				if c.disallowForksSecretSetting != "" {
					data := payload["data"].(map[string]interface{})
					data["X-Drone-Disallow-Forks"] = c.disallowForksSecretSetting
					payload["data"] = data
				}
				payloadAfter, _ := json.Marshal(payload)
				w.Write(payloadAfter)
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
					Slug: "octocat/hello-world",
				},
			}
			if c.isFork {
				req.Build.Fork = "spaceghost/hello-world"
			}

			plugin := New(client, c.disallowForksGlobalSetting)
			gotErr := ""
			if _, err := plugin.Find(noContext, req); err != nil {
				gotErr = err.Error()
			}

			if gotErr != c.expectedError {
				t.Errorf("Want error %q, got %q", c.expectedError, gotErr)
			}
		})
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
	plugin := New(client, false)
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
	plugin := New(client, false)
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

/* Test_rewritePath establishes that the behavior of our path rewrite logic exactly parallels that of the Vault CLI;
 generated with logic like (requires authentication to a vault namespace with both v1 and v2 engines mounted):

main() {
  for path in mount{/v2{,/data},/v1}{/bar,}{,/}; do
    jq --null-input \
      --arg mount_data "$(get_mount "${path}")" \
      --arg rewritten "$(get_rewritten_path "${path}")" \
      --arg path "${path}" \
      '{$mount_data, $rewritten, $path}'
  done |
    jq -s '[.[] | (.is_v2 = (.mount_data | fromjson).data.options.version == "2")]'
}

get_mount() {
  local path="${1}"
  curl \
    --silent \
    -H "X-Vault-Request: true" \
    -H "X-Vault-Token: $(vault print token)" \
    "https://vault.example.com/v1/sys/internal/ui/mounts/${path}"
}

get_rewritten_path() {
  local path="${1}"
  vault kv get -output-curl-string "${path}" | cut -d/ -f5-
}

main
*/
func Test_rewritePath(t *testing.T) {
	var testCases []struct {
		Path      string `json:"path"`
		MountData string `json:"mount_data"`
		Rewritten string `json:"rewritten"`
		IsV2      bool   `json:"is_v2"`
	}
	func() {
		f, err := os.Open("testdata/v2.json")
		if err != nil {
			t.Skipf("expected test data present at testdata/v2.json: %v", err)
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&testCases); err != nil {
			t.Fatalf("malformed test data: %v", err)
		}
	}()
	for _, tc := range testCases {
		t.Run(strings.ReplaceAll(tc.Path, "/", "_"), func(t *testing.T) {
			isV2, rewrite, err := rewritePath(strings.NewReader(tc.MountData), tc.Path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rewrite != tc.Rewritten {
				t.Errorf("expected %q but got %q", tc.Rewritten, rewrite)
			}
			if isV2 != tc.IsV2 {
				t.Errorf("expected %v but got %v", tc.IsV2, isV2)
			}
		})
	}
}
