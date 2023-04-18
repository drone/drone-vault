// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"errors"

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
	secret, err := p.client.Logical().Read(path)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, errors.New("secret not found")
	}

	// HACK: the vault v2 key value store is confusing
	// and I could not quite figure out how to work with
	// the api. This is the workaround I came up with.
	v := secret.Data["data"]
	if data, ok := v.(map[string]interface{}); ok {
		secret.Data = data
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
