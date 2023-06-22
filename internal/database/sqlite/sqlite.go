package sqlite

import "database/sql"

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
