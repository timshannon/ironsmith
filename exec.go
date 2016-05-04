// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runCmd(cmd, dir string, env []string) ([]byte, error) {
	s := strings.Fields(strings.Replace(cmd, "@dir", dir, -1))

	for i := range env {
		env[i] = strings.Replace(env[i], "@dir", dir, -1)
	}

	var args []string

	if len(s) > 1 {
		args = s[1:]
	}

	name := s[0]
	ec := &exec.Cmd{
		Path: name,
		Args: append([]string{name}, args...),
		Dir:  dir,
		Env:  env,
	}

	if filepath.Base(name) == name {
		lp, err := lookPath(name, env)
		if err != nil {
			return nil, err
		}
		ec.Path = lp
	}

	vlog("Executing command: %s in dir %s\n", cmd, dir)

	result, err := ec.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s\n%s", err, result)
	}
	return result, nil
}

// similar to os/exec.LookPath, except it checks if the passed in
// custom environment includes a path definitions and uses that path instead
// note this probably only works on unix, that's all I care about for now
func lookPath(file string, env []string) (string, error) {
	if strings.Contains(file, "/") {
		err := findExecutable(file)
		if err == nil {
			return file, nil
		}
		return "", &exec.Error{Name: file, Err: err}
	}

	for i := range env {
		if strings.HasPrefix(env[i], "PATH=") {
			pathenv := env[i][5:]
			if pathenv == "" {
				return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
			}
			for _, dir := range strings.Split(pathenv, ":") {
				if dir == "" {
					// Unix shell semantics: path element "" means "."
					dir = "."
				}
				path := dir + "/" + file
				if err := findExecutable(path); err == nil {
					return path, nil
				}
			}
			return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
		}
	}

	return exec.LookPath(file)
}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}

	return os.ErrPermission
}
