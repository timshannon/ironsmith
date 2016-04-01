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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"git.townsourced.com/ironsmith/datastore"
)

const enabledProjectDir = "enabled"

//stages
const (
	stageLoad    = "load"
	stageFetch   = "fetch"
	stageBuild   = "build"
	stageTest    = "test"
	stageRelease = "release"
)

// Project is an ironsmith project that contains how to fetch, build, test, and release a project
/*
The project lifecycle goes like this, each step calling the next if successful
	(Load Project file) -> (Fetch) -> (Build) -> (Test) -> (Release) - > (Sleep for polling period) ->
	(Reload Project File) -> (Fetch) -> etc...

	Changes the project file will be reloaded on every poll / trigger
	If a project file is deleted then the cycle will finish it's current poll and stop at the load phase
*/
type Project struct {
	Name string `json:"name"` // name of the project

	Fetch   string `json:"fetch"`   //Script to fetch the latest project code into the current directory
	Build   string `json:"build"`   //Script to build the latest project code
	Test    string `json:"test"`    //Script to test the latest project code
	Release string `json:"release"` //Script to build the release of latest project code

	Version string `json:"version"` //Script to generate the version num of the current build, should be indempotent

	ReleaseFile   string `json:"releaseFile"`
	PollInterval  string `json:"pollInterval"`  // if not poll interval is specified, this project is trigger only
	TriggerSecret string `json:"triggerSecret"` //secret to be included with a trigger call

	filename string
	poll     time.Duration
	ds       *datastore.Store
	stage    string
	version  string
}

const projectTemplateFilename = "template.project.json"

var projectTemplate = &Project{
	Name:    "Template Project",
	Fetch:   "git clone root@git.townsourced.com:tshannon/ironsmith.git .",
	Build:   "sh ./ironsmith/build.sh",
	Test:    "sh ./ironsmith/test.sh",
	Release: "sh ./ironsmith/release.sh",
	Version: "git describe --tags --long",

	ReleaseFile:  `json:"./ironsmith/release.tar.gz"`,
	PollInterval: "15m",
}

func prepTemplateProject() error {
	filename := filepath.Join(projectDir, projectTemplateFilename)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		f, err := os.Create(filename)
		defer func() {
			if cerr := f.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		if err != nil {
			return err
		}

		data, err := json.MarshalIndent(projectTemplate, "", "    ")
		if err != nil {
			return err
		}

		_, err = f.Write(data)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	return nil
}

func (p *Project) errHandled(err error) bool {
	if err == nil {
		return false
	}

	if p.ds == nil {
		log.Printf("Error in project %s: %s", p.filename, err)
		return true
	}

	p.ds.AddLog(p.version, p.stage, err.Error())

	return true
}

func (p *Project) load() {

	if p.filename == "" {
		p.errHandled(errors.New("Invalid project file name"))
		return
	}

	if !projects.exists(p.filename) {
		// project has been deleted
		// don't continue polling
		// TODO: Clean up Project data folder?
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

	//TODO: call fetch

	if p.PollInterval != "" {
		p.poll, err = time.ParseDuration(p.PollInterval)
		if p.errHandled(err) {
			p.poll = 0
		}
	}

	if p.poll > 0 {
		//start polling
	}

}

// prepData makes sure the project's data folder and data store is created
/*
	folder structure
	projectDataFolder/<project-name>/<project-version>

*/
func (p *Project) prepData() error {
	var name = strings.TrimSuffix(p.filename, filepath.Ext(p.filename))
	var dir = filepath.Join(dataDir, name)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}

	p.ds, err = datastore.Open(filepath.Join(dir, name+".ironsmith"))

	if err != nil {
		return err
	}

	return nil
}

type projectList struct {
	sync.RWMutex
	data map[string]*Project
}

var projects = projectList{
	data: make(map[string]*Project),
}

func (p *projectList) load() error {
	p.Lock()
	defer p.Unlock()

	dir, err := os.Open(filepath.Join(projectDir, enabledProjectDir))
	defer func() {
		if cerr := dir.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err != nil {
		return err
	}

	files, err := dir.Readdir(0)
	if err != nil {
		return err
	}

	for i := range files {
		if !files[i].IsDir() && filepath.Ext(files[i].Name()) == ".json" {
			prj := &Project{
				filename: files[i].Name(),
				Name:     files[i].Name(),
				version:  "starting up",
				stage:    stageLoad,
			}
			p.data[files[i].Name()] = prj

			prj.load()
		}
	}

	return nil
}

func (p *projectList) remove(name string) {
	p.Lock()
	delete(p.data, name)
	p.Unlock()
}

func (p *projectList) exists(name string) bool {
	p.RLock()
	defer p.RUnlock()

	_, ok := p.data[name]
	return ok
}

// startProjectLoader polls for new projects
func startProjectLoader() {

}
