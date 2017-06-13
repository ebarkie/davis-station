// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

// Weather station data archive storage.

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

			var a weatherlink.Archive
			for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
				json.Unmarshal(v, &a)
				// We want the most recent data first so it would make sense
				// to prepend here but it's MUCH faster to append and then
				// reverse.
				archive = append(archive, a)
			}
		}

		return nil
	})

	// Reverse archive record slice so most recent data is first.
	for i := 0; i < len(archive)/2; i++ {
		j := len(archive) - 1 - i
		archive[i], archive[j] = archive[j], archive[i]
	}

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
