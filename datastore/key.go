// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

// TimeKey is a unique time ordered key for use in the datastore
// A TimeKey is 32 bits random data + 96 bit UNIX timestamp (64bits seconds + 32 bit nanoseconds)
type TimeKey [16]byte

// NewTimeKey returns a newly generated TimeKey based on the current time
func NewTimeKey() TimeKey {
	rBits := make([]byte, 32/8)
	_, err := io.ReadFull(rand.Reader, rBits)
	if err != nil {
		panic(fmt.Sprintf("Error generating random values for New TimeKey: %v", err))
	}

	t := time.Now()
	sec := t.Unix()
	nsec := t.Nanosecond()

	return TimeKey{
		rBits[0], //random
		rBits[1],
		rBits[2],
		rBits[3],
		byte(sec >> 56), // seconds
		byte(sec >> 48),
		byte(sec >> 40),
		byte(sec >> 32),
		byte(sec >> 24),
		byte(sec >> 16),
		byte(sec >> 8),
		byte(sec),
		byte(nsec >> 24), // nanoseconds
		byte(nsec >> 16),
		byte(nsec >> 8),
		byte(nsec),
	}
}

// Time returns the time portion of a timekey
func (k TimeKey) Time() time.Time {
	buf := k[4:]

	sec := int64(buf[7]) | int64(buf[6])<<8 | int64(buf[5])<<16 | int64(buf[4])<<24 |
		int64(buf[3])<<32 | int64(buf[2])<<40 | int64(buf[1])<<48 | int64(buf[0])<<56

	buf = buf[8:]
	nsec := int32(buf[3]) | int32(buf[2])<<8 | int32(buf[1])<<16 | int32(buf[0])<<24

	return time.Unix(sec, int64(nsec))
}

// UUID returns the string representation of a TimeKey in Hex format separated by dashes
// similar to a UUID xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func (k TimeKey) UUID() string {
	buf := make([]byte, 36)

	hex.Encode(buf[0:8], k[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], k[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], k[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], k[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], k[10:])

	return string(buf)
}

// Bytes returns the a slice of the underlying bytes of a TimeKey
func (k TimeKey) Bytes() []byte {
	return []byte(k[:])
}
