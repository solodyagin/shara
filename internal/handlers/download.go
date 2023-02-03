package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"shara/internal/models"
	"shara/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// HandleDownload
func HandleDownload(p models.IProgram) gin.HandlerFunc {
	cfg := p.GetConfig()
	db := p.GetDB()
	client := p.GetClient()

	return func(c *gin.Context) {
		fileId := c.Param("fileId")
		if fileId == "" {
			response.SendErrorStatus(c, http.StatusBadRequest, "Не указан идентификатор файла")
			return
		}

		stmt, err := db.Prepare(`SELECT hash_sum FROM files WHERE file_id = :file_id`)
		if err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, "Возникла ошибка при поиске файла")
			return
		}
		defer stmt.Close()

		var hashSum string
		if err := stmt.QueryRow(sql.Named("file_id", fileId)).Scan(&hashSum); err != nil && err != sql.ErrNoRows {
			response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
			return
		}

		// Создаём временный файл
		tempFile, err := os.CreateTemp(cfg.GetString("temp_dir"), "temp")
		if err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, "Возникла ошибка при создании временного файла")
			return
		}
		tempFileName := tempFile.Name()
		tempFile.Close()
		defer os.Remove(tempFileName)

		// Скачиваем из Minio во временный файл
		ctx := context.Background()
		bucketName := cfg.GetString("minio.bucket_name")
		if err := client.FGetObject(ctx, bucketName, hashSum, tempFileName, minio.GetObjectOptions{}); err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, "Возникла ошибка при получении файла")
			return
		}

		// Отдаём файл клиенту
		c.File(tempFileName)
	}
}
