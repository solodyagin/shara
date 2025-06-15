package engine

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tenrok/filestore"

	"shara/internal/models"
	"shara/internal/response"
)

// HandleUpload
func (s *Server) HandleUpload() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ограничиваем размер загружаемого файла
		maxUploadSize := s.cfg.GetInt64("max_upload_size")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

		// Парсируем данные
		if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
			response.SendError(c, http.StatusBadRequest, "Возникла ошибка при разборе multipart сообщения")
			return
		}

		// Открываем файл
		file, fileHeader, err := c.Request.FormFile("file")
		if err != nil {
			response.SendError(c, http.StatusBadRequest, "Возникла ошибка при открытии файла")
			return
		}
		defer file.Close()

		// Сохраняем в локальное хранилище
		store, err := filestore.NewStore(s.cfg.GetString("local.endpoint"))
		if err != nil {
			response.SendError(c, http.StatusBadRequest, "Возникла ошибка при открытии локального файлового хранилища")
			return
		}
		fileInfo, err := store.Create(file)
		if err != nil {
			response.SendError(c, http.StatusBadRequest, "Возникла ошибка при сохранении файла в локальное файловое хранилище")
			return
		}
		defer func() {
			if s.cfg.GetString("storage") == "minio" {
				store.Remove(fileInfo.Name)
			}
		}()

		// Загружаем в MinIO
		if s.cfg.GetString("storage") == "minio" {
			remoteFile, err := s.fs.RemoteStorage().Create(fileInfo.Name)
			if err != nil {
				response.SendError(c, http.StatusInternalServerError, err.Error())
				return
			}
			defer remoteFile.Close()

			bufferReader := bufio.NewReaderSize(file, 4<<10)
			if _, err := bufferReader.WriteTo(remoteFile); err != nil {
				response.SendError(c, http.StatusInternalServerError, err.Error())
				return
			}
		}

		// Записываем в БД
		rec := &models.Record{
			Name:     fileInfo.Name,
			OrigName: fileHeader.Filename,
			Size:     fileInfo.Size,
		}
		if err := s.db.CreateRecord(rec); err != nil {
			response.SendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		response.SendSuccess(c, fmt.Sprintf("Successfully uploaded \"%s\" of size %d", rec.OrigName, rec.Size), response.Result{
			Name:     rec.Name,
			OrigName: rec.OrigName,
			Size:     rec.Size,
		})
	}
}
