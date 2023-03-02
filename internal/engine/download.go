package engine

import (
	"context"
	"net/http"
	"os"

	"shara/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// HandleDownload
func (e *Engine) HandleDownload() gin.HandlerFunc {
	return func(c *gin.Context) {
		fileId := c.Param("fileId")
		if fileId == "" {
			response.SendError(c, http.StatusBadRequest, "Не указан идентификатор файла")
			return
		}

		r, err := e.db.GetRecordByFileId(fileId)
		if err != nil {
			response.SendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		// Создаём временный файл
		tempFile, err := os.CreateTemp(e.config.GetString("pathes.temp_dir"), "temp")
		if err != nil {
			response.SendError(c, http.StatusInternalServerError, "Возникла ошибка при создании временного файла")
			return
		}
		defer tempFile.Close()
		defer os.Remove(tempFile.Name())

		// Скачиваем из Minio во временный файл
		ctx := context.Background()
		bucketName := e.config.GetString("minio.bucket_name")
		if err := e.client.FGetObject(ctx, bucketName, r.HashSum, tempFile.Name(), minio.GetObjectOptions{}); err != nil {
			response.SendError(c, http.StatusInternalServerError, "Возникла ошибка при получении файла")
			return
		}

		// Отдаём файл клиенту
		c.File(tempFile.Name())
	}
}
