package http

import (
	"avito/internal/domain"
	"avito/internal/errs"
	"avito/internal/service"
	"encoding/json"
	"net/http"
)

type TeamHandler struct {
	svc *service.TeamService
}

func NewTeamHandler(svc *service.TeamService) *TeamHandler {
	return &TeamHandler{svc: svc}
}

type bulkDeactivateRequest struct {
	TeamName string `json:"team_name"`
}

type bulkDeactivateResponse struct {
	TeamName            string `json:"team_name"`
	DeactivatedUsers    int64  `json:"deactivated_users"`
	ReassignedReviewers int64  `json:"reassigned_reviewers"`
}

func (h *TeamHandler) BulkDeactivate(w http.ResponseWriter, r *http.Request) {
	var req bulkDeactivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.TeamName == "" {
		http.Error(w, "team_name is required", http.StatusBadRequest)
		return
	}

	res, err := h.svc.BulkDeactivateTeam(r.Context(), req.TeamName)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bulkDeactivateResponse{
		TeamName:            res.TeamName,
		DeactivatedUsers:    res.DeactivatedUsers,
		ReassignedReviewers: res.ReassignedReviewers,
	})
}

// POST /team/add
func (h *TeamHandler) AddTeam(w http.ResponseWriter, r *http.Request) {
	var team domain.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	created, err := h.svc.CreateTeam(team)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			if appErr.Code == errs.CodeTeamExists {
				WriteAppError(w, appErr, http.StatusBadRequest)
				return
			}
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"team": created,
	})
}

// GET /team/get?team_name=...
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		http.Error(w, "team_name is required", http.StatusBadRequest)
		return
	}

	team, err := h.svc.GetTeam(teamName)
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
	_ = json.NewEncoder(w).Encode(team)
}
