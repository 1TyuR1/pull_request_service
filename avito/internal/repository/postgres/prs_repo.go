package postgres

import (
	"avito/internal/domain"
	"avito/internal/repository"
	"context"
	"database/sql"
	"strings"
)

type PRRepo struct {
	db *sql.DB
}

func NewPRRepo(db *sql.DB) *PRRepo {
	return &PRRepo{db: db}
}

func (r *PRRepo) GetOpenAssignmentsByTeam(ctx context.Context, teamName string) ([]repository.ReviewerAssignment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT pr.id, r.user_id
         FROM pull_requests pr
         JOIN pull_request_reviewers r ON pr.id = r.pull_request_id
         JOIN users u ON u.user_id = r.user_id
         WHERE u.team_name = $1 AND pr.status = 'OPEN'`,
		teamName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []repository.ReviewerAssignment
	for rows.Next() {
		var ra repository.ReviewerAssignment
		if err := rows.Scan(&ra.PRID, &ra.UserID); err != nil {
			return nil, err
		}
		res = append(res, ra)
	}
	return res, rows.Err()
}

func (r *PRRepo) GetStats(ctx context.Context) (repository.Stats, error) {
	s := repository.Stats{
		PerReviewer: make(map[string]int64),
		PerStatus:   make(map[string]int64),
	}

	// per reviewer
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, COUNT(*)
         FROM pull_request_reviewers
         GROUP BY user_id`)
	if err != nil {
		return s, err
	}
	defer rows.Close()

	for rows.Next() {
		var uid string
		var cnt int64
		if err := rows.Scan(&uid, &cnt); err != nil {
			return s, err
		}
		s.PerReviewer[uid] = cnt
	}
	if err := rows.Err(); err != nil {
		return s, err
	}

	// per status
	rows, err = r.db.QueryContext(ctx,
		`SELECT status, COUNT(*)
         FROM pull_requests
         GROUP BY status`)
	if err != nil {
		return s, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var cnt int64
		if err := rows.Scan(&status, &cnt); err != nil {
			return s, err
		}
		s.PerStatus[status] = cnt
	}
	return s, rows.Err()
}

func (r *PRRepo) CreatePR(pr domain.PullRequest) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// основная запись PR
	_, err = tx.ExecContext(ctx,
		`INSERT INTO pull_requests (id, name, author_id, status, created_at, merged_at)
         VALUES ($1, $2, $3, $4, $5, $6)`,
		pr.ID, pr.Name, pr.AuthorID, pr.Status, pr.CreatedAt, pr.MergedAt,
	)
	if err != nil {
		// проверка дубликата
		if isUniqueViolation(err) {
			return repository.ErrAlreadyExists
		}
		return err
	}

	// ревьюверы
	for _, rid := range pr.AssignedReviewers {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO pull_request_reviewers (pull_request_id, user_id)
             VALUES ($1, $2)`,
			pr.ID, rid,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PRRepo) GetPR(id string) (*domain.PullRequest, error) {
	ctx := context.Background()

	var pr domain.PullRequest
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, author_id, status, created_at, merged_at
         FROM pull_requests
         WHERE id = $1`,
		id,
	).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// подтягиваем ревьюверов
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id
         FROM pull_request_reviewers
         WHERE pull_request_id = $1`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviewers := make([]string, 0)
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, uid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	pr.AssignedReviewers = reviewers
	return &pr, nil
}

func (r *PRRepo) UpdatePR(pr domain.PullRequest) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE pull_requests
         SET name = $2,
             author_id = $3,
             status = $4,
             created_at = $5,
             merged_at = $6
         WHERE id = $1`,
		pr.ID, pr.Name, pr.AuthorID, pr.Status, pr.CreatedAt, pr.MergedAt,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return repository.ErrNotFound
	}

	// пересоздаём список ревьюверов
	_, err = tx.ExecContext(ctx,
		`DELETE FROM pull_request_reviewers
         WHERE pull_request_id = $1`,
		pr.ID,
	)
	if err != nil {
		return err
	}

	for _, rid := range pr.AssignedReviewers {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO pull_request_reviewers (pull_request_id, user_id)
             VALUES ($1, $2)`,
			pr.ID, rid,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PRRepo) GetPRsByReviewer(userID string) ([]domain.PullRequest, error) {
	ctx := context.Background()

	rows, err := r.db.QueryContext(ctx,
		`SELECT pr.id, pr.name, pr.author_id, pr.status, pr.created_at, pr.merged_at
         FROM pull_requests pr
         JOIN pull_request_reviewers r
           ON pr.id = r.pull_request_id
         WHERE r.user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]domain.PullRequest, 0)
	for rows.Next() {
		var pr domain.PullRequest
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt); err != nil {
			return nil, err
		}
		res = append(res, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key")
}
