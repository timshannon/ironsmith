// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	stageLoad    = "loading"
	stageFetch   = "fetching"
	stageBuild   = "building"
	stageTest    = "testing"
	stageRelease = "releasing"
	stageWait    = "waiting"
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
	PollInterval  string `json:"pollInterval,omitempty"`  // if not poll interval is specified, this project is trigger only
	TriggerSecret string `json:"triggerSecret,omitempty"` //secret to be included with a trigger call

	filename string
	poll     time.Duration
	ds       *datastore.Store
	stage    string
	status   string
	version  string
	hash     string

	sync.RWMutex
	processing sync.Mutex
}

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

func projectID(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

func (p *Project) id() string {
	if p.filename == "" {
		panic("invalid project filename")
	}
	return projectID(p.filename)
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
func (p *Project) open() error {
	p.Lock()
	defer p.Unlock()

	if p.ds != nil {
		return nil
	}

	err := os.MkdirAll(p.dir(), 0777)
	if err != nil {
		return err
	}

	ds, err := datastore.Open(filepath.Join(p.dir(), p.id()+".ironsmith"))
	if err != nil {
		return err
	}

	p.ds = ds

	return nil
}

func (p *Project) setVersion(version string) {
	p.Lock()
	defer p.Unlock()

	p.version = version
	if version == "" {
		p.hash = ""
		return
	}

	p.hash = fmt.Sprintf("%x", sha1.Sum([]byte(version)))
}

func (p *Project) setStage(stage string) {
	p.Lock()
	defer p.Unlock()

	if p.version != "" {
		vlog("Entering %s stage for Project: %s Version: %s\n", stage, p.id(), p.version)
	} else {
		vlog("Entering %s stage for Project: %s\n", stage, p.id())
	}

	p.stage = stage
}

type webProject struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	ReleaseVersion string         `json:"releaseVersion"` //last successfully released version
	Stage          string         `json:"stage"`          // current stage
	LastLog        *datastore.Log `json:"lastLog"`
}

func (p *Project) webData() (*webProject, error) {
	p.RLock()
	defer p.RUnlock()

	last, err := p.ds.LastVersion("")
	if err != nil {
		return nil, err
	}

	release, err := p.ds.LastVersion(stageRelease)
	if err != nil {
		return nil, err
	}

	d := &webProject{
		Name:           p.Name,
		ID:             p.id(),
		ReleaseVersion: release.Version,
		Stage:          p.stage,
		LastLog:        last,
	}

	return d, nil
}

func (p *Project) versions() ([]*datastore.Log, error) {
	p.RLock()
	defer p.RUnlock()

	return p.ds.Versions()
}

func (p *Project) versionLog(version string) ([]*datastore.Log, error) {
	p.RLock()
	defer p.RUnlock()

	return p.ds.VersionLog(version)
}

func (p *Project) stageLog(version, stage string) (*datastore.Log, error) {
	p.RLock()
	defer p.RUnlock()

	return p.ds.StageLog(version, stage)
}

func (p *Project) releases() ([]*datastore.Release, error) {
	p.RLock()
	defer p.RUnlock()

	return p.ds.Releases()
}

func (p *Project) lastRelease() (*datastore.Release, error) {
	p.RLock()
	defer p.RUnlock()

	return p.ds.LastRelease()
}
func (p *Project) releaseData(version string) (*datastore.Release, error) {
	p.RLock()
	defer p.RUnlock()

	return p.ds.Release(version)
}

func (p *Project) releaseFile(fileKey datastore.TimeKey) ([]byte, error) {
	p.RLock()
	defer p.RUnlock()

	return p.ds.ReleaseFile(fileKey)
}

// releaseFile

func (p *Project) setData(new *Project) {
	p.Lock()
	defer p.Unlock()

	p.Name = new.Name

	p.Fetch = new.Fetch
	p.Build = new.Build
	p.Test = new.Test
	p.Release = new.Release
	p.Version = new.Version

	p.ReleaseFile = new.ReleaseFile
	p.PollInterval = new.PollInterval
	p.TriggerSecret = new.TriggerSecret

	if p.PollInterval != "" {
		var err error
		p.poll, err = time.ParseDuration(p.PollInterval)
		if p.errHandled(err) {
			p.poll = 0
		}
	}
}

func (p *Project) close() error {
	p.Lock()
	defer p.Unlock()
	if p.ds == nil {
		return nil
	}
	err := p.ds.Close()
	if err != nil {
		return err
	}

	p.ds = nil
	return nil
}

const projectTemplateFilename = "template.project.json"

var projectTemplate = &Project{
	Name:    "Template Project",
	Fetch:   "git clone root@git.townsourced.com:tshannon/ironsmith.git .",
	Build:   "go build -a -v -o ironsmith",
	Test:    "go test ./...",
	Release: "tar -czf release.tar.gz ironsmith",
	Version: "git describe --tags --long",

	ReleaseFile:  "release.tar.gz",
	PollInterval: "15m",
}

func prepTemplateProject() error {
	filename := filepath.Join(projectDir, projectTemplateFilename)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		vlog("Creating template project file in %s", filename)
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
	vlog("Loading projects from the enabled definitions in %s\n", filepath.Join(projectDir, enabledProjectDir))
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

func (p *projectList) get(name string) (*Project, bool) {
	p.RLock()
	defer p.RUnlock()

	prj, ok := p.data[name]
	return prj, ok
}

func (p *projectList) add(name string) {
	vlog("Adding project %s to the project list.\n", name)
	p.Lock()
	defer p.Unlock()

	prj := &Project{
		filename: name,
		Name:     name,
		stage:    stageLoad,
	}
	p.data[projectID(name)] = prj

	go func() {
		err := prj.open()
		if err != nil {
			log.Printf("Error opening datastore for Project: %s Error: %s\n", prj.id(), err)
			return
		}
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
			if projectID(names[k]) == i {
				found = true
			}
		}
		if !found {
			vlog("Removing project %s from the project list, because the project file was removed.\n",
				i)
			delete(p.data, i)
		}
	}
}

func (p *projectList) stopAll() {
	p.RLock()
	defer p.RUnlock()

	for i := range p.data {
		err := p.data[i].close()
		if err != nil {
			log.Printf("Error closing project datastore for Project: %s Error: %s\n", p.data[i].id(), err)
		}
	}
}

func (p *projectList) webList() ([]*webProject, error) {
	p.RLock()
	defer p.RUnlock()

	list := make([]*webProject, 0, len(p.data))

	for i := range p.data {
		prj, err := p.data[i].webData()
		if err != nil {
			return nil, err
		}

		list = append(list, prj)
	}

	return list, nil
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
		log.Printf("Error in startProjectLoader opening the filepath %s: %s\n", projectDir, err)
		return
	}

	files, err := dir.Readdir(0)
	if err != nil {
		log.Printf("Error in startProjectLoader reading the dir %s: %s\n", projectDir, err)
		return
	}

	names := make([]string, len(files))

	for i := range files {
		if !files[i].IsDir() && filepath.Ext(files[i].Name()) == ".json" {
			names[i] = files[i].Name()
			if _, ok := projects.get(projectID(files[i].Name())); !ok {
				projects.add(files[i].Name())
			}
		}
	}

	//check for removed projects
	projects.removeMissing(names)

	time.AfterFunc(projectFilePoll, startProjectLoader)
}
