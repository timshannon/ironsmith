// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"net/http"
	"net/url"
	"strings"
)

// /path/<project-id>/<version>/<stage>
func splitPath(path string) (project, version, stage string) {
	s := strings.Split(path, "/")
	if len(s) < 3 {
		return
	}

	project, _ = url.QueryUnescape(s[2])
	if len(s) < 4 {
		return
	}

	version, _ = url.QueryUnescape(s[3])

	if len(s) < 5 {
		return
	}
	stage, _ = url.QueryUnescape(s[4])

	return
}

// /project/*
func projectGet(w http.ResponseWriter, r *http.Request) {
	prj, ver, _ := splitPath(r.URL.Path)

	//values := r.URL.Query()

	if prj == "" {
		//get all projects
		pList, err := projects.webList()
		if errHandled(err, w) {
			return
		}

		respondJsend(w, &JSend{
			Status: statusSuccess,
			Data:   pList,
		})

		return
	}

	project, ok := projects.get(prj)
	if !ok {
		four04(w, r)
		return
	}

	//project found

	if ver == "" {
		//list versions
		vers, err := project.versions()
		if errHandled(err, w) {
			return
		}
		respondJsend(w, &JSend{
			Status: statusSuccess,
			Data:   vers,
		})
		return
	}

}
