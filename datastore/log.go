// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
)

// Log is a version log entry for a project
type Log struct {
	When    time.Time `json:"when,omitempty"`
	Version string    `json:"version,omitempty"`
	Stage   string    `json:"stage,omitempty"`
	Log     string    `json:"log,omitempty"`
}

const bucketLog = "log"

// AddLog adds a new log entry
func (ds *Store) AddLog(version, stage, entry string) error {
	key := NewTimeKey()

	data := &Log{
		When:    key.Time(),
		Version: version,
		Stage:   stage,
		Log:     entry,
	}

	return ds.put(bucketLog, key.Bytes(), data)
}

// LastVersion returns the last version in the log for the given stage.  If stage is blank,
// then it returns the last of any stage
func (ds *Store) LastVersion(stage string) (*Log, error) {
	last := &Log{}

	err := ds.bolt.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucketLog)).Cursor()

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			l := &Log{}
			err := json.Unmarshal(v, l)
			if err != nil {
				return err
			}

			if l.Version != "" {
				if stage == "" || l.Stage == stage {
					last = l
					return nil
				}
			}
		}

		return nil // not found return blank
	})

	if err != nil {
		return nil, err
	}

	return last, nil
}

// Versions lists the versions in a given project, including the last stage that version got to
func (ds *Store) Versions() ([]*Log, error) {
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

// VersionLog returns all the log entries for a given version
func (ds *Store) VersionLog(version string) ([]*Log, error) {
	var logs []*Log

	if version == "" {
		return logs, nil
	}

	verFound := false

	err := ds.bolt.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucketLog)).Cursor()

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			l := &Log{}
			err := json.Unmarshal(v, l)
			if err != nil {
				return err
			}

			if verFound && l.Version != version {
				return nil
			}

			if l.Version == version {
				logs = append(logs, l)
				verFound = true
			}

		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return logs, nil
}

// StageLog returns the log entry for a given version + stage
func (ds *Store) StageLog(version, stage string) (*Log, error) {
	var entry *Log

	if version == "" || stage == "" {
		return nil, ErrNotFound
	}

	err := ds.bolt.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(bucketLog)).Cursor()

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			l := &Log{}
			err := json.Unmarshal(v, l)
			if err != nil {
				return err
			}

			if l.Version == version && l.Stage == stage {
				entry = l
				return nil
			}

		}

		return ErrNotFound
	})

	if err != nil {
		return nil, err
	}

	return entry, nil
}
