package handlers

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

	"shara/internal/models"
	"shara/internal/response"
	"shara/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/minio/minio-go/v7"
)

// HandleUpload
func HandleUpload(p models.IProgram) gin.HandlerFunc {
	cfg := p.GetConfig()
	db := p.GetDB()
	client := p.GetClient()

	return func(c *gin.Context) {
		// Ограничиваем размер загружаемого файла
		maxUploadSize := cfg.GetInt64("max_upload_size")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

		// Парсируем данные
		if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
			response.SendErrorStatus(c, http.StatusBadRequest, "Возникла ошибка при разборе multipart сообщения")
			return
		}

		// Открываем файл
		formFile, formFileHeader, err := c.Request.FormFile("file")
		if err != nil {
			response.SendErrorStatus(c, http.StatusBadRequest, "Возникла ошибка при открытии файла")
			return
		}
		defer formFile.Close()

		// Проверяем тип файла
		b := make([]byte, utils.Min(512, formFileHeader.Size))
		formFile.Read(b)
		formFileType := http.DetectContentType(b)

		// Подсчитываем контрольную сумму
		hash := sha256.New()
		if _, err := io.Copy(hash, formFile); err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, "Возникла ошибка при подсчёте контрольной суммы файла")
			return
		}
		formFileHash := fmt.Sprintf("%x", hash.Sum(nil))
		formFile.Close()

		// Ищем в БД запись с такой контрольной суммой
		stmt, err := db.Prepare(`SELECT hash_sum, orig_name, file_id, size FROM files WHERE hash_sum = :hash`)
		if err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
			return
		}
		defer stmt.Close()

		var hashSum string
		var origName string
		var fileId string
		var size int64
		if err := stmt.QueryRow(sql.Named("hash", formFileHash)).Scan(&hashSum, &origName, &fileId, &size); err != nil && err != sql.ErrNoRows {
			response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
			return
		}
		if hashSum != "" {
			response.SendSuccess(c, "The file already exists", response.Result{
				OrigName: origName,
				FileId:   fileId,
				Size:     size,
			})
			return
		}

		// Создаём временный файл
		tempFile, err := os.CreateTemp(cfg.GetString("pathes.temp_dir"), "temp")
		if err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, "Возникла ошибка при создании временного файла")
			return
		}
		defer tempFile.Close()
		defer os.Remove(tempFile.Name())

		// Сохраняем во временный файл
		if err := c.SaveUploadedFile(formFileHeader, tempFile.Name()); err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, "Возникла ошибка при сохранении во временный файл")
			return
		}

		// Создаём bucket
		ctx := context.Background()
		bucketName := cfg.GetString("minio.bucket_name")
		location := cfg.GetString("minio.location")
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location, ObjectLocking: true}); err != nil {
			if exists, errBucketExists := client.BucketExists(ctx, bucketName); exists && errBucketExists == nil {
				log.Printf("We already own %s\n", bucketName)
			} else {
				response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			log.Printf("Successfully created bucket \"%s\"\n", bucketName)
		}

		// Загружаем файл
		metadata := map[string]string{}
		metadata["origName"] = formFileHeader.Filename

		info, err := client.FPutObject(ctx, bucketName, formFileHash, tempFile.Name(), minio.PutObjectOptions{
			UserMetadata: metadata,
			ContentType:  formFileType,
		})
		if err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
			return
		}

		u, _ := uuid.NewV4()
		fileId = hex.EncodeToString(u.Bytes()) + utils.RandomString(16)

		// Записываем в БД
		tx, err := db.Begin()
		if err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
			return
		}
		stmt2, err := tx.Prepare(`INSERT INTO files(hash_sum, orig_name, file_id, size) VALUES(:hash_sum, :orig_name, :file_id, :size) ON CONFLICT(hash_sum) DO NOTHING;`)
		if err != nil {
			tx.Rollback()
			response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
			return
		}
		defer stmt2.Close()

		if _, err := stmt2.Exec(
			sql.Named("hash_sum", formFileHash),
			sql.Named("orig_name", formFileHeader.Filename),
			sql.Named("file_id", fileId),
			sql.Named("size", info.Size),
		); err != nil {
			tx.Rollback()
			response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
			return
		}

		if err := tx.Commit(); err != nil {
			response.SendErrorStatus(c, http.StatusInternalServerError, err.Error())
			return
		}

		response.SendSuccess(c, fmt.Sprintf("Successfully uploaded \"%s\" of size %d", formFileHeader.Filename, info.Size), response.Result{
			OrigName: formFileHeader.Filename,
			FileId:   fileId,
			Size:     info.Size,
		})
	}
}
