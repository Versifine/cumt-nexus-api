package httpserver

import (
	"errors"
	"net/http"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/gin-gonic/gin"
)

func writeError(c *gin.Context, err error) {
	c.JSON(mapError(err))
}

func mapError(err error) (int, ErrorResponse) {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return mapAppError(appErr)
	}

	return http.StatusInternalServerError, ErrorResponse{
		Error: ErrorBody{
			Code:    string(apperr.CodeInternal),
			Message: "internal server error",
		},
	}
}

func mapAppError(err *apperr.Error) (int, ErrorResponse) {
	switch err.Code() {
	case apperr.CodeInvalidArgument:
		return http.StatusBadRequest, errorResponse(err)
	case apperr.CodeUnauthenticated:
		return http.StatusUnauthorized, errorResponse(err)
	case apperr.CodeForbidden:
		return http.StatusForbidden, errorResponse(err)
	case apperr.CodeNotFound:
		return http.StatusNotFound, errorResponse(err)
	case apperr.CodeConflict:
		return http.StatusConflict, errorResponse(err)
	case apperr.CodeRateLimited:
		return http.StatusTooManyRequests, errorResponse(err)
	default:
		return http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{
				Code:    string(apperr.CodeInternal),
				Message: "internal server error",
			},
		}
	}
}
