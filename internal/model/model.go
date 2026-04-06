package model

import "time"

type SubscriptionType string

const (
	SubscriptionTypeClash        SubscriptionType = "clash"
	SubscriptionTypeShadowrocket SubscriptionType = "shadowrocket"
	SubscriptionTypeBoth         SubscriptionType = "both"
)

type User struct {
	ID                int64     `json:"id"`
	Username          string    `json:"username"`
	Note              string    `json:"note"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	QuotaBytes        int64     `json:"quota_bytes"`
	UsedUploadBytes   int64     `json:"used_upload_bytes"`
	UsedDownloadBytes int64     `json:"used_download_bytes"`
	VLESSUUID         string    `json:"vless_uuid"`
	Hysteria2Password string    `json:"hysteria2_password"`
	VLESSEnabled      bool      `json:"vless_enabled"`
	Hysteria2Enabled  bool      `json:"hysteria2_enabled"`
}

func (u User) UsedTotalBytes() int64 {
	return u.UsedUploadBytes + u.UsedDownloadBytes
}

type Subscription struct {
	ID             int64            `json:"id"`
	UserID         int64            `json:"user_id"`
	Name           string           `json:"name"`
	Type           SubscriptionType `json:"type"`
	Token          string           `json:"token"`
	CustomPath     string           `json:"custom_path"`
	Enabled        bool             `json:"enabled"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	LastAccessedAt *time.Time       `json:"last_accessed_at,omitempty"`
}
