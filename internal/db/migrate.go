package db

import "fmt"

func (s *Store) Migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			note TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			quota_bytes INTEGER NOT NULL DEFAULT 0,
			used_upload_bytes INTEGER NOT NULL DEFAULT 0,
			used_download_bytes INTEGER NOT NULL DEFAULT 0,
			vless_uuid TEXT NOT NULL,
			hysteria2_password TEXT NOT NULL,
			vless_enabled INTEGER NOT NULL DEFAULT 1,
			hysteria2_enabled INTEGER NOT NULL DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			custom_path TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_accessed_at TEXT,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
	}

	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("apply migration: %w", err)
		}
	}

	return nil
}
