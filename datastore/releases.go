// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
)

type release struct {
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

	r := &release{
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

func (ds *Store) Release(version string) {

}

// Releases lists all the releases in a given project
func (ds *Store) Releases() ([]*Log, error) {
	var vers []*Log

	err := ds.bolt.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucketLog)).Cursor()

		var current = ""

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			l := &Log{}
			err := json.Unmarshal(v, l)
			if err != nil {
				return err
			}

			// capture the newest entry for each version
			if l.Version != current {
				l.Log = "" // only care about date, ver and stage
				vers = append(vers, l)
				current = l.Version
			}

		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return vers, nil
}
