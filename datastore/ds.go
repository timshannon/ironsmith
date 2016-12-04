// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package datastore manages the bolt data files and the reading and writing data to them
package datastore

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
	"github.com/timshannon/bolthold"
)

// Store is a datastore for getting and setting data for a given ironsmith project
// run on top of a Bolt DB file
type Store struct {
	store *bolthold.Store
}

// Open opens an existing datastore file, or creates a new one
// caller is responsible for closing the datastore
func Open(filename string) (*Store, error) {
	db, err := bolthold.Open(filename, 0666, &bolthold.Options{Options: &bolt.Options{Timeout: 1 * time.Minute}})

	if err != nil {
		return nil, err
	}

	store := &Store{
		store: db,
	}

	return store, nil
}

// Close closes the bolt datastore
func (ds *Store) Close() error {
	return ds.store.Close()
}

// TrimVersions Removes versions from the datastore file until it reaches the maxVersions count
func (ds *Store) TrimVersions(maxVersions int) error {
	if maxVersions <= 0 {
		// no max set
		return nil
	}

	versions, err := ds.Versions()
	if err != nil {
		return err
	}

	if len(versions) <= maxVersions {
		return nil
	}

	remove := versions[maxVersions:]

	for i := range remove {
		err = ds.deleteVersion(remove[i].Version)
		if err != nil {
			return err
		}
	}

	return nil
}

// removes the earliest instance of a specific version
func (ds *Store) deleteVersion(version string) error {
	return ds.bolt.Update(func(tx *bolt.Tx) error {
		// remove all logs for this version
		c := tx.Bucket([]byte(bucketLog)).Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			lg := &Log{}

			err := json.Unmarshal(v, lg)
			if err != nil {
				return err
			}

			if lg.Version != version {
				break
			}

			err = c.Delete()
			if err != nil {
				return err
			}
		}

		// remove all releases for this version
		release, err := ds.Release(version)
		if err == ErrNotFound {
			return nil
		}
		if err != nil {
			return err
		}

		err = tx.Bucket([]byte(bucketReleases)).Delete(release.FileKey.Bytes())
		if err != nil {
			return err
		}

		// remove release file for this version
		err = tx.Bucket([]byte(bucketFiles)).Delete(release.FileKey.Bytes())
		if err != nil {
			return err
		}

		return nil
	})

}
