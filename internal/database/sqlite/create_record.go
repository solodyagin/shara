package sqlite

import (
	"database/sql"

	"shara/internal/models"
)

// CreateRecord создаёт запись в базе
func (d *SqliteDB) CreateRecord(rec *models.Record) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`insert into "files" ("name", "orig_name", "size") values (:name, :orig_name, :size) on conflict("name") do nothing;`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		sql.Named("name", rec.Name),
		sql.Named("orig_name", rec.OrigName),
		sql.Named("size", rec.Size),
	)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
