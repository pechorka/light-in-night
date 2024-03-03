package db

import (
	"encoding/json"
	"errors"
	"slices"

	bolt "go.etcd.io/bbolt"
)

type DB struct {
	db *bolt.DB
}

func New(db *bolt.DB) *DB {
	return &DB{db: db}
}

var (
	bktHighscores = []byte("highscores")
)

var (
	highscoreList = []byte("highscoreList")
)

type Highscore struct {
	Name    string
	Time    float32
	Score   int
	Victory bool
}

var ErrNotFound = errors.New("not found")

func (db *DB) AddHighscore(score Highscore) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists(bktHighscores)
		if err != nil {
			return err
		}

		list, err := readFromBucket[[]Highscore](bkt, highscoreList)
		if err != nil && err != ErrNotFound {
			return err
		}

		list = append(list, score)
		slices.SortFunc(list, func(e1, e2 Highscore) int {
			return e2.Score - e1.Score
		})

		err = putToBucket(bkt, highscoreList, list)
		if err != nil {
			return err
		}

		return nil
	})
}

func (db *DB) GetHighscores() ([]Highscore, error) {
	var list []Highscore
	err := db.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktHighscores)
		if bkt == nil {
			return ErrNotFound
		}

		var err error
		list, err = readFromBucket[[]Highscore](bkt, highscoreList)
		if err != nil {
			return err
		}

		return nil
	})

	return list, err
}

func putToBucket(bkt *bolt.Bucket, key []byte, value any) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return bkt.Put(key, bytes)
}

func readFromBucket[T any](bkt *bolt.Bucket, key []byte) (T, error) {
	var val T
	bytes := bkt.Get(key)
	if bytes == nil {
		return val, ErrNotFound
	}

	err := json.Unmarshal(bytes, &val)
	if err != nil {
		return val, err
	}

	return val, nil
}
