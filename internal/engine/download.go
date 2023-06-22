package engine

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"

	"shara/internal/response"
)

// HandleDownload
func (s *Server) HandleDownload() gin.HandlerFunc {
	return func(c *gin.Context) {
		fileId := c.Param("fileId")
		if fileId == "" {
			response.SendError(c, http.StatusBadRequest, "Не указан идентификатор файла")
			return
		}

		rec, err := s.db.GetRecordById(fileId)
		if err != nil {
			response.SendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		// Создаём временный файл
		tempFile, err := os.CreateTemp(s.cfg.GetString("pathes.temp_dir"), "temp")
		if err != nil {
			response.SendError(c, http.StatusInternalServerError, "Возникла ошибка при создании временного файла")
			return
		}
		defer tempFile.Close()
		defer os.Remove(tempFile.Name())

		// Скачиваем из Minio во временный файл
		ctx := context.Background()
		bucketName := s.cfg.GetString("minio.bucket_name")
		if err := s.client.FGetObject(ctx, bucketName, rec.HashSum, tempFile.Name(), minio.GetObjectOptions{}); err != nil {
			response.SendError(c, http.StatusInternalServerError, "Возникла ошибка при получении файла")
			return
		}

		// Отдаём файл клиенту
		c.FileAttachment(tempFile.Name(), rec.OrigName)
	}
}
