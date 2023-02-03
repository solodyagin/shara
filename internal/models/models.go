package models

import (
	"database/sql"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
)

type IProgram interface {
	GetConfig() *viper.Viper
	GetDB() *sql.DB
	GetClient() *minio.Client
}
