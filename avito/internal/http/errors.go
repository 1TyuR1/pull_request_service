package http

import (
	"avito/internal/errs"
	"encoding/json"
	"net/http"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func WriteAppError(w http.ResponseWriter, err *errs.AppError, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	body := errorBody{}
	body.Error.Code = string(err.Code)
	body.Error.Message = err.Msg

	_ = json.NewEncoder(w).Encode(body)
}
