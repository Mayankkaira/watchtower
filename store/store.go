package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Plan         string    `json:"plan"`
	CreatedAt    time.Time `json:"created_at"`
}

type APIKey struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Monitor struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Type         string    `json:"type"`
	Interval     int       `json:"interval"`
	Status       string    `json:"status"`
	Uptime       float64   `json:"uptime"`
	ResponseTime int       `json:"response_time"`
	CreatedAt    time.Time `json:"created_at"`
}

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			plan TEXT NOT NULL DEFAULT 'free',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			key TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id)`,
		`CREATE TABLE IF NOT EXISTS monitors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'http',
			interval_seconds INTEGER NOT NULL DEFAULT 60,
			status TEXT NOT NULL DEFAULT 'pending',
			uptime REAL NOT NULL DEFAULT 100.0,
			response_time INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_monitors_user ON monitors(user_id)`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("exec %q: %w", q[:40], err)
		}
	}
	return nil
}

func (s *Store) CreateUser(email, passwordHash, name string) (*User, error) {
	res, err := s.db.Exec(
		"INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)",
		email, passwordHash, name,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetUserByID(id)
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, email, password_hash, name, plan, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Plan, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, email, password_hash, name, plan, created_at FROM users WHERE email = ?", email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Plan, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) UpdateUserPlan(userID int64, plan string) error {
	_, err := s.db.Exec("UPDATE users SET plan = ? WHERE id = ?", plan, userID)
	return err
}

func (s *Store) CreateAPIKey(userID int64, key, name string) (*APIKey, error) {
	res, err := s.db.Exec(
		"INSERT INTO api_keys (user_id, key, name) VALUES (?, ?, ?)",
		userID, key, name,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	ak := &APIKey{}
	err = s.db.QueryRow(
		"SELECT id, user_id, key, name, created_at FROM api_keys WHERE id = ?", id,
	).Scan(&ak.ID, &ak.UserID, &ak.Key, &ak.Name, &ak.CreatedAt)
	return ak, err
}

func (s *Store) GetAPIKeysByUser(userID int64) ([]APIKey, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, key, name, created_at FROM api_keys WHERE user_id = ?", userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var ak APIKey
		if err := rows.Scan(&ak.ID, &ak.UserID, &ak.Key, &ak.Name, &ak.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, ak)
	}
	return keys, rows.Err()
}

func (s *Store) DeleteAPIKey(id, userID int64) error {
	res, err := s.db.Exec("DELETE FROM api_keys WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) GetUserByAPIKey(key string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(`
		SELECT u.id, u.email, u.password_hash, u.name, u.plan, u.created_at
		FROM users u JOIN api_keys ak ON u.id = ak.user_id
		WHERE ak.key = ?`, key,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Plan, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) UpdateUser(userID int64, name, email string) (*User, error) {
	_, err := s.db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", name, email, userID)
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(userID)
}

func (s *Store) DeleteUser(userID int64) error {
	res, err := s.db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) CreateMonitor(userID int64, name, url, mType string, interval int) (*Monitor, error) {
	res, err := s.db.Exec(
		"INSERT INTO monitors (user_id, name, url, type, interval_seconds) VALUES (?, ?, ?, ?, ?)",
		userID, name, url, mType, interval,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.getMonitorByID(id)
}

func (s *Store) GetMonitorsByUser(userID int64) ([]Monitor, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, name, url, type, interval_seconds, status, uptime, response_time, created_at FROM monitors WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var monitors []Monitor
	for rows.Next() {
		var m Monitor
		if err := rows.Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.Type, &m.Interval, &m.Status, &m.Uptime, &m.ResponseTime, &m.CreatedAt); err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

func (s *Store) DeleteMonitor(id, userID int64) error {
	res, err := s.db.Exec("DELETE FROM monitors WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) getMonitorByID(id int64) (*Monitor, error) {
	m := &Monitor{}
	err := s.db.QueryRow(
		"SELECT id, user_id, name, url, type, interval_seconds, status, uptime, response_time, created_at FROM monitors WHERE id = ?", id,
	).Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.Type, &m.Interval, &m.Status, &m.Uptime, &m.ResponseTime, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) CountMonitorsByUser(userID int64) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM monitors WHERE user_id = ?", userID).Scan(&count)
	return count, err
}
