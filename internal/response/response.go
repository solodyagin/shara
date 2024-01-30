package response

import (
	"net/http"

	"github.com/andoma-go/gin"
)

const (
	MsgTypeSuccess = "success"
	MsgTypeError   = "error"
)

type Result struct {
	Name     string `json:"name"`
	OrigName string `json:"origName"`
	Size     int64  `json:"size"`
}

type ResponseSuccess struct {
	MsgType string `json:"msgType" example:"success"`
	Msg     string `json:"msg" example:"Success message"`
	Result  Result `json:"result"`
}

// Сообщение об ошибке
type ResponseError struct {
	MsgType string `json:"msgType" example:"error"`
	Msg     string `json:"msg" example:"Error message"`
}

func SendSuccess(c *gin.Context, msg string, result Result) {
	c.JSON(http.StatusOK, ResponseSuccess{MsgType: MsgTypeSuccess, Msg: msg, Result: result})
}

func SendError(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, ResponseError{MsgType: MsgTypeError, Msg: msg})
}
