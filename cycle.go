// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git.townsourced.com/ironsmith/datastore"
)

func (p *Project) errHandled(err error) bool {
	if err == nil {
		return false
	}

	vlog("Error in project %s: %s\n", p.id(), err)

	if p.ds == nil {
		log.Printf("Error in project %s: %s\n", p.id(), err)
		return true
	}
	defer func() {
		err = p.ds.Close()
		if err != nil {
			log.Printf("Error closing the datastore for project %s: %s\n", p.id(), err)
		}
		p.ds = nil

		//clean up version folder if it exists

		if p.version != "" {
			err = os.RemoveAll(p.workingDir())
			if err != nil {
				log.Printf("Error deleting the version directory project %s version %s: %s\n",
					p.id(), p.version, err)
			}

		}
	}()

	lerr := p.ds.AddLog(p.version, p.stage, err.Error())
	if lerr != nil {
		log.Printf("Error logging an error in project %s: Original error %s, Logging Error: %s",
			p.id(), err, lerr)
	}

	return true
}

func (p *Project) id() string {
	if p.filename == "" {
		panic("invalid project filename")
	}
	return strings.TrimSuffix(p.filename, filepath.Ext(p.filename))
}

func (p *Project) dir() string {
	return filepath.Join(dataDir, p.id())
}

func (p *Project) workingDir() string {
	if p.hash == "" {
		panic(fmt.Sprintf("Working dir called with no version hash set for project %s", p.id()))
	}

	//It's probably overkill to use a sha1 hash to identify the build folder, when putting a simple
	// timestamp on instead would work just fine, but I like having the working dir tied directly to the
	// version returned by project script

	return filepath.Join(p.dir(), p.hash)
}

// prepData makes sure the project's data folder and data store is created
/*
	folder structure
	projectDataFolder/<project-name>/<project-version>

*/
func (p *Project) prepData() error {
	err := os.MkdirAll(p.dir(), 0777)
	if err != nil {
		return err
	}

	p.ds, err = datastore.Open(filepath.Join(p.dir(), p.id()+".ironsmith"))

	if err != nil {
		return err
	}

	return nil
}

/*
Project life cycle:
(Load Project file) -> (Fetch) -> (Build) -> (Test) -> (Release) - > (Sleep for polling period) ->
	(Reload Project File) -> (Fetch) -> etc...

*/

// load is the beginning of the cycle.  Loads / reloads the project file to make sure that the scripts are up-to-date
// call's fetch and triggers the next poll if one exists
func (p *Project) load() {
	p.version = ""
	p.hash = ""

	vlog("Entering %s stage for Project: %s\n", stageLoad, p.id())

	if p.filename == "" {
		p.errHandled(errors.New("Invalid project file name"))
		return
	}

	if !projects.exists(p.filename) {
		// project has been deleted
		// don't continue polling
		// move project data to deleted folder with a timestamp
		p.errHandled(os.Rename(p.dir(), filepath.Join(dataDir, deletedProjectDir,
			strconv.FormatInt(time.Now().Unix(), 10), p.id())))
		return
	}

	data, err := ioutil.ReadFile(filepath.Join(projectDir, enabledProjectDir, p.filename))
	if p.errHandled(err) {
		return
	}

	if p.errHandled(json.Unmarshal(data, p)) {
		return
	}

	p.stage = stageLoad

	if p.errHandled(p.prepData()) {
		return
	}

	if p.PollInterval != "" {
		p.poll, err = time.ParseDuration(p.PollInterval)
		if p.errHandled(err) {
			p.poll = 0
		}
	}

	p.fetch()

	//full cycle completed

	if p.poll > 0 {
		//start polling
		time.AfterFunc(p.poll, p.load)
	}
}

// fetch first runs the fetch script into a temporary directory
// then it runs the version script in the temp directory to see if there is a newer version of the
// fetched code, if there is then the temp dir is renamed to the version name
func (p *Project) fetch() {
	p.stage = stageFetch

	vlog("Entering %s stage for Project: %s\n", p.stage, p.id())
	tempDir := filepath.Join(p.dir(), strconv.FormatInt(time.Now().Unix(), 10))

	if p.errHandled(os.MkdirAll(tempDir, 0777)) {
		return
	}

	//fetch project
	fetchResult, err := runCmd(p.Fetch, tempDir)
	if p.errHandled(err) {
		return
	}

	// fetched succesfully, determine version
	version, err := runCmd(p.Version, tempDir)

	if p.errHandled(err) {
		return
	}

	p.version = strings.TrimSpace(string(version))

	lVer, err := p.ds.LatestVersion()
	if err != datastore.ErrNotFound && p.errHandled(err) {
		return
	}

	if p.version == lVer {
		// no new build clean up temp dir
		p.errHandled(os.RemoveAll(tempDir))

		vlog("No new version found for Project: %s Version: %s.\n", p.id(), p.version)
		return
	}

	p.hash = fmt.Sprintf("%x", sha1.Sum([]byte(p.version)))

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
	p.stage = stageBuild
	vlog("Entering %s stage for Project: %s Version: %s\n", p.stage, p.id(), p.version)

	output, err := runCmd(p.Build, p.workingDir())

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
	p.stage = stageTest
	vlog("Entering %s stage for Project: %s Version: %s\n", p.stage, p.id(), p.version)

	output, err := runCmd(p.Test, p.workingDir())

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
	p.stage = stageRelease
	vlog("Entering %s stage for Project: %s Version: %s\n", p.stage, p.id(), p.version)

	output, err := runCmd(p.Release, p.workingDir())

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

	if p.errHandled(p.ds.AddRelease(p.version, buff)) {
		return
	}

	//build successfull, remove working dir
	p.errHandled(os.RemoveAll(p.workingDir()))

	if p.errHandled(p.ds.Close()) {
		return
	}

	vlog("Project: %s Version %s built, tested, and released successfully.\n", p.id(), p.version)
}
