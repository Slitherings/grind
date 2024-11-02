package db

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

type TokenRecord struct {
	Address     string
	Name        string
	Symbol      string
	FirstSeen   time.Time
	LastUpdated time.Time
	Liquidity   float64
	HolderCount int
	IsScam      bool
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	err = createTables(db)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tokens (
			address TEXT PRIMARY KEY,
			name TEXT,
			symbol TEXT,
			first_seen DATETIME,
			last_updated DATETIME,
			liquidity REAL,
			holder_count INTEGER,
			is_scam BOOLEAN
		);
	`)
	return err
}

func (d *Database) AddToken(token TokenRecord) error {
	_, err := d.db.Exec(`
		INSERT INTO tokens 
		(address, name, symbol, first_seen, last_updated, liquidity, holder_count, is_scam)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, token.Address, token.Name, token.Symbol, token.FirstSeen, token.LastUpdated,
		token.Liquidity, token.HolderCount, token.IsScam)
	return err
}

func (d *Database) GetToken(address string) (*TokenRecord, error) {
	var token TokenRecord
	err := d.db.QueryRow(`
		SELECT address, name, symbol, first_seen, last_updated, liquidity, holder_count, is_scam
		FROM tokens WHERE address = ?
	`, address).Scan(&token.Address, &token.Name, &token.Symbol, &token.FirstSeen,
		&token.LastUpdated, &token.Liquidity, &token.HolderCount, &token.IsScam)
	if err != nil {
		return nil, err
	}
	return &token, nil
}
