package sqlite

import (
	"database/sql"

	"shara/internal/models"
)

// GetRecord получает запись из базы
func (d *SqliteDB) GetRecord(name string) (*models.Record, error) {
	stmt, err := d.Prepare(`select "name", "orig_name", "size" from "files" where "name" = :name`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var rec models.Record
	if err := stmt.QueryRow(sql.Named("name", name)).Scan(&rec.Name, &rec.OrigName, &rec.Size); err != nil {
		return nil, err
	}

	return &rec, nil
}
