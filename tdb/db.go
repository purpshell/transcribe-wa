package tdb

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type TranscriptionDB struct {
	db  *sql.DB
	log waLog.Logger
}

func NewTranscriptionDB(dialect string, address string, log waLog.Logger) (*TranscriptionDB, error) {
	db, err := sql.Open(dialect, address)

	if err != nil {
		return nil, err
	}

	return &TranscriptionDB{
		db:  db,
		log: log,
	}, nil
}
