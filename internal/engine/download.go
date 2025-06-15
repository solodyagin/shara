package engine

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

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

		c.FileFromFS(rec.Name, s.fs)
	}
}
