// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func runCmd(cmd, dir string) ([]byte, error) {
	s := strings.Fields(cmd)

	var args []string

	if len(s) > 1 {
		args = s[1:]
	}

	ec := exec.Command(s[0], args...)

	ec.Dir = dir

	vlog("Executing command: %s in dir %s\n", cmd, dir)

	result, err := ec.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s:\n%s", err, result)
	}
	return result, nil
}
