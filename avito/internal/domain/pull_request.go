package domain

import "time"

type PullRequestStatus string

const (
	PRStatusOpen   PullRequestStatus = "OPEN"
	PRStatusMerged PullRequestStatus = "MERGED"
)

type PullRequest struct {
	ID                string            `json:"pull_request_id"`
	Name              string            `json:"pull_request_name"`
	AuthorID          string            `json:"author_id"`
	Status            PullRequestStatus `json:"status"`
	AssignedReviewers []string          `json:"assigned_reviewers"`
	CreatedAt         *time.Time        `json:"createdAt,omitempty"`
	MergedAt          *time.Time        `json:"mergedAt,omitempty"`
}

type PullRequestShort struct {
	ID       string            `json:"pull_request_id"`
	Name     string            `json:"pull_request_name"`
	AuthorID string            `json:"author_id"`
	Status   PullRequestStatus `json:"status"`
}
