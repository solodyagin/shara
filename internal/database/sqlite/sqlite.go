package sqlite

import (
	"database/sql"
	"shara/internal/models"
)

type SqliteDB struct {
	*sql.DB
}

// New подключается к существующей БД или создаёт новую
func New(path string) (*SqliteDB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return &SqliteDB{db}, nil
}

// GetRecordByFileId
func (db *SqliteDB) GetRecordByFileId(fileId string) (*models.Record, error) {
	stmt, err := db.Prepare(`SELECT hash_sum, orig_name, file_id, size FROM files WHERE file_id = :file_id`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var r models.Record
	if err := stmt.QueryRow(sql.Named("file_id", fileId)).Scan(&r.HashSum, &r.OrigName, &r.FileId, &r.Size); err != nil {
		return nil, err
	}

	return &r, nil
}

// GetRecordByHashSum
func (db *SqliteDB) GetRecordByHashSum(hashSum string) (*models.Record, error) {
	stmt, err := db.Prepare(`SELECT hash_sum, orig_name, file_id, size FROM files WHERE hash_sum = :hash_sum`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var r models.Record
	if err := stmt.QueryRow(sql.Named("hash_sum", hashSum)).Scan(&r.HashSum, &r.OrigName, &r.FileId, &r.Size); err != nil {
		return nil, err
	}

	return &r, nil
}

// PutRecord
func (db *SqliteDB) PutRecord(r *models.Record) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO files(hash_sum, orig_name, file_id, size) VALUES(:hash_sum, :orig_name, :file_id, :size) ON CONFLICT(hash_sum) DO NOTHING;`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec(
		sql.Named("hash_sum", r.HashSum),
		sql.Named("orig_name", r.OrigName),
		sql.Named("file_id", r.FileId),
		sql.Named("size", r.Size),
	); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
