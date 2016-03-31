// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import "time"

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
