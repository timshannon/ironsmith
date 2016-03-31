// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import (
	"sort"
	"testing"
	"time"
)

func TestNewTimeKey(t *testing.T) {
	tk := NewTimeKey()

	if len(tk) != (128 / 8) {
		t.Errorf("Invalid Time Key Length want %d got %d", 128/8, len(tk))
	}
}

func TestTimeKeyTime(t *testing.T) {

	now := time.Now()

	tk := NewTimeKey()

	tkTime := tk.Time()

	if !tkTime.After(now) && !tkTime.Equal(now) {
		t.Errorf("TimeKey's time is not after or equal a previous generated timestamp. want: %s, got: %s", now, tkTime)
	}

	cTime := tk.Time()

	if !tkTime.Equal(cTime) {
		t.Errorf("TimeKey's time is not consistently parsed from the timekey.  want: %s, got: %s", tkTime, cTime)
	}
}

type ByTime []TimeKey

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Time().Before(a[j].Time()) }

func TestTimeKeyOrder(t *testing.T) {
	keys := make([]TimeKey, 1000)
	for i := range keys {
		keys[i] = NewTimeKey()
	}

	if !sort.IsSorted(ByTime(keys)) {
		t.Errorf("TimeKey's are not properly sorted by time")
	}

}
