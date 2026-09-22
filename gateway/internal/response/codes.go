package response

import "net/http"

const (
	CodeValidationError = "VALIDATION_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeTooManyRequests = "TOO_MANY_REQUESTS"
	CodeInternalError   = "INTERNAL_ERROR"
)

var statusToCode = map[int]string{
	http.StatusBadRequest:          CodeValidationError,
	http.StatusUnauthorized:        CodeUnauthorized,
	http.StatusForbidden:           CodeForbidden,
	http.StatusNotFound:            CodeNotFound,
	http.StatusConflict:            CodeConflict,
	http.StatusTooManyRequests:     CodeTooManyRequests,
	http.StatusInternalServerError: CodeInternalError,
}

func CodeForStatus(status int) string {
	if code, ok := statusToCode[status]; ok {
		return code
	}
	return CodeInternalError
}
