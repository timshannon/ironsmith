// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/timshannon/ironsmith/datastore"
)

/*
Project life cycle:
(Load Project file) -> (Fetch) -> (Build) -> (Test) -> (Release) - > (Sleep for polling period) ->
	(Reload Project File) -> (Fetch) -> etc...

*/

// load is the beginning of the cycle.  Loads / reloads the project file to make sure that the scripts are up-to-date
// call's fetch and triggers the next poll if one exists
func (p *Project) load(forceBuild bool) {
	p.processing.Lock() // ensure only one cycle is running at a time per project
	defer p.processing.Unlock()

	p.setStage(stageLoad)
	p.setVersion("Version not yet set")
	p.start = time.Time{}

	if p.filename == "" {
		p.errHandled(errors.New("Invalid project file name"))
		return
	}

	if _, ok := projects.get(p.id()); !ok {
		// project has been deleted
		// don't continue polling
		// move project data to deleted folder with a timestamp
		if p.errHandled(p.close()) {
			return
		}
		p.errHandled(os.Rename(p.dir(), filepath.Join(dataDir, deletedProjectDir,
			strconv.FormatInt(time.Now().Unix(), 10), p.id())))
		return
	}

	data, err := ioutil.ReadFile(filepath.Join(projectDir, enabledProjectDir, p.filename))
	if p.errHandled(err) {
		return
	}

	new := &Project{}
	if p.errHandled(json.Unmarshal(data, new)) {
		return
	}

	p.setData(new)

	p.fetch(forceBuild)

	p.setStage(stageWait)

	//full cycle completed
	p.errHandled(p.ds.TrimVersions(p.MaxVersions))

	if p.poll > 0 {
		//start polling
		time.AfterFunc(p.poll, func() {
			p.load(false)
		})
	}
}

// fetch first runs the fetch script into a temporary directory
// then it runs the version script in the temp directory to see if there is a newer version of the
// fetched code, if there is then the temp dir is renamed to the version name
func (p *Project) fetch(forceBuild bool) {
	p.setStage(stageFetch)
	p.start = time.Now()

	if p.Fetch == "" {
		return
	}

	tempDir := filepath.Join(p.dir(), strconv.FormatInt(time.Now().Unix(), 10))

	if p.errHandled(os.MkdirAll(tempDir, 0777)) {
		return
	}

	//fetch project
	fetchResult, err := runCmd(p.Fetch, tempDir, p.Environment)
	if p.errHandled(err) {
		return
	}

	// fetched succesfully, determine version
	version, err := runCmd(p.Version, tempDir, p.Environment)

	if p.errHandled(err) {
		return
	}

	p.setVersion(strings.TrimSpace(string(version)))

	if !forceBuild {
		// if not forced build, then check if this specific version has attempted a build yet
		lVer, err := p.ds.LastVersion(stageBuild)
		if err != datastore.ErrNotFound && p.errHandled(err) {
			return
		}

		if p.version == "" || p.version == lVer.Version {
			// no new build clean up temp dir
			p.errHandled(os.RemoveAll(tempDir))

			vlog("No new version found for Project: %s Version: %s.\n", p.id(), p.version)
			return
		}
	}

	//remove any existing data that matches version hash
	if p.errHandled(os.RemoveAll(p.workingDir())) {
		return
	}

	//new version move tempdir to workingDir
	if p.errHandled(os.Rename(tempDir, p.workingDir())) {
		// cleanup temp dir if rename failed
		p.errHandled(os.RemoveAll(tempDir))
		return
	}

	//log fetch results
	if p.errHandled(p.ds.AddLog(p.version, p.stage, string(fetchResult))) {
		return
	}

	// continue to build
	p.build()
}

// build  runs the build scripts to build the project which should result in the a single file
// configured in the ReleaseFile section of the project file
func (p *Project) build() {
	p.setStage(stageBuild)

	if p.Build == "" {
		return
	}

	output, err := runCmd(p.Build, p.workingDir(), p.Environment)

	if p.errHandled(err) {
		return
	}

	if p.errHandled(p.ds.AddLog(p.version, p.stage, string(output))) {
		return
	}

	// built successfully, onto test stage
	p.test()
}

// test runs the test scripts
func (p *Project) test() {
	p.setStage(stageTest)

	if p.Test == "" {
		return
	}
	output, err := runCmd(p.Test, p.workingDir(), p.Environment)

	if p.errHandled(err) {
		return
	}

	if p.errHandled(p.ds.AddLog(p.version, p.stage, string(output))) {
		return
	}

	//  Tests passed, onto release
	p.release()
}

// release runs the release scripts and builds the release file
func (p *Project) release() {
	p.setStage(stageRelease)

	if p.Release == "" {
		return
	}

	output, err := runCmd(p.Release, p.workingDir(), p.Environment)

	if p.errHandled(err) {
		return
	}

	if p.errHandled(p.ds.AddLog(p.version, p.stage, string(output))) {
		return
	}

	//get release file
	f, err := os.Open(filepath.Join(p.workingDir(), p.ReleaseFile))
	if p.errHandled(err) {
		return
	}

	buff, err := ioutil.ReadAll(f)
	if p.errHandled(err) {
		return
	}

	if p.errHandled(p.ds.AddRelease(p.version, filepath.Base(p.ReleaseFile), buff)) {
		return
	}

	p.setStage(stageReleased)

	if p.errHandled(p.ds.AddLog(p.version, p.stage,
		fmt.Sprintf("Project %s Version %s built, tested, and released successfully and took %s.\n", p.id(), p.version,
			time.Now().Sub(p.start)))) {
		return
	}

	//build successfull, remove working dir
	p.errHandled(os.RemoveAll(p.workingDir()))

	vlog("Project %s Version %s built, tested, and released successfully.\n", p.id(), p.version)
}
