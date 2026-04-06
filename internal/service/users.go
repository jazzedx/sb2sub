package service

import (
	"database/sql"
	"fmt"
	"time"

	"sb2sub/internal/db"
	"sb2sub/internal/model"
)

type Service struct {
	store *db.Store
}

func New(store *db.Store) *Service {
	return &Service{store: store}
}

const timeLayout = time.RFC3339

func (s *Service) CreateUser(user model.User) (model.User, error) {
	now := time.Now().UTC().Truncate(time.Second)
	if user.ExpiresAt.IsZero() {
		user.ExpiresAt = now.Add(30 * 24 * time.Hour)
	}

	result, err := s.store.DB().Exec(`
		INSERT INTO users (
			username, note, enabled, created_at, updated_at, expires_at,
			quota_bytes, used_upload_bytes, used_download_bytes,
			vless_uuid, hysteria2_password, vless_enabled, hysteria2_enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Username,
		user.Note,
		boolToInt(user.Enabled),
		now.Format(timeLayout),
		now.Format(timeLayout),
		user.ExpiresAt.UTC().Format(timeLayout),
		user.QuotaBytes,
		user.UsedUploadBytes,
		user.UsedDownloadBytes,
		user.VLESSUUID,
		user.Hysteria2Password,
		boolToInt(user.VLESSEnabled),
		boolToInt(user.Hysteria2Enabled),
	)
	if err != nil {
		return model.User{}, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.User{}, fmt.Errorf("user last insert id: %w", err)
	}

	return s.GetUserByID(id)
}

func (s *Service) UpdateUser(user model.User) error {
	now := time.Now().UTC().Truncate(time.Second)
	_, err := s.store.DB().Exec(`
		UPDATE users
		SET note = ?, enabled = ?, updated_at = ?, expires_at = ?, quota_bytes = ?,
		    used_upload_bytes = ?, used_download_bytes = ?, vless_uuid = ?,
		    hysteria2_password = ?, vless_enabled = ?, hysteria2_enabled = ?
		WHERE id = ?`,
		user.Note,
		boolToInt(user.Enabled),
		now.Format(timeLayout),
		user.ExpiresAt.UTC().Format(timeLayout),
		user.QuotaBytes,
		user.UsedUploadBytes,
		user.UsedDownloadBytes,
		user.VLESSUUID,
		user.Hysteria2Password,
		boolToInt(user.VLESSEnabled),
		boolToInt(user.Hysteria2Enabled),
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (s *Service) ListUsers() ([]model.User, error) {
	rows, err := s.store.DB().Query(`
		SELECT id, username, note, enabled, created_at, updated_at, expires_at,
		       quota_bytes, used_upload_bytes, used_download_bytes,
		       vless_uuid, hysteria2_password, vless_enabled, hysteria2_enabled
		FROM users
		ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

func (s *Service) GetUserByID(id int64) (model.User, error) {
	row := s.store.DB().QueryRow(`
		SELECT id, username, note, enabled, created_at, updated_at, expires_at,
		       quota_bytes, used_upload_bytes, used_download_bytes,
		       vless_uuid, hysteria2_password, vless_enabled, hysteria2_enabled
		FROM users
		WHERE id = ?`, id)
	return scanUser(row)
}

func (s *Service) GetUserByUsername(username string) (model.User, error) {
	row := s.store.DB().QueryRow(`
		SELECT id, username, note, enabled, created_at, updated_at, expires_at,
		       quota_bytes, used_upload_bytes, used_download_bytes,
		       vless_uuid, hysteria2_password, vless_enabled, hysteria2_enabled
		FROM users
		WHERE username = ?`, username)
	return scanUser(row)
}

func (s *Service) SetUserUsage(username string, uploadBytes, downloadBytes int64) error {
	_, err := s.store.DB().Exec(`
		UPDATE users
		SET used_upload_bytes = ?, used_download_bytes = ?, updated_at = ?
		WHERE username = ?`,
		uploadBytes,
		downloadBytes,
		time.Now().UTC().Truncate(time.Second).Format(timeLayout),
		username,
	)
	if err != nil {
		return fmt.Errorf("set user usage: %w", err)
	}
	return nil
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (model.User, error) {
	var (
		user             model.User
		enabled          int
		vlessEnabled     int
		hysteria2Enabled int
		createdAtRaw     string
		updatedAtRaw     string
		expiresAtRaw     string
	)

	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.Note,
		&enabled,
		&createdAtRaw,
		&updatedAtRaw,
		&expiresAtRaw,
		&user.QuotaBytes,
		&user.UsedUploadBytes,
		&user.UsedDownloadBytes,
		&user.VLESSUUID,
		&user.Hysteria2Password,
		&vlessEnabled,
		&hysteria2Enabled,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, err
		}
		return model.User{}, fmt.Errorf("scan user: %w", err)
	}

	user.Enabled = enabled == 1
	user.VLESSEnabled = vlessEnabled == 1
	user.Hysteria2Enabled = hysteria2Enabled == 1

	if user.CreatedAt, err = time.Parse(timeLayout, createdAtRaw); err != nil {
		return model.User{}, fmt.Errorf("parse created_at: %w", err)
	}
	if user.UpdatedAt, err = time.Parse(timeLayout, updatedAtRaw); err != nil {
		return model.User{}, fmt.Errorf("parse updated_at: %w", err)
	}
	if user.ExpiresAt, err = time.Parse(timeLayout, expiresAtRaw); err != nil {
		return model.User{}, fmt.Errorf("parse expires_at: %w", err)
	}

	return user, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
