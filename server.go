// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"
	"path"
	"time"
)

var webRoot *http.ServeMux

func startServer() error {
	var err error

	routes()

	server := &http.Server{
		Handler: webRoot,
		Addr:    address,
	}

	if certFile == "" || keyFile == "" {
		err = server.ListenAndServe()
	} else {
		server.Addr = address
		err = server.ListenAndServeTLS(certFile, keyFile)
	}

	if err != nil {
		return err
	}

	return nil
}

type methodHandler struct {
	get    http.HandlerFunc
	post   http.HandlerFunc
	put    http.HandlerFunc
	delete http.HandlerFunc
}

func (m *methodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.get == nil {
		m.get = http.NotFound
	}
	if m.post == nil {
		m.post = http.NotFound
	}
	if m.put == nil {
		m.put = http.NotFound
	}
	if m.delete == nil {
		m.delete = http.NotFound
	}
	switch r.Method {
	case "GET":
		m.get(w, r)
		return
	case "POST":
		m.post(w, r)
		return
	case "PUT":
		m.put(w, r)
		return
	case "DELETE":
		m.delete(w, r)
		return
	default:
		http.NotFound(w, r)
		return
	}
}

/*
log Routes
	/log/<project-id>/<version>/<stage>

	/log/ - list all projects
	/log/<project-id> - list all versions in a project, triggers new builds
	/log/<project-id>/<version> - list combined output of all stages for a given version
	/log/<project-id>/<version>/<stage> - list output of a given stage of a given version

release routes
	/release/<project-id>/<version>

	/release/<project-id> - list last release for a given project  ?all returns all the releases for a project
	/release/<project-id>/<version> - list release for a given project version

trigger routes
	/trigger/<project-id>
		Triggers a project to start a cycle
*/

func routes() {
	webRoot = http.NewServeMux()

	webRoot.Handle("/", &methodHandler{
		get: rootGet,
	})

	webRoot.Handle("/log/", &methodHandler{
		get: logGet,
	})

	webRoot.Handle("/release/", &methodHandler{
		get: releaseGet,
	})

	webRoot.Handle("/trigger/", &methodHandler{
		post: triggerPost,
	})

}

func rootGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		//send index.html
		serveAsset(w, r, "web/index.html")
		return
	}

	serveAsset(w, r, path.Join("web", r.URL.Path))
}

func serveAsset(w http.ResponseWriter, r *http.Request, asset string) {
	data, err := Asset(asset)

	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeContent(w, r, r.URL.Path, time.Time{}, bytes.NewReader(data))
}
