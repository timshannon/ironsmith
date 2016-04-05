// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "net/http"

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

func routes() {
	webRoot = http.NewServeMux()

}
