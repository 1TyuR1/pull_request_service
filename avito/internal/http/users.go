package http

import (
	"avito/internal/errs"
	"avito/internal/service"
	"encoding/json"
	"net/http"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

type setIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

// POST /users/setIsActive
func (h *UserHandler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req setIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	user, err := h.svc.SetIsActive(req.UserID, req.IsActive)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			if appErr.Code == errs.CodeNotFound {
				WriteAppError(w, appErr, http.StatusNotFound)
				return
			}
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user": user,
	})
}
