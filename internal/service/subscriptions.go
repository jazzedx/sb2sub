package service

import (
	"database/sql"
	"fmt"
	"time"

	"sb2sub/internal/model"
)

func (s *Service) CreateSubscription(subscription model.Subscription) (model.Subscription, error) {
	now := time.Now().UTC().Truncate(time.Second)
	result, err := s.store.DB().Exec(`
		INSERT INTO subscriptions (
			user_id, name, type, token, custom_path, enabled, created_at, updated_at, last_accessed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		subscription.UserID,
		subscription.Name,
		string(subscription.Type),
		subscription.Token,
		subscription.CustomPath,
		boolToInt(subscription.Enabled),
		now.Format(timeLayout),
		now.Format(timeLayout),
	)
	if err != nil {
		return model.Subscription{}, fmt.Errorf("insert subscription: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.Subscription{}, fmt.Errorf("subscription last insert id: %w", err)
	}

	return s.GetSubscriptionByID(id)
}

func (s *Service) RevokeSubscription(id int64) error {
	_, err := s.store.DB().Exec(`
		UPDATE subscriptions
		SET enabled = 0, updated_at = ?
		WHERE id = ?`, time.Now().UTC().Truncate(time.Second).Format(timeLayout), id)
	if err != nil {
		return fmt.Errorf("revoke subscription: %w", err)
	}
	return nil
}

func (s *Service) ListSubscriptionsByUser(userID int64) ([]model.Subscription, error) {
	rows, err := s.store.DB().Query(`
		SELECT id, user_id, name, type, token, custom_path, enabled, created_at, updated_at, last_accessed_at
		FROM subscriptions
		WHERE user_id = ?
		ORDER BY id ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]model.Subscription, 0)
	for rows.Next() {
		subscription, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions, rows.Err()
}

func (s *Service) GetSubscriptionByID(id int64) (model.Subscription, error) {
	row := s.store.DB().QueryRow(`
		SELECT id, user_id, name, type, token, custom_path, enabled, created_at, updated_at, last_accessed_at
		FROM subscriptions
		WHERE id = ?`, id)
	return scanSubscription(row)
}

func (s *Service) GetSubscriptionByToken(token string) (model.Subscription, error) {
	row := s.store.DB().QueryRow(`
		SELECT id, user_id, name, type, token, custom_path, enabled, created_at, updated_at, last_accessed_at
		FROM subscriptions
		WHERE token = ?`, token)
	return scanSubscription(row)
}

func (s *Service) GetSubscriptionByCustomPath(customPath string) (model.Subscription, error) {
	row := s.store.DB().QueryRow(`
		SELECT id, user_id, name, type, token, custom_path, enabled, created_at, updated_at, last_accessed_at
		FROM subscriptions
		WHERE custom_path = ?`, customPath)
	return scanSubscription(row)
}

func (s *Service) TouchSubscriptionAccess(id int64) error {
	_, err := s.store.DB().Exec(`
		UPDATE subscriptions
		SET last_accessed_at = ?, updated_at = ?
		WHERE id = ?`,
		time.Now().UTC().Truncate(time.Second).Format(timeLayout),
		time.Now().UTC().Truncate(time.Second).Format(timeLayout),
		id,
	)
	if err != nil {
		return fmt.Errorf("touch subscription access: %w", err)
	}
	return nil
}

type subscriptionScanner interface {
	Scan(dest ...any) error
}

func scanSubscription(scanner subscriptionScanner) (model.Subscription, error) {
	var (
		subscription   model.Subscription
		enabled        int
		createdAtRaw   string
		updatedAtRaw   string
		lastAccessedAt sql.NullString
	)

	err := scanner.Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.Name,
		&subscription.Type,
		&subscription.Token,
		&subscription.CustomPath,
		&enabled,
		&createdAtRaw,
		&updatedAtRaw,
		&lastAccessedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Subscription{}, err
		}
		return model.Subscription{}, fmt.Errorf("scan subscription: %w", err)
	}

	subscription.Enabled = enabled == 1
	if subscription.CreatedAt, err = time.Parse(timeLayout, createdAtRaw); err != nil {
		return model.Subscription{}, fmt.Errorf("parse subscription created_at: %w", err)
	}
	if subscription.UpdatedAt, err = time.Parse(timeLayout, updatedAtRaw); err != nil {
		return model.Subscription{}, fmt.Errorf("parse subscription updated_at: %w", err)
	}
	if lastAccessedAt.Valid {
		parsed, err := time.Parse(timeLayout, lastAccessedAt.String)
		if err != nil {
			return model.Subscription{}, fmt.Errorf("parse subscription last_accessed_at: %w", err)
		}
		subscription.LastAccessedAt = &parsed
	}

	return subscription, nil
}
