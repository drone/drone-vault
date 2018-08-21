// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package token

import "time"

// Token represents the Vault token and token TTL.
type Token struct {
	Token string
	TTL   time.Duration
}
