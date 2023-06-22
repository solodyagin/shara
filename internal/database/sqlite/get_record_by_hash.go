package sqlite

import (
	"database/sql"

	"shara/internal/models"
)

// GetRecordByHash
func (d *SqliteDB) GetRecordByHash(hashSum string) (*models.Record, error) {
	stmt, err := d.Prepare(`SELECT hash_sum, orig_name, file_id, size FROM shara_files WHERE hash_sum = :hash_sum`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var rec models.Record
	if err := stmt.QueryRow(sql.Named("hash_sum", hashSum)).Scan(&rec.HashSum, &rec.OrigName, &rec.FileId, &rec.Size); err != nil {
		return nil, err
	}

	return &rec, nil
}
