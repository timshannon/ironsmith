// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/boltdb/bolt"
)

// Release is a record of the fully built and ready to deploy release file
type Release struct {
	When     time.Time `json:"when"`
	Version  string    `json:"version"`
	FileName string    `json:"fileName"`
	FileKey  TimeKey   `json:"fileKey"`
}

const (
	bucketReleases = "releases"
	bucketFiles    = "files"
)

// AddRelease adds a new Release
func (ds *Store) AddRelease(version, fileName string, fileData []byte) error {
	fileKey := NewTimeKey()

	r := &Release{
		When:     fileKey.Time(),
		Version:  version,
		FileName: fileName,
		FileKey:  fileKey,
	}

	dsValue, err := json.Marshal(r)
	if err != nil {
		return err
	}

	return ds.bolt.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(bucketReleases)).Put([]byte(version), dsValue)
		if err != nil {
			return err
		}

		return tx.Bucket([]byte(bucketFiles)).Put(fileKey.Bytes(), fileData)
	})
}

// ReleaseFile returns a specific file from a release for the given file key
func (ds *Store) ReleaseFile(fileKey TimeKey) ([]byte, error) {
	var fileData bytes.Buffer

	err := ds.bolt.View(func(tx *bolt.Tx) error {
		dsValue := tx.Bucket([]byte(bucketFiles)).Get(fileKey.Bytes())

		if dsValue == nil {
			return ErrNotFound
		}

		_, err := io.Copy(&fileData, bytes.NewReader(dsValue))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return fileData.Bytes(), nil
}

// Release gets the release record for a specific version
func (ds *Store) Release(version string) (*Release, error) {
	r := &Release{}
	err := ds.get(bucketReleases, []byte(version), r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Releases lists all the releases in a given project
func (ds *Store) Releases() ([]*Release, error) {
	var vers []*Release

	err := ds.bolt.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucketReleases)).Cursor()

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			r := &Release{}
			err := json.Unmarshal(v, r)
			if err != nil {
				return err
			}

			vers = append(vers, r)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return vers, nil
}

// LastRelease lists the last release for a project
func (ds *Store) LastRelease() (*Release, error) {
	r := &Release{}

	err := ds.bolt.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucketReleases)).Cursor()

		_, v := c.Last()
		if v == nil {
			return ErrNotFound
		}

		err := json.Unmarshal(v, r)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}
