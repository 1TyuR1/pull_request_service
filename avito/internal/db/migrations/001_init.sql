CREATE TABLE IF NOT EXISTS teams (
    team_name TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users (
    user_id   TEXT PRIMARY KEY,
    username  TEXT NOT NULL,
    team_name TEXT NOT NULL REFERENCES teams(team_name) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_type WHERE typname = 'pr_status'
    ) THEN
        CREATE TYPE pr_status AS ENUM ('OPEN', 'MERGED');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS pull_requests (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    author_id  TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    status     pr_status NOT NULL,
    created_at TIMESTAMPTZ,
    merged_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS pull_request_reviewers (
    pull_request_id TEXT NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    PRIMARY KEY (pull_request_id, user_id)
);
