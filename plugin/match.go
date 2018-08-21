// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package plugin

import (
	"path"
	"strings"
)

func match(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	name = strings.ToLower(name)
	for _, pattern := range patterns {
		pattern = strings.ToLower(pattern)
		match, _ := path.Match(pattern, name)
		if match {
			return true
		}
	}
	return false
}
