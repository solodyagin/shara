package database

import "shara/internal/models"

type Database interface {
	GetRecord(name string) (*models.Record, error)
	CreateRecord(rec *models.Record) error
}
