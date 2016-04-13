// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"git.townsourced.com/ironsmith/datastore"
)

const (
	acceptHTML = "text/html"
)

// Err404 is a standard 404 error response
var Err404 = errors.New("Resource not found")

func errHandled(err error, w http.ResponseWriter, r *http.Request) bool {
	if err == nil {
		return false
	}

	if err == datastore.ErrNotFound {
		four04(w, r)
		return true
	}

	var status, errMsg string

	errMsg = err.Error()

	switch err.(type) {

	case *Fail:
		status = statusFail
	case *http.ProtocolError, *json.SyntaxError, *json.UnmarshalTypeError:
		//Hardcoded external errors which can bubble up to the end users
		// without exposing internal server information, make them failures
		err = FailFromErr(err)
		status = statusFail

		errMsg = fmt.Sprintf("We had trouble parsing your input, please check your input and try again: %s", err)
	default:
		status = statusError
		log.Printf("An error has occurred from a web request: %s", errMsg)
		errMsg = "An internal server error has occurred"
	}

	if status == statusFail {
		respondJsendCode(w, &JSend{
			Status:  status,
			Message: errMsg,
			Data:    err.(*Fail).Data,
		}, err.(*Fail).HTTPStatus)
	} else {
		respondJsend(w, &JSend{
			Status:  status,
			Message: errMsg,
		})
	}

	return true
}

// four04 is a standard 404 response if request header accepts text/html
// they'll get a 404 page, otherwise a json response
func four04(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "application/json")

	response := &JSend{
		Status:  statusFail,
		Message: "Resource not found",
		Data:    r.URL.String(),
	}

	w.WriteHeader(http.StatusNotFound)

	result, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling 404 response: %s", err)
		return
	}

	_, err = w.Write(result)
	if err != nil {
		log.Printf("Error in four04: %s", err)
	}
}

// Fail is an error whose contents can be exposed to the client and is usually the result
// of incorrect client input
type Fail struct {
	Message    string      `json:"message,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	HTTPStatus int         `json:"-"` //gets set in the error response
}

func (f *Fail) Error() string {
	return f.Message
}

// NewFail creates a new failure, data is optional
func NewFail(message string, data ...interface{}) error {
	return &Fail{
		Message:    message,
		Data:       data,
		HTTPStatus: 0,
	}
}

// FailFromErr returns a new failure based on the passed in error, data is optional
// if passed in error is nil, then nil is returned
func FailFromErr(err error, data ...interface{}) error {
	if err == nil {
		return nil
	}
	return NewFail(err.Error(), data...)
}

// IsEqual tests whether an error is equal to another error / failure
func (f *Fail) IsEqual(err error) bool {
	if err == nil {
		return false
	}

	return err.Error() == f.Error()
}

// IsFail tests whether the passed in error is a failure
func IsFail(err error) bool {
	if err == nil {
		return false
	}
	switch err.(type) {
	case *Fail:
		return true
	default:
		return false
	}
}
