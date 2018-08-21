// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package plugin

import "strings"

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

func parseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}
