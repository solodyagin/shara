package sqlite

import (
	"database/sql"

	"shara/internal/models"
)

// InsertRecord
func (d *SqliteDB) InsertRecord(rec *models.Record) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO shara_files(hash_sum, orig_name, file_id, size) VALUES(:hash_sum, :orig_name, :file_id, :size) ON CONFLICT(hash_sum) DO NOTHING;`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec(
		sql.Named("hash_sum", rec.HashSum),
		sql.Named("orig_name", rec.OrigName),
		sql.Named("file_id", rec.FileId),
		sql.Named("size", rec.Size),
	); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
