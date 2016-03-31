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
	"strings"
	"sync"
	"time"
)

const enabledProjectDir = "enabled"

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

func (p *Project) load() error {
	if p.filename == "" {
		return fmt.Errorf("Invalid project file name")
	}

	if !projects.exists(p.filename) {
		// project has been deleted
		// don't continue polling
		// TODO: Clean up Project data folder?
		return nil
	}

	data, err := ioutil.ReadFile(filepath.Join(projectDir, enabledProjectDir, p.filename))
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, p)
	if err != nil {
		return err
	}

	err = p.prepData()
	if err != nil {
		return err
	}

	if p.PollInterval != "" {
		p.poll, err = time.ParseDuration(p.PollInterval)
		if err != nil {
			//TODO: Log pollInterval parse failure in project store
			p.poll = 0
		}
	}

	if p.poll > 0 {
		//start polling
	}

	return nil
}

// prepData makes sure the project's data folder and data store is created
/*
	folder structure
	projectDataFolder/<project-name>/<project-version>

*/
func (p *Project) prepData() error {
	var dirName = strings.TrimSuffix(p.filename, filepath.Ext(p.filename))
	err := os.MkdirAll(filepath.Join(dataDir, dirName), 0777)
	if err != nil {
		return err
	}

	//TODO: Create data store

	return errors.New("TODO")
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
			prj := &Project{filename: files[i].Name()}
			p.data[files[i].Name()] = prj

			err = prj.load()
			if err != nil {
				delete(p.data, files[i].Name())
				return err
			}
		}
	}

	return nil
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
