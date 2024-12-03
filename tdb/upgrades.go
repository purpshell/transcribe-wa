package tdb

import (
	"database/sql"
)

type upgradeFunc func(*sql.Tx) error

var Upgrades = [...]upgradeFunc{upgradeV0}

func (t *TranscriptionDB) Upgrade() error {
	version, err := t.getVersion()
	if err != nil {
		return err
	}

	for ; version < len(Upgrades); version++ {
		var tx *sql.Tx
		tx, err = t.db.Begin()
		if err != nil {
			return err
		}

		migrateFunc := Upgrades[version]
		t.log.Infof("Upgrading database to v%d", version+1)
		err = migrateFunc(tx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		if err = t.setVersion(tx, version+1); err != nil {
			return err
		}

		if err = tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func (db *TranscriptionDB) getVersion() (int, error) {
	_, err := db.db.Exec("CREATE TABLE IF NOT EXISTS transcribe_version (version INTEGER)")
	if err != nil {
		return -1, err
	}

	version := 0
	row := db.db.QueryRow("SELECT version FROM transcribe_version LIMIT 1")
	if row != nil {
		_ = row.Scan(&version)
	}
	return version, nil
}

func (db *TranscriptionDB) setVersion(tx *sql.Tx, version int) error {
	_, err := tx.Exec("DELETE FROM transcribe_version")
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO transcribe_version (version) VALUES ($1)", version)
	return err
}

func upgradeV0(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS transcribe_chats (
		chat_id TEXT PRIMARY KEY,
		enabled BOOLEAN NOT NULL
	)`)
	return err
}
