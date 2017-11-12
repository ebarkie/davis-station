// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package archive

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/ebarkie/weatherlink/data"

	"github.com/boltdb/bolt"
)

// Records stores a handle for the archive database.
type Records struct {
	db *bolt.DB
}

// Add adds an archive record to the database.
func (r Records) Add(a data.Archive) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("archive"))
		if err != nil {
			return err
		}

		encoded, _ := json.Marshal(a)
		return b.Put([]byte(a.Timestamp.In(time.UTC).Format(time.RFC3339)), encoded)
	})
}

// Last returns the timestamp of the most recent archive record in the database.
func (r Records) Last() (t time.Time) {
	r.db.View(func(tx *bolt.Tx) error {
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
func (r Records) Get(begin time.Time, end time.Time) (archive []data.Archive) {
	ac := r.NewGet(begin, end)
	for a := range ac {
		archive = append(archive, a)
	}

	return
}

// NewGet creates a channel and sends the requested range of archive records to it
// in descending order.
func (r Records) NewGet(begin time.Time, end time.Time) <-chan data.Archive {
	ac := make(chan data.Archive)

	go func() {
		defer close(ac)

		r.db.View(func(tx *bolt.Tx) error {
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

				var a data.Archive
				for k, v := c.Seek(max); k != nil && bytes.Compare(k, min) >= 0; k, v = c.Prev() {
					err := json.Unmarshal(v, &a)
					if err != nil {
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

// Open opens up the archive records database.
func Open(file string) (r Records, err error) {
	r.db, err = bolt.Open(file, 0600, nil)
	return
}

// Close closes the archive database.
func (r Records) Close() {
	r.db.Close()
}
