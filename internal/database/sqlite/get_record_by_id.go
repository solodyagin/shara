package sqlite

import (
	"database/sql"

	"shara/internal/models"
)

// GetRecordById
func (d *SqliteDB) GetRecordById(fileId string) (*models.Record, error) {
	stmt, err := d.Prepare(`SELECT hash_sum, orig_name, file_id, size FROM shara_files WHERE file_id = :file_id`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var rec models.Record
	if err := stmt.QueryRow(sql.Named("file_id", fileId)).Scan(&rec.HashSum, &rec.OrigName, &rec.FileId, &rec.Size); err != nil {
		return nil, err
	}

	return &rec, nil
}
