package http

import (
	"avito/internal/domain"
	"avito/internal/errs"
	"avito/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// PullRequestHandler обрабатывает HTTP-запросы, связанные с PR.
type PullRequestHandler struct {
	svc *service.PullRequestService
}

func NewPullRequestHandler(svc *service.PullRequestService) *PullRequestHandler {
	return &PullRequestHandler{svc: svc}
}

//DTO для PR

type createPullRequestRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type mergePullRequestRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type reassignPullRequestRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type reassignPullRequestResponse struct {
	PR         *domain.PullRequest `json:"pr"`
	ReplacedBy string              `json:"replaced_by"`
}

type pullRequestShort struct {
	PullRequestID   string  `json:"pull_request_id"`
	PullRequestName string  `json:"pull_request_name"`
	AuthorID        string  `json:"author_id"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"createdAt"`
	MergedAt        *string `json:"mergedAt,omitempty"`
}

type getUserReviewsResponse struct {
	UserID       string             `json:"user_id"`
	PullRequests []pullRequestShort `json:"pull_requests"`
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ErrorResponse из openapi.yml
type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *errs.AppError

	if errors.As(err, &appErr) {
		resp := errorResponse{}
		resp.Error.Code = string(appErr.Code)
		resp.Error.Message = appErr.Error()

		switch appErr.Code {
		case errs.CodeNotFound:
			respondJSON(w, http.StatusNotFound, resp)
		case errs.CodeTeamExists:
			respondJSON(w, http.StatusBadRequest, resp)
		case errs.CodePRExists:
			respondJSON(w, http.StatusConflict, resp)
		case errs.CodePRMerged, errs.CodeNotAssigned, errs.CodeNoCandidate:
			respondJSON(w, http.StatusConflict, resp)
		default:
			respondJSON(w, http.StatusInternalServerError, resp)
		}
		return
	}

	resp := errorResponse{}
	resp.Error.Code = "INTERNAL"
	resp.Error.Message = "internal error"
	respondJSON(w, http.StatusInternalServerError, resp)
}

// Create: POST /pullRequest/create.
func (h *PullRequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.PullRequestID == "" || req.PullRequestName == "" || req.AuthorID == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	pr, err := h.svc.Create(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := struct {
		PR *domain.PullRequest `json:"pr"`
	}{
		PR: pr,
	}

	respondJSON(w, http.StatusCreated, resp)
}

// Merge: POST /pullRequest/merge.
func (h *PullRequestHandler) Merge(w http.ResponseWriter, r *http.Request) {
	var req mergePullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.PullRequestID == "" {
		http.Error(w, "pull_request_id is required", http.StatusBadRequest)
		return
	}

	pr, err := h.svc.Merge(r.Context(), req.PullRequestID)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := struct {
		PR *domain.PullRequest `json:"pr"`
	}{
		PR: pr,
	}

	respondJSON(w, http.StatusOK, resp)
}

// Reassign: POST /pullRequest/reassign.
func (h *PullRequestHandler) Reassign(w http.ResponseWriter, r *http.Request) {
	var req reassignPullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.PullRequestID == "" || req.OldUserID == "" {
		http.Error(w, "pull_request_id and old_user_id are required", http.StatusBadRequest)
		return
	}

	pr, replacedBy, err := h.svc.Reassign(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := reassignPullRequestResponse{
		PR:         pr,
		ReplacedBy: replacedBy,
	}
	respondJSON(w, http.StatusOK, resp)
}

// GetUserReviews: GET /users/getReview?user_id=....
func (h *PullRequestHandler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	prs, err := h.svc.GetUserReviews(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	out := make([]pullRequestShort, 0, len(prs))
	for _, pr := range prs {
		item := pullRequestShort{
			PullRequestID:   pr.ID,
			PullRequestName: pr.Name,
			AuthorID:        pr.AuthorID,
			Status:          string(pr.Status),
			CreatedAt:       pr.CreatedAt.Format(time.RFC3339),
		}

		if pr.MergedAt != nil {
			s := pr.MergedAt.Format(time.RFC3339)
			item.MergedAt = &s
		}
		out = append(out, item)
	}

	resp := getUserReviewsResponse{
		UserID:       userID,
		PullRequests: out,
	}

	respondJSON(w, http.StatusOK, resp)
}

// StatsResponse возвращает простую статистику по назначениям и статусам.
type StatsResponse struct {
	PerReviewer map[string]int64 `json:"per_reviewer"`
	PerStatus   map[string]int64 `json:"per_status"`
}

type StatsHandler struct {
	prSvc *service.PullRequestService
}

func NewStatsHandler(prSvc *service.PullRequestService) *StatsHandler {
	return &StatsHandler{prSvc: prSvc}
}

// GetStats: GET /stats.
func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.prSvc.GetStats(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	resp := StatsResponse{
		PerReviewer: stats.PerReviewer,
		PerStatus:   stats.PerStatus,
	}
	respondJSON(w, http.StatusOK, resp)
}
