// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import "time"

type release struct {
	When    time.Time `json:"when"`
	Version string    `json:"version"`
	FileKey TimeKey   `json:"file"`
}

const (
	bucketReleases = "releases"
	bucketFiles    = "files"
)

// AddRelease adds a new Release
func (ds *Store) AddRelease(version string, fileData []byte) error {
	key := NewTimeKey()
	fileKey := NewTimeKey()

	r := &release{
		When:    key.Time(),
		Version: version,
		FileKey: fileKey,
	}

	err := ds.put(bucketReleases, key, r)
	if err != nil {
		return err
	}

	err = ds.put(bucketFiles, fileKey, fileData)
	if err != nil {
		return err
	}

	return nil
}
