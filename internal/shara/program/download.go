package program

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"application/internal/shara/response"
)

// HandleDownload
func (p *Program) HandleDownload() gin.HandlerFunc {
	return func(gc *gin.Context) {
		link := gc.Param("link")
		if link == "" {
			response.SendError(gc, http.StatusBadRequest, "Неправильная ссылка")
			return
		}

		var rec Record
		if err := p.db.Where("link = @link", sql.Named("link", link)).First(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.SendError(gc, http.StatusNotFound, "Файл не найден")
				return
			}

			response.SendError(gc, http.StatusInternalServerError, err.Error())
			return
		}

		gc.FileFromFS(rec.Name, p.httpFS)
	}
}
