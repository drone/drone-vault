// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package plugin

import (
	"reflect"
	"testing"
)

func TestExtractBranches(t *testing.T) {
	tests := []struct {
		params   map[string]string
		patterns []string
	}{
		{
			params:   map[string]string{"X-Drone-Branches": ""},
			patterns: nil,
		},
		{
			params:   map[string]string{"X-Drone-Branches": "master"},
			patterns: []string{"master"},
		},
		{
			params:   map[string]string{"X-Drone-Branches": "master,development"},
			patterns: []string{"master", "development"},
		},
		{
			params:   map[string]string{"x-drone-branches": "master,development"},
			patterns: []string{"master", "development"},
		},
		{
			params:   map[string]string{"foo": "bar"},
			patterns: nil,
		},
	}

	for i, test := range tests {
		got, want := extractBranches(test.params), test.patterns
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unexpected results at %d", i)
		}
	}
}

func TestExtractRepos(t *testing.T) {
	tests := []struct {
		params   map[string]string
		patterns []string
	}{
		{
			params:   map[string]string{"X-Drone-Repos": ""},
			patterns: nil,
		},
		{
			params:   map[string]string{"X-Drone-Repos": "octocat/Spoon-Fork"},
			patterns: []string{"octocat/Spoon-Fork"},
		},
		{
			params:   map[string]string{"X-Drone-Repos": "octocat/Spoon-Fork,octocat/Hello-World"},
			patterns: []string{"octocat/Spoon-Fork", "octocat/Hello-World"},
		},
		{
			params:   map[string]string{"x-drone-repos": "octocat/Spoon-Fork,octocat/Hello-World"},
			patterns: []string{"octocat/Spoon-Fork", "octocat/Hello-World"},
		},
		{
			params:   map[string]string{"foo": "bar"},
			patterns: nil,
		},
	}

	for i, test := range tests {
		got, want := extractRepos(test.params), test.patterns
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unexpected results at %d", i)
		}
	}
}

func TestExtractEvents(t *testing.T) {
	tests := []struct {
		params   map[string]string
		patterns []string
	}{
		{
			params:   map[string]string{"X-Drone-Events": ""},
			patterns: nil,
		},
		{
			params:   map[string]string{"X-Drone-Events": "push"},
			patterns: []string{"push"},
		},
		{
			params:   map[string]string{"X-Drone-Events": "push,tag"},
			patterns: []string{"push", "tag"},
		},
		{
			params:   map[string]string{"x-drone-events": "push,tag"},
			patterns: []string{"push", "tag"},
		},
		{
			params:   map[string]string{"foo": "bar"},
			patterns: nil,
		},
	}

	for i, test := range tests {
		got, want := extractEvents(test.params), test.patterns
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unexpected results at %d", i)
		}
	}
}
