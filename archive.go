// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/ebarkie/weatherlink"

	"github.com/boltdb/bolt"
)

// ArchiveData stores a handle for the archive database.
type ArchiveData struct {
	db *bolt.DB
}

// Add adds an archive record to the database.
func (ad ArchiveData) Add(a weatherlink.Archive) (err error) {
	err = ad.db.Update(func(tx *bolt.Tx) (err error) {
		b, err := tx.CreateBucketIfNotExists([]byte("archive"))
		if err != nil {
			return
		}

		encoded, _ := json.Marshal(a)
		err = b.Put([]byte(a.Timestamp.In(time.UTC).Format(time.RFC3339)), encoded)
		return
	})
	return
}

// Last returns the timestamp of the most recent archive record in the database.
func (ad ArchiveData) Last() (t time.Time) {
	ad.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("archive"))
		if b != nil {
			c := b.Cursor()
			k, _ := c.Last()
			t, _ = time.Parse(time.RFC3339, string(k))
			t = t.In(time.Now().Location())
		}
		return nil
	})

	return
}

// Get retrieves the archive records from the database for a specified period.
func (ad ArchiveData) Get(begin time.Time, end time.Time) (archive []weatherlink.Archive) {
	ad.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("archive"))
		if b != nil {
			c := b.Cursor()

			min := []byte(begin.In(time.UTC).Format(time.RFC3339))
			max := []byte(end.In(time.UTC).Format(time.RFC3339))

			// Find starting position.
			if k, _ := c.Seek(max); k == nil {
				// If max is not found then use the last key.
				max, _ = c.Last()
			} else if !bytes.Equal(k, max) {
				// If Seek() does not get an exact match it returns
				// the next key.  This goes beyond max so we really
				// want to start at the key before it.
				max, _ = c.Prev()
			}

			var a weatherlink.Archive
			for k, v := c.Seek(max); k != nil && bytes.Compare(k, min) >= 0; k, v = c.Prev() {
				json.Unmarshal(v, &a)
				archive = append(archive, a)
			}
		}

		return nil
	})

	return
}

// OpenArchive opens up the archive database.
func OpenArchive(filename string) (ad ArchiveData, err error) {
	ad.db, err = bolt.Open(filename, 0600, nil)
	return
}

// Close closes the archive database.
func (ad ArchiveData) Close() {
	ad.db.Close()
}
