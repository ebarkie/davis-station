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
func (ad ArchiveData) Add(a weatherlink.Archive) error {
	return ad.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("archive"))
		if err != nil {
			return err
		}

		encoded, _ := json.Marshal(a)
		return b.Put([]byte(a.Timestamp.In(time.UTC).Format(time.RFC3339)), encoded)
	})
}

// Last returns the timestamp of the most recent archive record in the database.
func (ad ArchiveData) Last() (t time.Time) {
	ad.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("archive"))
		if b != nil {
			c := b.Cursor()
			k, _ := c.Last()
			t, _ = time.Parse(time.RFC3339, string(k))
			t = t.Local()
		}
		return nil
	})

	return
}

// Get returns the requested range of archive records as a slice in descending
// order.
func (ad ArchiveData) Get(begin time.Time, end time.Time) (archive []weatherlink.Archive) {
	ac := ad.NewGet(begin, end)
	for a := range ac {
		archive = append(archive, a)
	}

	return
}

// NewGet creates a channel and sends the requested range of archive records to it
// in descending order.
func (ad ArchiveData) NewGet(begin time.Time, end time.Time) <-chan weatherlink.Archive {
	ac := make(chan weatherlink.Archive)

	go func() {
		defer close(ac)

		ad.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("archive"))
			if b != nil {
				c := b.Cursor()

				min := []byte(begin.In(time.UTC).Format(time.RFC3339))
				max := []byte(end.In(time.UTC).Format(time.RFC3339))

				// Find starting position
				if k, _ := c.Seek(max); k == nil {
					// If max is not found then use the last key
					max, _ = c.Last()
				} else if !bytes.Equal(k, max) {
					// If Seek() does not get an exact match it returns
					// the next key.  This goes beyond max so we really
					// want to start at the key before it.
					max, _ = c.Prev()
				}

				var a weatherlink.Archive
				for k, v := c.Seek(max); k != nil && bytes.Compare(k, min) >= 0; k, v = c.Prev() {
					err := json.Unmarshal(v, &a)
					if err != nil {
						Error.Printf("Unable to unmarshal archive record: %s", k)
						continue
					}
					ac <- a
				}
			}

			return nil
		})
	}()

	return ac
}

// OpenArchive opens up the archive database.
func OpenArchive(file string) (ad ArchiveData, err error) {
	ad.db, err = bolt.Open(file, 0600, nil)
	return
}

// Close closes the archive database.
func (ad ArchiveData) Close() {
	ad.db.Close()
}
