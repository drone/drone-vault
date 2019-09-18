// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package kubernetes

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/drone/drone-vault/plugin/token"
)

// Name that identifies the auth method.
const Name = "kubernetes"

// kubernetes token file path.
const path = "/var/run/secrets/kubernetes.io/serviceaccount/token"

type (
	// kubernetes authorization provider request.
	request struct {
		Jwt  string `json:"jwt"`
		Role string `json:"role"`
	}

	// kubernetes authorization provider response.
	response struct {
		Auth struct {
			Token string `json:"client_token"`
			Lease int    `json:"lease_duration"`
		}
	}
)

// Load loads the Vault token using the Kubernetes
// authorization provider.
func Load(address, role, mount string) (*token.Token, error) {
	return load(address, role, mount, path)
}

func load(address, role, mount, tokenpath string) (*token.Token, error) {
	// create the vault endpoint address.
	endpoint := fmt.Sprintf("%s/v1/auth/%s/login", address, mount)

	// reads the jwt token mounted inside the container.
	b, err := ioutil.ReadFile(tokenpath)
	if err != nil {
		return nil, err
	}

	res := &response{}
	req := &request{
		Jwt:  string(b),
		Role: role,
	}

	err = post(endpoint, req, res)
	if err != nil {
		return nil, err
	}

	// convert the response to the generic token structure
	// with the token ttl calculated from the lease.
	return &token.Token{
		Token: res.Auth.Token,
		TTL:   time.Duration(res.Auth.Lease) * time.Second,
	}, nil
}
