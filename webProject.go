// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// /project/<project-id>/<version>/<stage>
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

func projectGet(w http.ResponseWriter, r *http.Request) {
	prj, ver, stg := splitPath(r.URL.Path)

	fmt.Printf("Project: %s Version: %s Stage: %s\n", prj, ver, stg)
}
