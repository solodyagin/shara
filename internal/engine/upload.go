package engine

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
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

		var fileInfo *filestore.FileInfo

		switch s.cfg.GetString("storage") {
		case "local":
			// Сохраняем в локальное хранилище
			store, err := filestore.Open(s.cfg.GetString("local.endpoint"))
			if err != nil {
				response.SendError(c, http.StatusBadRequest, "Возникла ошибка при открытии локального файлового хранилища")
				return
			}
			fileInfo, err = store.Create("", file)
			if err != nil {
				response.SendError(c, http.StatusBadRequest, "Возникла ошибка при сохранении файла в локальное файловое хранилище")
				return
			}

		case "minio":
			// Сохраняем во временную директорию
			store, err := filestore.Open(s.cfg.GetString("temp_dir"))
			if err != nil {
				response.SendError(c, http.StatusBadRequest, "Возникла ошибка при открытии временного файлового хранилища")
				return
			}
			fileInfo, err = store.Create("", file)
			if err != nil {
				response.SendError(c, http.StatusBadRequest, "Возникла ошибка при сохранении файла во временное файловое хранилище")
				return
			}
			defer store.Remove("", fileInfo.Name)

			// Ищем запись в базе
			// rec, err := s.db.GetRecord(fileInfo.Name)
			// if err != nil && err != sql.ErrNoRows {
			// 	response.SendError(c, http.StatusInternalServerError, err.Error())
			// 	return
			// }

			// Если запись не нашли, то загружаем в MinIO
			// if rec == nil {

			// Создаём bucket
			ctx := context.Background()
			bucketName := s.cfg.GetString("minio.bucket_name")
			location := s.cfg.GetString("minio.location")
			if err := s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location, ObjectLocking: true}); err != nil {
				if exists, errBucketExists := s.client.BucketExists(ctx, bucketName); exists && errBucketExists == nil {
					log.Printf("We already own %s\n", bucketName)
				} else {
					response.SendError(c, http.StatusInternalServerError, err.Error())
					return
				}
			} else {
				log.Printf("Successfully created bucket \"%s\"\n", bucketName)
			}

			// Загружаем файл
			metadata := map[string]string{}
			metadata["origName"] = fileHeader.Filename

			if _, err := s.client.FPutObject(ctx, bucketName, fileInfo.Name, fileInfo.Location, minio.PutObjectOptions{
				UserMetadata: metadata,
				ContentType:  fileInfo.Mimetype,
			}); err != nil {
				response.SendError(c, http.StatusInternalServerError, err.Error())
				return
			}

			// }
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
