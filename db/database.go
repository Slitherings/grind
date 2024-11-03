package db

import "grind/types"

type SQLiteDB struct {
	StorePair(pair RaydiumPair) error
}

func NewDatabase(path string) (*SQLiteDB, error) {
	return &SQLiteDB{
		path: path,
	}, nil
}

func (d *Database) Close() error {
	// Implement any cleanup needed
	return nil
}
