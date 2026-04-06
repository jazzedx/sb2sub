package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"sb2sub/internal/config"
	"sb2sub/internal/db"
	"sb2sub/internal/model"
	"sb2sub/internal/service"
)

func TestHTTPServer(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "sb2sub.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}

	svc := service.New(store)
	user, err := svc.CreateUser(model.User{
		Username:          "alice",
		Enabled:           true,
		ExpiresAt:         time.Now().UTC().Add(24 * time.Hour),
		QuotaBytes:        10 << 30,
		VLESSUUID:         "11111111-1111-1111-1111-111111111111",
		Hysteria2Password: "alice-pass",
		VLESSEnabled:      true,
		Hysteria2Enabled:  true,
	})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	_, err = svc.CreateSubscription(model.Subscription{
		UserID:     user.ID,
		Name:       "clash-default",
		Type:       model.SubscriptionTypeClash,
		Token:      "sub-token",
		CustomPath: "sub/alice",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}

	handler := NewHandler(config.DefaultConfig(), svc)

	t.Run("health endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("public subscription endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sub/sub-token", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if got := rec.Body.String(); got == "" {
			t.Fatal("expected subscription body")
		}
	})

	t.Run("local user management endpoint", func(t *testing.T) {
		payload, err := json.Marshal(map[string]any{
			"username":           "bob",
			"note":               "second",
			"enabled":            true,
			"quota_bytes":        int64(20 << 30),
			"expires_at":         time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339),
			"vless_uuid":         "22222222-2222-2222-2222-222222222222",
			"hysteria2_password": "bob-pass",
			"vless_enabled":      true,
			"hysteria2_enabled":  false,
		})
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("subscription management endpoint", func(t *testing.T) {
		payload, err := json.Marshal(map[string]any{
			"user_id":     user.ID,
			"name":        "shadowrocket-default",
			"type":        "shadowrocket",
			"token":       "shadow-token",
			"custom_path": "sub/shadow",
			"enabled":     true,
		})
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/sub/shadow-token", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if got := rec.Body.String(); got == "" {
			t.Fatal("expected shadowrocket subscription body")
		}
	})

	t.Run("custom subscription path endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sub/shadow", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if got := rec.Body.String(); got == "" {
			t.Fatal("expected custom-path subscription body")
		}
	})
}
