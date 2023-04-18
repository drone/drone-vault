// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package plugin

import (
	"strconv"
	"strings"
)

// helper function extracts the branch filters from the
// secret payload in key value format.
func extractBranches(params map[string]string) []string {
	for key, value := range params {
		if strings.EqualFold(key, "X-Drone-Branches") {
			return parseCommaSeparated(value)
		}
	}
	return nil
}

// helper function extracts the repository filters from the
// secret payload in key value format.
func extractRepos(params map[string]string) []string {
	for key, value := range params {
		if strings.EqualFold(key, "X-Drone-Repos") {
			return parseCommaSeparated(value)
		}
	}
	return nil
}

// helper function extracts the event filters from the
// secret payload in key value format.
func extractEvents(params map[string]string) []string {
	for key, value := range params {
		if strings.EqualFold(key, "X-Drone-Events") {
			return parseCommaSeparated(value)
		}
	}
	return nil
}

// helper function extracts the fork filter from the
// secret payload in key value format.
func extractDisallowForks(params map[string]string) *bool {
	for key, value := range params {
		if strings.EqualFold(key, "X-Drone-Disallow-Forks") {
			v, _ := strconv.ParseBool(value)
			return &v // Allow non-truthy or non-falsey values are false
		}
	}
	return nil
}

func parseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}
