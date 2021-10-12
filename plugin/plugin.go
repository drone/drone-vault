// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/secret"

	"github.com/hashicorp/vault/api"
)

// New returns a new secret plugin that sources secrets
// from the AWS secrets manager.
func New(client *api.Client) secret.Plugin {
	return &plugin{
		client: client,
	}
}

type plugin struct {
	client *api.Client
}

func (p *plugin) Find(ctx context.Context, req *secret.Request) (*drone.Secret, error) {
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

	// the user can filter out requets based on event type
	// using the X-Drone-Events secret key. Check for this
	// user-defined filter logic.
	events := extractEvents(params)
	if !match(req.Build.Event, events) {
		return nil, errors.New("access denied: event does not match")
	}

	// the user can filter out requets based on repository
	// using the X-Drone-Repos secret key. Check for this
	// user-defined filter logic.
	repos := extractRepos(params)
	if !match(req.Repo.Slug, repos) {
		return nil, errors.New("access denied: repository does not match")
	}

	// the user can filter out requets based on repository
	// branch using the X-Drone-Branches secret key. Check
	// for this user-defined filter logic.
	branches := extractBranches(params)
	if !match(req.Build.Target, branches) {
		return nil, errors.New("access denied: branch does not match")
	}

	return &drone.Secret{
		Name: name,
		Data: value,
		Pull: true, // always true. use X-Drone-Events to prevent pull requests.
		Fork: true, // always true. use X-Drone-Events to prevent pull requests.
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
	for k, rawValue := range secret.Data {
		var s string
		switch castValue := rawValue.(type) {
		case bool:
			s = strconv.FormatBool(castValue)
		case string:
			s = castValue
		case int:
			s = strconv.FormatInt(int64(castValue), 10)
		case int8:
			s = strconv.FormatInt(int64(castValue), 10)
		case int16:
			s = strconv.FormatInt(int64(castValue), 10)
		case int32:
			s = strconv.FormatInt(int64(castValue), 10)
		case int64:
			s = strconv.FormatInt(int64(castValue), 10)
		case uint:
			s = strconv.FormatUint(uint64(castValue), 10)
		case uint8:
			s = strconv.FormatUint(uint64(castValue), 10)
		case uint16:
			s = strconv.FormatUint(uint64(castValue), 10)
		case uint32:
			s = strconv.FormatUint(uint64(castValue), 10)
		case uint64:
			s = strconv.FormatUint(uint64(castValue), 10)
		default:
			s = fmt.Sprint(castValue)
		}

		params[k] = s
	}
	return params, err
}
