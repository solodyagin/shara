package engine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/minio/minio-go/v7"

	"shara/internal/models"
	"shara/internal/response"
	"shara/internal/utils"
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

		// Проверяем тип файла
		b := make([]byte, utils.Min(512, fileHeader.Size))
		file.Read(b)
		contentType := http.DetectContentType(b)

		// Подсчитываем контрольную сумму
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			response.SendError(c, http.StatusInternalServerError, "Возникла ошибка при подсчёте контрольной суммы файла")
			return
		}
		hashSum := fmt.Sprintf("%x", hash.Sum(nil))

		// Ищем в БД запись с такой контрольной суммой
		r, err := s.db.GetRecordByHash(hashSum)
		if err != nil && err != sql.ErrNoRows {
			response.SendError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if r != nil {
			response.SendSuccess(c, "The file already exists", response.Result{
				OrigName: r.OrigName,
				FileId:   r.FileId,
				Size:     r.Size,
			})
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

		// Сохраняем во временный файл
		if err := c.SaveUploadedFile(fileHeader, tempFile.Name()); err != nil {
			response.SendError(c, http.StatusInternalServerError, "Возникла ошибка при сохранении во временный файл")
			return
		}

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

		info, err := s.client.FPutObject(ctx, bucketName, hashSum, tempFile.Name(), minio.PutObjectOptions{
			UserMetadata: metadata,
			ContentType:  contentType,
		})
		if err != nil {
			response.SendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		u, _ := uuid.NewV4()

		// Записываем в БД
		rec := &models.Record{
			HashSum:  hashSum,
			OrigName: fileHeader.Filename,
			FileId:   utils.RandomString(16) + hex.EncodeToString(u.Bytes()),
			Size:     info.Size,
		}

		if err := s.db.InsertRecord(rec); err != nil {
			response.SendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		response.SendSuccess(c, fmt.Sprintf("Successfully uploaded \"%s\" of size %d", rec.OrigName, rec.Size), response.Result{
			OrigName: rec.OrigName,
			FileId:   rec.FileId,
			Size:     rec.Size,
		})
	}
}
