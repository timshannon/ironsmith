// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package datastore manages the bolt data files and the reading and writing data to them
package datastore

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/boltdb/bolt"
)

// ErrNotFound is the error returned when a value cannot be found in the store for the given key
var ErrNotFound = errors.New("Value not found")

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
	if ds != nil {
		return ds.Close()
	}

	return nil
}

func (ds *Store) get(bucket string, key TimeKey, result interface{}) error {
	return ds.bolt.View(func(tx *bolt.Tx) error {
		dsValue := tx.Bucket([]byte(bucket)).Get(key.Bytes())

		if dsValue == nil {
			return ErrNotFound
		}

		if value, ok := result.([]byte); ok {
			buff := bytes.NewBuffer(value)
			_, err := io.Copy(buff, bytes.NewReader(dsValue))
			if err != nil {
				return err
			}
			return nil
		}

		return json.Unmarshal(dsValue, result)
	})
}

func (ds *Store) put(bucket string, key TimeKey, value interface{}) error {
	var err error
	dsValue, ok := value.([]byte)
	if !ok {
		dsValue, err = json.Marshal(value)
		if err != nil {
			return err
		}
	}

	return ds.bolt.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucket)).Put(key.Bytes(), dsValue)
	})
}

func (ds *Store) delete(bucket string, key TimeKey) error {
	return ds.bolt.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucket)).Delete(key.Bytes())
	})
}
