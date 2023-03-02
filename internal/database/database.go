package database

import "shara/internal/models"

type Database interface {
	GetRecordByFileId(fileId string) (*models.Record, error)
	GetRecordByHashSum(hashSum string) (*models.Record, error)
	PutRecord(r *models.Record) error
}
