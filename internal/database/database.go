package database

type Database interface {
	GetRecord(name string) (*Record, error)
	CreateRecord(rec *Record) error
}
