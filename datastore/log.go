// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
)

type log struct {
	When    time.Time `json:"when"`
	Version string    `json:"version"`
	Stage   string    `json:"stage"`
	Log     string    `json:"log"`
}

const bucketLog = "log"

// AddLog adds a new log entry
func (ds *Store) AddLog(version, stage, entry string) error {
	key := NewTimeKey()

	data := &log{
		When:    key.Time(),
		Version: version,
		Stage:   stage,
		Log:     entry,
	}

	return ds.put(bucketLog, key, data)
}

// LatestVersion returns the latest version (successful or otherwise) for the current project
func (ds *Store) LatestVersion() (string, error) {
	version := ""

	err := ds.bolt.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucketLog)).Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			l := &log{}
			err := json.Unmarshal(v, l)
			if err != nil {
				return err
			}

			if l.Version != "" {
				version = l.Version
				return nil
			}
		}

		return ErrNotFound
	})

	if err != nil {
		return "", err
	}

	return version, nil
}
