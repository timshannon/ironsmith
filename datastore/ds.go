// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package datastore manages the bolt data files and the reading and writing data to them
package datastore

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/boltdb/bolt"
)

//TODO: Move this all over to GobStore if I ever get around to finishing it

// ErrNotFound is the error returned when a value cannot be found in the store for the given key
var ErrNotFound = errors.New("Value not found in datastore")

// Store is a datastore for getting and setting data for a given ironsmith project
// run on top of a Bolt DB file
type Store struct {
	bolt *bolt.DB
}

// Open opens an existing datastore file, or creates a new one
// caller is responsible for closing the datastore
func Open(filename string) (*Store, error) {
	db, err := bolt.Open(filename, 0666, &bolt.Options{Timeout: 1 * time.Minute})

	if err != nil {
		return nil, err
	}

	store := &Store{
		bolt: db,
	}

	err = store.bolt.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketLog))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(bucketReleases))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(bucketFiles))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return store, nil
}

// Close closes the bolt datastore
func (ds *Store) Close() error {
	return ds.bolt.Close()
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

func (ds *Store) get(bucket string, key []byte, result interface{}) error {
	return ds.bolt.View(func(tx *bolt.Tx) error {
		dsValue := tx.Bucket([]byte(bucket)).Get(key)

		if dsValue == nil {
			return ErrNotFound
		}

		return json.Unmarshal(dsValue, result)
	})
}

func (ds *Store) put(bucket string, key []byte, value interface{}) error {
	var err error
	dsValue, ok := value.([]byte)
	if !ok {
		dsValue, err = json.Marshal(value)
		if err != nil {
			return err
		}
	}

	return ds.bolt.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucket)).Put(key, dsValue)
	})
}
