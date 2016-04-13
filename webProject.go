// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"time"
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

/*
	/log/ - list all projects
	/log/<project-id> - list all versions in a project,  POST triggers new builds
	/log/<project-id>/<version> - list combined output of all stages for a given version
	/log/<project-id>/<version>/<stage> - list output of a given stage of a given version
*/
func logGet(w http.ResponseWriter, r *http.Request) {
	prj, ver, stg := splitPath(r.URL.Path)

	if prj == "" {
		///log/ - list all projects
		pList, err := projects.webList()
		if errHandled(err, w, r) {
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
		///log/<project-id> - list all versions in a project

		vers, err := project.versions()
		if errHandled(err, w, r) {
			return
		}
		respondJsend(w, &JSend{
			Status: statusSuccess,
			Data:   vers,
		})
		return
	}

	//ver found
	if stg == "" {
		///log/<project-id>/<version> - list combined output of all stages for a given version
		logs, err := project.versionLog(ver)
		if errHandled(err, w, r) {
			return
		}
		respondJsend(w, &JSend{
			Status: statusSuccess,
			Data:   logs,
		})
		return
	}

	//stage found
	///log/<project-id>/<version>/<stage> - list output of a given stage of a given version

	log, err := project.stageLog(ver, stg)
	if errHandled(err, w, r) {
		return
	}

	respondJsend(w, &JSend{
		Status: statusSuccess,
		Data:   log,
	})
	return
}

/*
	/release/<project-id>/<version>

	/release/<project-id> - list last release for a given project
		?all returns all the releases for a project ?file returns the last release file
	/release/<project-id>/<version> - list release for a given project version ?file returns the file for a given release version
*/
func releaseGet(w http.ResponseWriter, r *http.Request) {
	prj, ver, _ := splitPath(r.URL.Path)

	values := r.URL.Query()

	_, all := values["all"]
	_, file := values["file"]

	if prj == "" {
		four04(w, r)
		return
	}

	project, ok := projects.get(prj)
	if !ok {
		four04(w, r)
		return
	}

	//project found

	if ver == "" {
		///release/<project-id> - list last release for a given project
		//	?all returns all the releases for a project ?file returns the last release file

		if all {
			releases, err := project.releases()
			if errHandled(err, w, r) {
				return
			}

			respondJsend(w, &JSend{
				Status: statusSuccess,
				Data:   releases,
			})
			return

		}

		last, err := project.lastRelease()
		if errHandled(err, w, r) {
			return
		}

		if file {
			fileData, err := project.releaseFile(last.FileKey)
			if errHandled(err, w, r) {
				return
			}

			http.ServeContent(w, r, last.FileName, time.Time{}, bytes.NewReader(fileData))
			return
		}

		respondJsend(w, &JSend{
			Status: statusSuccess,
			Data:   last,
		})

		return
	}

	//ver found
	// /release/<project-id>/<version> - list release for a given project version ?file returns the file for a given release version

	release, err := project.releaseData(ver)
	if errHandled(err, w, r) {
		return
	}

	if file {
		fileData, err := project.releaseFile(release.FileKey)
		if errHandled(err, w, r) {
			return
		}
		http.ServeContent(w, r, release.FileName, time.Time{}, bytes.NewReader(fileData))
		return
	}

	respondJsend(w, &JSend{
		Status: statusSuccess,
		Data:   release,
	})
}
