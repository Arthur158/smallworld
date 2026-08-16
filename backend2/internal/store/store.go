package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists      = errors.New("user already exists")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("incorrect password")
	ErrSaveNotFound    = errors.New("save not found for user")
)

type Store struct {
	db *sql.DB
}

type SaveInfo struct {
	SaveID  int64  `json:"saveId"`
	Summary string `json:"summary"`
}

func Open(databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	// database/sql is already a pool. These are conservative defaults for a
	// gateway; tune them after observing real traffic.
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// EnsureSchema creates only the tables the gateway depends on. The room worker
// will later use the same game_states table for durable save games.
func (s *Store) EnsureSchema(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS game_states (
    id BIGSERIAL PRIMARY KEY,
    state_json JSONB NOT NULL,
    saver_index INTEGER,
    summary TEXT NOT NULL,
    map_name TEXT NOT NULL,
    players_tribes JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS user_savegames (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_state_id BIGINT NOT NULL REFERENCES game_states(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, game_state_id)
);
`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("ensure database schema: %w", err)
	}
	return nil
}

func (s *Store) AddUser(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO users (username, password_hash) VALUES ($1, $2)`,
		username,
		string(hash),
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrUserExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *Store) AuthenticateUser(ctx context.Context, username, password string) error {
	var storedHash string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT password_hash FROM users WHERE username = $1`,
		username,
	).Scan(&storedHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("read user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		return ErrInvalidPassword
	}
	return nil
}

func (s *Store) GetUserSaves(ctx context.Context, username string) ([]SaveInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT gs.id, gs.summary
FROM user_savegames us
JOIN users u ON u.id = us.user_id
JOIN game_states gs ON gs.id = us.game_state_id
WHERE u.username = $1
ORDER BY gs.id ASC
`, username)
	if err != nil {
		return nil, fmt.Errorf("query user saves: %w", err)
	}
	defer rows.Close()

	saves := make([]SaveInfo, 0)
	for rows.Next() {
		var save SaveInfo
		if err := rows.Scan(&save.SaveID, &save.Summary); err != nil {
			return nil, fmt.Errorf("scan user save: %w", err)
		}
		saves = append(saves, save)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user saves: %w", err)
	}
	return saves, nil
}

func (s *Store) RemoveSaveFromUser(ctx context.Context, username string, saveID int64) error {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM user_savegames
WHERE user_id = (SELECT id FROM users WHERE username = $1)
  AND game_state_id = $2
`, username, saveID)
	if err != nil {
		return fmt.Errorf("remove user save: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if n == 0 {
		return ErrSaveNotFound
	}
	return nil
}
