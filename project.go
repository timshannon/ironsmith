package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const enabledProjectDir = "enabled"

// Project is an ironsmith project that contains how to fetch, build, test, and release a project
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
