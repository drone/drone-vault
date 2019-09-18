// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

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
