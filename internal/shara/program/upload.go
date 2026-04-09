package program

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tenrok/filestore/remote"

	"application/internal/shara/response"
)

// HandleUpload
func (p *Program) HandleUpload() gin.HandlerFunc {
	return func(gc *gin.Context) {
		// Ограничиваем размер загружаемого файла
		maxUploadSize := p.cfg.GetInt64("maxUploadSize")
		gc.Request.Body = http.MaxBytesReader(gc.Writer, gc.Request.Body, maxUploadSize)

		// Парсируем данные
		if err := gc.Request.ParseMultipartForm(maxUploadSize); err != nil {
			response.SendError(gc, http.StatusBadRequest, "Возникла ошибка при разборе multipart сообщения")
			return
		}

		// Открываем файл
		file, fileHeader, err := gc.Request.FormFile("file")
		if err != nil {
			response.SendError(gc, http.StatusBadRequest, "Возникла ошибка при открытии файла")
			return
		}
		defer file.Close()

		// Сохраняем в локальное хранилище
		localStorage := p.httpFS.LocalStorage()
		fileInfo, err := localStorage.Create(gc.Request.Context(), file)
		if err != nil {
			response.SendError(gc, http.StatusBadRequest, "Возникла ошибка при сохранении файла в локальное файловое хранилище")
			return
		}

		// Загружаем в MinIO
		if p.cfg.GetString("storage") == "minio" {
			// Перемещаемся на начало файла
			if _, err = file.Seek(0, io.SeekStart); err != nil {
				gc.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			remoteStorage := p.httpFS.RemoteStorage()
			if remoteStorage != nil {
				uploader := remoteStorage.Uploader()
				if err := uploader.Upload(
					fileInfo.Name,
					file,
					remote.WithContentType(fileInfo.Mimetype),
					remote.WithMetadata(remote.Metadata{
						"origName": fileHeader.Filename,
					}),
				); err != nil {
					response.SendError(gc, http.StatusInternalServerError, "Возникла ошибка при загрузке файла в S3")
					return
				}
			}

			// Удаляем временный файл после успешной загрузки в S3
			_ = localStorage.Remove(fileInfo.Name)
		}

		// Записываем в БД
		rec := Record{
			Name:     fileInfo.Name,
			OrigName: fileHeader.Filename,
			Size:     fileInfo.Size,
		}
		if result := p.db.Create(&rec); result.Error != nil {
			response.SendError(gc, http.StatusInternalServerError, result.Error.Error())
			return
		}

		response.SendSuccess(gc, fmt.Sprintf("Successfully uploaded %q of size %d", rec.OrigName, rec.Size), response.Result{
			Link:     rec.Link,
			OrigName: rec.OrigName,
			Size:     rec.Size,
		})
	}
}
