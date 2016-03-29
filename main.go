package main

import (
	"log"
	"os"
	"path/filepath"

	"git.townsourced.com/config"
)

//settings
var (
	projectDir = "./projects" // /etc/
	dataDir    = "./data"
	address    = "http://localhost:8026"
	certFile   = ""
	keyFile    = ""
)

func main() {
	settingPaths := config.StandardFileLocations("ironsmith/settings.json")
	log.Println("IronSmith will use settings files in the following locations (in order of priority):")
	for i := range settingPaths {
		log.Println("\t", settingPaths[i])
	}
	cfg, err := config.LoadOrCreate(settingPaths...)
	if err != nil {
		log.Fatalf("Error loading or creating IronSmith settings file: %s", err)
	}

	log.Printf("IronSmith is currently using the file %s for settings.\n", cfg.FileName())

	projectDir = cfg.String("projectDir", projectDir)
	dataDir = cfg.String("dataDir", dataDir)
	address = cfg.String("address", address)
	certFile = cfg.String("certFile", certFile)
	keyFile = cfg.String("keyFile", keyFile)

	//prep dirs
	err = os.MkdirAll(filepath.Join(projectDir, enabledProjectDir), os.ModeDir)
	if err != nil {
		log.Fatalf("Error Creating project directory at %s: %s", projectDir, err)
	}

	err = os.MkdirAll(dataDir, os.ModeDir)
	if err != nil {
		log.Fatalf("Error Creating project data directory at %s: %s", dataDir, err)
	}

	err = prepTemplateProject()
	if err != nil {
		log.Fatalf("Error Creating project template file: %s", err)
	}

	//load projects
	//start server
}
