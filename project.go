// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"git.townsourced.com/ironsmith/datastore"
)

const (
	enabledProjectDir = "enabled"
	deletedProjectDir = "deleted"
)

//stages
const (
	stageLoad    = "load"
	stageFetch   = "fetch"
	stageBuild   = "build"
	stageTest    = "test"
	stageRelease = "release"
)

const projectFilePoll = 30 * time.Second

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

type projectList struct {
	sync.RWMutex
	data map[string]*Project
}

var projects = projectList{
	data: make(map[string]*Project),
}

func (p *projectList) load() error {
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
			p.add(files[i].Name())
		}
	}

	time.AfterFunc(projectFilePoll, startProjectLoader)

	return nil
}

func (p *projectList) exists(name string) bool {
	p.RLock()
	defer p.RUnlock()

	_, ok := p.data[name]
	return ok
}

func (p *projectList) add(name string) {
	p.Lock()
	defer p.Unlock()

	prj := &Project{
		filename: name,
		Name:     name,
		stage:    stageLoad,
	}
	p.data[name] = prj

	go func() {
		prj.load()
	}()
}

// removeMissing removes projects that are missing from the passed in list of names
func (p *projectList) removeMissing(names []string) {
	p.Lock()
	defer p.Unlock()

	for i := range p.data {
		found := false
		for k := range names {
			if names[k] == i {
				found = true
			}
		}
		if !found {
			delete(p.data, i)
		}
	}
}

// startProjectLoader polls for new projects
func startProjectLoader() {
	dir, err := os.Open(filepath.Join(projectDir, enabledProjectDir))
	defer func() {
		if cerr := dir.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err != nil {
		log.Printf("Error in startProjectLoader opening the filepath %s: %s\n", dir, err)
		return
	}

	files, err := dir.Readdir(0)
	if err != nil {
		log.Printf("Error in startProjectLoader reading the dir %s: %s\n", dir, err)
		return
	}

	names := make([]string, len(files))

	for i := range files {
		if !files[i].IsDir() && filepath.Ext(files[i].Name()) == ".json" {
			names[i] = files[i].Name()
			if !projects.exists(files[i].Name()) {
				projects.add(files[i].Name())
			}
		}
	}

	//check for removed projects
	projects.removeMissing(names)

	time.AfterFunc(projectFilePoll, startProjectLoader)
}
