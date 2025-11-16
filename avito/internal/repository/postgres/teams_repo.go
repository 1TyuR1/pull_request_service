package postgres

import (
	"avito/internal/domain"
	"avito/internal/repository"
	"context"
	"database/sql"
)

type TeamRepo struct {
	db *sql.DB
}

func NewTeamRepo(db *sql.DB) *TeamRepo {
	return &TeamRepo{db: db}
}

func (r *TeamRepo) CreateTeam(team domain.Team) error {
	ctx := context.Background()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO teams (team_name) VALUES ($1)`,
		team.TeamName,
	)
	if err != nil {
		return err
	}

	// вставляем/обновляем пользователей
	for _, m := range team.Members {
		_, err := r.db.ExecContext(ctx,
			`INSERT INTO users (user_id, username, team_name, is_active)
             VALUES ($1, $2, $3, $4)
             ON CONFLICT (user_id) DO UPDATE
               SET username = EXCLUDED.username,
                   team_name = EXCLUDED.team_name,
                   is_active = EXCLUDED.is_active`,
			m.UserID, m.Username, team.TeamName, m.IsActive,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TeamRepo) GetTeam(name string) (*domain.Team, error) {
	ctx := context.Background()

	// проверяем, что команда существует
	var teamName string
	err := r.db.QueryRowContext(ctx,
		`SELECT team_name FROM teams WHERE team_name = $1`,
		name,
	).Scan(&teamName)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, username, is_active
         FROM users
         WHERE team_name = $1`,
		name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]domain.TeamMember, 0)
	for rows.Next() {
		var m domain.TeamMember
		if err := rows.Scan(&m.UserID, &m.Username, &m.IsActive); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &domain.Team{
		TeamName: name,
		Members:  members,
	}, nil
}
