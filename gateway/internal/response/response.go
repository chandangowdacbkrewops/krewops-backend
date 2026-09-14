package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const RequestIDContextKey = "request_id"

type Meta struct {
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    Meta        `json:"meta"`
}

type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
	Meta    Meta        `json:"meta"`
}

func newMeta(c *gin.Context) Meta {
	return Meta{
		RequestID: c.GetString(RequestIDContextKey),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    newMeta(c),
	})
}

func Error(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: newMeta(c),
	})
}

func BadRequest(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusBadRequest, CodeForStatus(http.StatusBadRequest), message, details)
}

func Unauthorized(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusUnauthorized, CodeForStatus(http.StatusUnauthorized), message, details)
}

func Forbidden(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusForbidden, CodeForStatus(http.StatusForbidden), message, details)
}

func NotFound(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusNotFound, CodeForStatus(http.StatusNotFound), message, details)
}

func Conflict(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusConflict, CodeForStatus(http.StatusConflict), message, details)
}

func Internal(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusInternalServerError, CodeForStatus(http.StatusInternalServerError), message, details)
}
