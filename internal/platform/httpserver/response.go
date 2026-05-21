package httpserver

import "github.com/Versifine/cumt-nexus-api/internal/apperr"

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func errorResponse(err *apperr.Error) ErrorResponse {
	return ErrorResponse{
		Error: ErrorBody{
			Code:    string(err.Code()),
			Message: err.Error(),
		},
	}
}
