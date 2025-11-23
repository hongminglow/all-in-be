package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/hongminglow/all-in-be/internal/models"
	"github.com/hongminglow/all-in-be/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Ensure Store satisfies the storage.UserStore interface at compile time.
var _ storage.UserStore = (*Store)(nil)

// Store provides Postgres-backed persistence for users.
type Store struct {
	pool *pgxpool.Pool
}

// NewUserStore creates a new Store and runs migrations.
func NewUserStore(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return s, nil
}

// Close releases database resources.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			phone TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'player',
			balance NUMERIC(24,2) NOT NULL DEFAULT 0,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;`,
		`UPDATE users SET password_hash = '' WHERE password_hash IS NULL;`,
		`ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;`,
		`ALTER TABLE users DROP COLUMN IF EXISTS auth_provider_id;`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'player';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS balance NUMERIC(24,2) NOT NULL DEFAULT 0;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx ON users (email);`,
		`CREATE TABLE IF NOT EXISTS role (id BIGINT PRIMARY KEY, role_name TEXT UNIQUE NOT NULL, role_description TEXT);`,
		`INSERT INTO role (id, role_name, role_description) VALUES (1, 'player', 'Normal User'), (2, 'vip-player', 'VIP User'), (3, 'vvip-player', 'VVIP User') ON CONFLICT (id) DO UPDATE SET role_name = EXCLUDED.role_name;`,
		`CREATE TABLE IF NOT EXISTS permission (id BIGINT PRIMARY KEY, permission_name TEXT UNIQUE NOT NULL, permission_description TEXT);`,
		`INSERT INTO permission (id, permission_name, permission_description) VALUES (1, 'game:play', 'Play games'), (2, 'bonus:claim', 'Claim bonuses'), (3, 'support:priority', 'Priority support') ON CONFLICT (id) DO NOTHING;`,
		`CREATE TABLE IF NOT EXISTS role_permissions (role_id BIGINT NOT NULL, permission_id BIGINT NOT NULL, PRIMARY KEY (role_id, permission_id), FOREIGN KEY (role_id) REFERENCES role(id), FOREIGN KEY (permission_id) REFERENCES permission(id));`,
		`INSERT INTO role_permissions (role_id, permission_id) VALUES (1, 1), (2, 1), (2, 2), (3, 1), (3, 2), (3, 3) ON CONFLICT DO NOTHING;`,
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
	}
	return nil
}

// CreateUser inserts a new user row.
func (s *Store) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	const query = `
		WITH inserted AS (
			INSERT INTO users (username, email, phone, role, balance, password_hash)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, username, email, phone, role, balance, password_hash, created_at
		)
		SELECT i.id, i.username, i.email, i.phone, i.role, i.balance, i.password_hash, i.created_at, r.role_name,
		(
			SELECT COALESCE(array_agg(p.permission_name), '{}')
			FROM role_permissions rp
			JOIN permission p ON rp.permission_id = p.id
			WHERE rp.role_id = r.id
		)
		FROM inserted i
		JOIN role r ON i.role = r.role_name;
		`
	row := s.pool.QueryRow(ctx, query, user.Username, user.Email, user.Phone, user.Role, user.Balance, user.PasswordHash)
	created, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, storage.ErrAlreadyExists
		}
		return models.User{}, err
	}
	return created, nil
}

// FindByUsername fetches a user by username.
func (s *Store) FindByUsername(ctx context.Context, username string) (models.User, error) {
	const query = `
	SELECT u.id, u.username, u.email, u.phone, u.role, u.balance, u.password_hash, u.created_at, r.role_name,
	(
		SELECT COALESCE(array_agg(p.permission_name), '{}')
		FROM role_permissions rp
		JOIN permission p ON rp.permission_id = p.id
		WHERE rp.role_id = r.id
	)
	FROM users u
	JOIN role r ON u.role = r.role_name
	WHERE u.username = $1;
	`
	row := s.pool.QueryRow(ctx, query, username)
	return scanUser(row)
}

// FindByEmail fetches a user by email address.
func (s *Store) FindByEmail(ctx context.Context, email string) (models.User, error) {
	const query = `
	SELECT u.id, u.username, u.email, u.phone, u.role, u.balance, u.password_hash, u.created_at, r.role_name,
	(
		SELECT COALESCE(array_agg(p.permission_name), '{}')
		FROM role_permissions rp
		JOIN permission p ON rp.permission_id = p.id
		WHERE rp.role_id = r.id
	)
	FROM users u
	JOIN role r ON u.role = r.role_name
	WHERE u.email = $1;
	`
	row := s.pool.QueryRow(ctx, query, email)
	return scanUser(row)
}

// FindByUsernameOrEmail fetches the first user matching the identifier as username or email.
func (s *Store) FindByUsernameOrEmail(ctx context.Context, identifier string) (models.User, error) {
	const query = `
	SELECT u.id, u.username, u.email, u.phone, u.role, u.balance, u.password_hash, u.created_at, r.role_name,
	(
		SELECT COALESCE(array_agg(p.permission_name), '{}')
		FROM role_permissions rp
		JOIN permission p ON rp.permission_id = p.id
		WHERE rp.role_id = r.id
	)
	FROM users u
	JOIN role r ON u.role = r.role_name
	WHERE u.username = $1 OR u.email = $1
	LIMIT 1;
	`
	row := s.pool.QueryRow(ctx, query, identifier)
	return scanUser(row)
}

func scanUser(row pgx.Row) (models.User, error) {
	var user models.User
	var roleName string
	if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Phone, &user.Role, &user.Balance, &user.PasswordHash, &user.CreatedAt, &roleName, &user.Permissions); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, storage.ErrNotFound
		}
		return models.User{}, err
	}
	return user, nil
}
