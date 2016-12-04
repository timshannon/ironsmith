// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
	"github.com/timshannon/bolthold"
)

// Log is a version log entry for a project
type Log struct {
	When    time.Time `json:"when,omitempty"`
	Version string    `json:"version,omitempty"`
	Stage   string    `json:"stage,omitempty"`
	Log     string    `json:"log,omitempty"`
}

// AddLog adds a new log entry
func (ds *Store) AddLog(version, stage, entry string) error {
	key := NewTimeKey()

	data := &Log{
		When:    key.Time(),
		Version: version,
		Stage:   stage,
		Log:     entry,
	}

	return ds.store.Insert(key, data)
}

// LastVersion returns the last version in the log for the given stage.  If stage is blank,
// then it returns the last of any stage
func (ds *Store) LastVersion(stage string) (*Log, error) {
	var last []Log

	if stage != "" {
		err := ds.store.Find(&last, bolthold.Where("Stage").Eq(stage).And("Version").Ne("").Limit(1))
		if err != nil {
			return nil, err
		}

	} else {
		err := ds.store.Find(&last, bolthold.Where("Version").Ne("").Limit(1))
		if err != nil {
			return nil, err
		}
	}

	if len(last) == 0 {
		return nil, nil
	}

	return &last[0], nil
}

// Versions lists the versions in a given project, including the last stage that version got to
func (ds *Store) Versions() ([]Log, error) {
	var vers []Log

	//TODO: Replace with bolthold aggregates

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
func (ds *Store) VersionLog(version string) ([]Log, error) {
	var logs []Log

	if version == "" {
		return logs, nil
	}

	err := ds.store.Find(&logs, bolthold.Where("Version").Eq(version))

	if err != nil {
		return nil, err
	}

	return logs, nil
}

// StageLog returns the log entry for a given version + stage
func (ds *Store) StageLog(version, stage string) (*Log, error) {
	var entries []Log

	if version == "" || stage == "" {
		return nil, bolthold.ErrNotFound
	}

	err := bolthold.Find(&entries, bolthold.Where("Version").Eq(version).And("Stage").Eq(stage).Limit(1))

	if err != nil {
		return nil, err
	}

	if len(entries) < 1 {
		return nil, bolthold.ErrNotFound
	}

	return &entries[0], nil
}
