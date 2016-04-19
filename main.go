// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"git.townsourced.com/townsourced/config"
)

//settings
var (
	projectDir = "./projects" // /etc/ironsmith/
	dataDir    = "./data"     // /var/ironsmith/
	address    = ":8026"
	certFile   = ""
	keyFile    = ""
)

//flags
var (
	verbose = false
)

func init() {
	flag.BoolVar(&verbose, "v", false, "Verbose prints to stdOut every command and stage as it processes")

	//Capture program shutdown, to make sure everything shuts down nicely
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig == os.Interrupt {
				projects.stopAll()
				os.Exit(0)
			}
		}
	}()
}

func main() {
	flag.Parse()

	settingPaths := config.StandardFileLocations("ironsmith/settings.json")
	vlog("IronSmith will use settings files in the following locations (in order of priority):\n")
	for i := range settingPaths {
		vlog("\t%s\n", settingPaths[i])
	}
	cfg, err := config.LoadOrCreate(settingPaths...)
	if err != nil {
		log.Fatalf("Error loading or creating IronSmith settings file: %s", err)
	}

	vlog("IronSmith is currently using the file %s for settings.\n", cfg.FileName())

	projectDir = cfg.String("projectDir", projectDir)
	dataDir = cfg.String("dataDir", dataDir)
	address = cfg.String("address", address)
	certFile = cfg.String("certFile", certFile)
	keyFile = cfg.String("keyFile", keyFile)

	vlog("Project Definition Directory: %s\n", projectDir)
	vlog("Project Data Directory: %s\n", dataDir)

	//prep dirs
	err = os.MkdirAll(filepath.Join(projectDir, enabledProjectDir), 0777)
	if err != nil {
		log.Fatalf("Error Creating project directory at %s: %s", projectDir, err)
	}

	err = os.MkdirAll(dataDir, 0777)
	if err != nil {
		log.Fatalf("Error Creating project data directory at %s: %s", dataDir, err)
	}

	err = prepTemplateProject()
	if err != nil {
		log.Fatalf("Error Creating project template file: %s", err)
	}

	//load projects
	err = projects.load()
	if err != nil {
		log.Fatalf("Error loading projects: %s", err)
	}

	//start web server
	err = startServer()
	if err != nil {
		log.Fatalf("Error Starting web server: %s", err)
	}

}
