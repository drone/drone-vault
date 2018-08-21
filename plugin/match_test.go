// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package plugin

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		match    bool
	}{
		// direct match
		{
			name:     "octocat/Spoon-Fork",
			patterns: []string{"octocat/Spoon-Fork"},
			match:    true,
		},
		// wildcard match
		{
			name:     "octocat/Spoon-Fork",
			patterns: []string{"octocat/*"},
			match:    true,
		},
		// wildcard match
		{
			name:     "octocat/Spoon-Fork",
			patterns: []string{"github/*", "octocat/*"},
			match:    true,
		},
		// wildcard match, case-insensitive
		{
			name:     "OCTOCAT/HELLO-WORLD",
			patterns: []string{"octocat/HELLO-world"},
			match:    true,
		},
		// match when no filter
		{
			name:     "octocat/Spoon-Fork",
			patterns: []string{},
			match:    true,
		},
		// no wildcard match
		{
			name:     "octocat/Spoon-Fork",
			patterns: []string{"github/*"},
			match:    false,
		},
		// no direct match
		{
			name:     "octocat/Spoon-Fork",
			patterns: []string{"octocat/Hello-World"},
			match:    false,
		},
	}

	for _, test := range tests {
		got, want := match(test.name, test.patterns), test.match
		if got != want {
			t.Errorf("Want matched %v, got %v", want, got)
		}
	}
}
