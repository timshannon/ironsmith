// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.townsourced.com/ironsmith/datastore"
)

func (p *Project) errHandled(err error) bool {
	if err == nil {
		return false
	}

	if p.ds == nil {
		log.Printf("Error in project %s: %s", p.id(), err)
		return true
	}
	defer func() {
		err = p.ds.Close()
		if err != nil {
			log.Printf("Error closing the datastore for project %s: %s", p.id(), err)
		}
		p.ds = nil

		//clean up version folder if it exists

		if p.version != "" {
			err = os.RemoveAll(p.verDir())
			log.Printf("Error deleting the version directory project %s version %s: %s",
				p.id(), p.version, err)

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

func (p *Project) verDir() string {
	return filepath.Join(p.dir(), p.version)
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

	if p.filename == "" {
		p.errHandled(errors.New("Invalid project file name"))
		return
	}

	if !projects.exists(p.filename) {
		// project has been deleted
		// don't continue polling
		// move project data to deleted folder
		p.errHandled(os.Rename(p.dir(), filepath.Join(dataDir, deletedProjectDir, p.id())))
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

// fetch first runs the version script and checks the returned version against the latest version in the
// project database. If the version hasn't changed, then it breaks out of the cycle early doing nothing
// if the version has changed, then it runs the fetch script
func (p *Project) fetch() {
	p.stage = stageFetch
	verCmd := &exec.Cmd{
		Path: p.Version,
		Dir:  p.dir(),
	}

	version, err := verCmd.Output()

	if p.errHandled(err) {
		return
	}

	p.version = string(version)

	lVer, err := p.ds.LatestVersion()
	if err != datastore.ErrNotFound && p.errHandled(err) {
		return
	}

	if p.version == lVer {
		// no new build
		return
	}

	if p.errHandled(os.MkdirAll(p.verDir(), 0777)) {
		return
	}

	//fetch project
	fetchCmd := &exec.Cmd{
		Path: p.Fetch,
		Dir:  p.verDir(),
	}

	fetchResult, err := fetchCmd.Output()
	if p.errHandled(err) {
		return
	}

	if p.errHandled(p.ds.AddLog(p.stage, p.version, string(fetchResult))) {
		return
	}

	// fetched succesfully, onto the build stage
	p.build()

}

// build  runs the build scripts to build the project which should result in the a single file
// configured in the ReleaseFile section of the project file
func (p *Project) build() {
	p.stage = stageBuild

	buildCmd := &exec.Cmd{
		Path: p.Build,
		Dir:  p.verDir(),
	}

	output, err := buildCmd.Output()

	if p.errHandled(err) {
		return
	}

	if p.errHandled(p.ds.AddLog(p.stage, p.version, string(output))) {
		return
	}

	// built successfully, onto test stage
	p.test()
}

// test runs the test scripts
func (p *Project) test() {
	p.stage = stageTest

	testCmd := &exec.Cmd{
		Path: p.Test,
		Dir:  p.verDir(),
	}

	output, err := testCmd.Output()

	if p.errHandled(err) {
		return
	}

	if p.errHandled(p.ds.AddLog(p.stage, p.version, string(output))) {
		return
	}

	//  Tests passed, onto release
	p.release()

}

// release runs the release scripts and builds the release file
func (p *Project) release() {
	p.stage = stageRelease

	releaseCmd := &exec.Cmd{
		Path: p.Release,
		Dir:  p.verDir(),
	}

	output, err := releaseCmd.Output()

	if p.errHandled(err) {
		return
	}

	if p.errHandled(p.ds.AddLog(p.stage, p.version, string(output))) {
		return
	}

	//get release file
	f, err := os.Open(filepath.Join(p.verDir(), p.ReleaseFile))
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

}
