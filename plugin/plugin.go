// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/secret"
	"github.com/sirupsen/logrus"

	"github.com/hashicorp/vault/api"
)

// New returns a new secret plugin that sources secrets
// from the AWS secrets manager.
func New(client *api.Client, disallowForks bool) secret.Plugin {
	return &plugin{
		client:        client,
		disallowForks: disallowForks,
	}
}

type plugin struct {
	client        *api.Client
	disallowForks bool
}

func (p *plugin) Find(ctx context.Context, req *secret.Request) (*drone.Secret, error) {
	// The Fork attribute will be empty on a branch build (e.g. master).
	// Branch builds cannot be from a fork.
	isFork := req.Build.Fork != "" && req.Build.Fork != req.Repo.Slug
	forkRepo := ""
	if isFork {
		forkRepo = req.Build.Fork
	}

	logEvent := logrus.WithFields(logrus.Fields{
		"event":  req.Build.Event,
		"repo":   req.Repo.Slug,
		"ref":    req.Build.Ref,
		"secret": req.Path,
		"fork":   forkRepo,
	})

	path := req.Path
	name := req.Name
	if name == "" {
		name = "value"
	}

	// makes an api call to the aws secrets manager and attempts
	// to retrieve the secret at the requested path.
	params, err := p.find(path)
	if err != nil {
		return nil, errors.New("secret not found")
	}
	value, ok := params[name]
	if !ok {
		return nil, errors.New("secret key not found")
	}

	// the user can filter out requests based on event type
	// using the X-Drone-Events secret key. Check for this
	// user-defined filter logic.
	events := extractEvents(params)
	if !match(req.Build.Event, events) {
		msg := "access denied: event does not match"
		logEvent.WithField("allowed_events", events).Debug(msg)
		return nil, errors.New(msg)
	}

	// the user can filter out requests based on repository
	// using the X-Drone-Repos secret key. Check for this
	// user-defined filter logic.
	repos := extractRepos(params)
	if !match(req.Repo.Slug, repos) {
		msg := "access denied: repository does not match"
		logEvent.WithField("allowed_repos", repos).Debug(msg)
		return nil, errors.New(msg)
	}

	// the user can filter out requests based on repository
	// branch using the X-Drone-Branches secret key. Check
	// for this user-defined filter logic.
	branches := extractBranches(params)
	if !match(req.Build.Target, branches) {
		msg := "access denied: branch does not match"
		logEvent.WithField("allowed_branches", branches).Debug(msg)
		return nil, errors.New(msg)
	}

	// the user can disallow fork builds using the
	// X-Drone-Disallow-Forks secret key. Check for this
	// user-defined filter logic.
	disallowForks := p.disallowForks
	if secretSetting := extractDisallowForks(params); secretSetting != nil {
		disallowForks = *secretSetting
	}
	if disallowForks && isFork {
		msg := "access denied: forks are not allowed"
		logEvent.WithField("disallow_forks", disallowForks).Debug(msg)
		return nil, errors.New(msg)
	}

	logEvent.Debug("secret matched and returned")

	return &drone.Secret{
		Name: name,
		Data: value,
		Pull: true, // always true. use X-Drone-Events to prevent pull requests.
		Fork: true, // always true. use X-Drone-Disallow-Forks to prevent secrets from forks.
	}, nil
}

// helper function returns the secret from vault.
func (p *plugin) find(path string) (map[string]string, error) {
	isV2, path := p.rewritePath(path)

	secret, err := p.client.Logical().Read(path)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, errors.New("secret not found")
	}

	// the V2 api includes both "data" and "metadata" fields within the top level "data" -- we only care about data.
	// https://www.vaultproject.io/api-docs/secret/kv/kv-v1#sample-response
	// v1 data schema:
	// { properties: { data: { type: object, description: "the actual data" }}}
	// https://www.vaultproject.io/api/secret/kv/kv-v2#sample-response-1
	// v2 data schema:
	// { properties: { data: { properties: { data: { type: object, description: "the actual data" }}}}}
	if isV2 {
		v := secret.Data["data"]
		if data, ok := v.(map[string]interface{}); ok {
			secret.Data = data
		}
	}

	params := map[string]string{}
	for k, v := range secret.Data {
		s, ok := v.(string)
		if !ok {
			continue
		}
		params[k] = s
	}
	return params, err
}

// rewritePath rewrites a secret path if need be according to storage engine constraints; if it fails, it returns the
// original path.
//
// TL;DR: vault requires rewriting secret paths for the V2 engine mount points. This is most visible when you use the
// CLI to output curl strings:
//    $ vault kv get \
//      -output-curl-string \
//     	foo/versioned/bar
//    curl -H "X-Vault-Request: true" \
//   	-H "X-Vault-Token: $(vault print token)" \
//  	https://vault.example.com/v1/foo/versioned/data/bar
//
// Note the addition of "data" in the output curl string. This only occurs for the v2 engine. This function
// reproduces the logic from the CLI:
// https://github.com/hashicorp/vault/blob/7aa1ffa92ee61b977efad1488b8f309b1e2136df/command/kv_get.go#L94-L110
func (p *plugin) rewritePath(path string) (bool, string) {
	r := p.client.NewRequest("GET", "/v1/sys/internal/ui/mounts/"+path)
	resp, err := p.client.RawRequest(r)
	if err != nil {
		logrus.Debugf("failed querying mount point; defaulting to original: %v", err)
		return false, path
	}
	defer resp.Body.Close()
	isV2, rewritten, err := rewritePath(resp.Body, path)
	if err != nil {
		logrus.Debugf("failed rewriting; defaulting to original: %v", err)
		return false, path
	}
	logrus.Debugf("rewrote %q to %q", path, rewritten)
	return isV2, rewritten
}

func rewritePath(r io.Reader, original string) (isV2 bool, rewritten string, _ error) {
	defer func() {
		// never permit a trailing slash, no matter what user puts in
		rewritten = strings.TrimSuffix(rewritten, "/")
	}()
	/*
		Example v2 response:
		{
		  "request_id": "4a3a3ef6-d0a8-9a9b-d7eb-c320ef170b55",
		  "lease_id": "",
		  "renewable": false,
		  "lease_duration": 0,
		  "data": {
		    "accessor": "kv_f055aa7b",
		    "config": {
		      "default_lease_ttl": 0,
		      "force_no_cache": false,
		      "max_lease_ttl": 0
		    },
		    "description": "versioned encrypted key/value storage",
		    "local": false,
		    "options": {
		      "version": "2"
		    },
		    "path": "foo/versioned/",
		    "seal_wrap": false,
		    "type": "kv",
		    "uuid": "eb3b578c-a0bf-2a91-19dc-4155e8ae0116"
		  },
		  "wrap_info": null,
		  "warnings": null,
		  "auth": null
		}
	*/
	var response struct {
		Data struct {
			Options struct {
				Version string `json:"version"`
			} `json:"options"`
			Path string `json:"path"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r).Decode(&response); err != nil {
		return false, original, fmt.Errorf("failed parsing response: %v", err)
	}
	v, err := strconv.Atoi(response.Data.Options.Version)
	if err != nil || v != 2 {
		return false, original, nil // we only rewrite v2
	}

	mountPath := response.Data.Path
	if original == mountPath || original == strings.TrimSuffix(mountPath, "/") {
		return true, path.Join(mountPath, "data"), nil
	}

	return true, path.Join(mountPath, "data", strings.TrimPrefix(original, mountPath)), nil
}
