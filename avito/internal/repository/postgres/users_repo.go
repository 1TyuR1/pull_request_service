package postgres

import (
	"avito/internal/domain"
	"avito/internal/repository"
	"context"
	"database/sql"
	"fmt"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) DeactivateByTeam(ctx context.Context, teamName string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users
         SET is_active = false
         WHERE team_name = $1 AND is_active = true`,
		teamName,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *UserRepo) UpsertUser(u domain.User) error {
	ctx := context.Background()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (user_id, username, team_name, is_active)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (user_id) DO UPDATE
           SET username = EXCLUDED.username,
               team_name = EXCLUDED.team_name,
               is_active = EXCLUDED.is_active`,
		u.ID, u.Username, u.TeamName, u.IsActive,
	)
	return err
}

func (r *UserRepo) GetUser(userID string) (*domain.User, error) {
	ctx := context.Background()

	var u domain.User
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, username, team_name, is_active
         FROM users
         WHERE user_id = $1`,
		userID,
	).Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) SetUserActive(userID string, isActive bool) (*domain.User, error) {
	ctx := context.Background()

	res, err := r.db.ExecContext(ctx,
		`UPDATE users
         SET is_active = $2
         WHERE user_id = $1`,
		userID, isActive,
	)
	if err != nil {
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return nil, repository.ErrNotFound
	}

	return r.GetUser(userID)
}

func (r *UserRepo) GetActiveUsersByTeam(teamName string, excludeIDs []string) ([]domain.User, error) {
	ctx := context.Background()

	query := `
        SELECT user_id, username, team_name, is_active
        FROM users
        WHERE team_name = $1 AND is_active = true`
	args := []any{teamName}

	if len(excludeIDs) > 0 {

		query += " AND user_id NOT IN ("
		for i := range excludeIDs {
			if i > 0 {
				query += ","
			}
			query += "$" + fmt.Sprint(i+2)
		}
		query += ")"

		for _, id := range excludeIDs {
			args = append(args, id)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, err
		}
		res = append(res, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}
