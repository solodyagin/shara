package database

import "shara/internal/models"

type Database interface {
	GetRecordById(fileId string) (*models.Record, error)
	GetRecordByHash(hashSum string) (*models.Record, error)
	InsertRecord(rec *models.Record) error
}
