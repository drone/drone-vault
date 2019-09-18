// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package token

import "time"

// Token represents the Vault token and token TTL.
type Token struct {
	Token string
	TTL   time.Duration
}
