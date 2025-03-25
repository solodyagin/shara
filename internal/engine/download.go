package engine

import (
	"context"
	"database/sql"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/tenrok/filestore"

	"shara/internal/response"
)

// HandleDownload
func (s *Server) HandleDownload() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			response.SendError(c, http.StatusBadRequest, "Не указано имя файла")
			return
		}

		// Ищем запись в базе
		rec, err := s.db.GetRecord(name)
		if err != nil && err != sql.ErrNoRows {
			response.SendError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if rec == nil {
			response.SendError(c, http.StatusNotFound, "Файл не найден")
			return
		}

		var fullPath string

		switch s.cfg.GetString("storage") {

		case "local":
			// Открываем локальное файловое хранилище
			store, err := filestore.Open(s.cfg.GetString("local.endpoint"))
			if err != nil {
				response.SendError(c, http.StatusBadRequest, "Возникла ошибка при открытии локального файлового хранилища")
				return
			}

			// Существует ли такой файл физически?
			if !store.IsExists("", name) {
				response.SendError(c, http.StatusNotFound, "Файл не найден")
				return
			}

			fullPath = store.GetFullName("", name)

		case "minio":
			// Создаём временный файл
			tempFile, err := os.CreateTemp(s.cfg.GetString("temp_dir"), "temp")
			if err != nil {
				response.SendError(c, http.StatusInternalServerError, "Возникла ошибка при создании временного файла")
				return
			}
			defer func() {
				tempFile.Close()
				os.Remove(tempFile.Name())
			}()

			// Скачиваем из MinIO во временный файл
			bucketName := s.cfg.GetString("minio.bucket_name")
			if err := s.client.FGetObject(context.Background(), bucketName, rec.Name, tempFile.Name(), minio.GetObjectOptions{}); err != nil {
				response.SendError(c, http.StatusInternalServerError, "Возникла ошибка при получении файла")
				return
			}

			fullPath = tempFile.Name()
		}

		// Отдаём файл клиенту
		c.FileAttachment(fullPath, rec.OrigName)
	}
}
