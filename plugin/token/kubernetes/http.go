// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package kubernetes

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

func post(path string, in, out interface{}) error {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(in)
	if err != nil {
		return err
	}
	res, err := http.Post(path, "application/json", buf)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return errors.New(
			res.Status,
		)
	}
	return json.NewDecoder(res.Body).Decode(out)
}
