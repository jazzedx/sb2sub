package service

import (
	"path/filepath"
	"testing"
	"time"

	"sb2sub/internal/db"
	"sb2sub/internal/model"
)

func TestUserAndSubscriptionLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "sb2sub.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}

	svc := New(store)
	user := model.User{
		Username:          "alice",
		Note:              "first user",
		Enabled:           true,
		ExpiresAt:         time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second),
		QuotaBytes:        50 << 30,
		VLESSUUID:         "11111111-1111-1111-1111-111111111111",
		Hysteria2Password: "alice-pass",
		VLESSEnabled:      true,
		Hysteria2Enabled:  true,
	}

	createdUser, err := svc.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if createdUser.ID == 0 {
		t.Fatal("expected created user ID")
	}

	users, err := svc.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected one user, got %d", len(users))
	}
	if users[0].QuotaBytes != user.QuotaBytes {
		t.Fatalf("expected quota %d, got %d", user.QuotaBytes, users[0].QuotaBytes)
	}

	createdUser.Enabled = false
	createdUser.Note = "updated"
	if err := svc.UpdateUser(createdUser); err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}

	updatedUsers, err := svc.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}
	if updatedUsers[0].Enabled {
		t.Fatal("expected updated user to be disabled")
	}

	subscription, err := svc.CreateSubscription(model.Subscription{
		UserID:     createdUser.ID,
		Name:       "clash-default",
		Type:       model.SubscriptionTypeClash,
		Token:      "sub-token",
		CustomPath: "links/alice",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}
	if subscription.ID == 0 {
		t.Fatal("expected created subscription ID")
	}

	subs, err := svc.ListSubscriptionsByUser(createdUser.ID)
	if err != nil {
		t.Fatalf("ListSubscriptionsByUser returned error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected one subscription, got %d", len(subs))
	}

	if err := svc.RevokeSubscription(subscription.ID); err != nil {
		t.Fatalf("RevokeSubscription returned error: %v", err)
	}

	subs, err = svc.ListSubscriptionsByUser(createdUser.ID)
	if err != nil {
		t.Fatalf("ListSubscriptionsByUser returned error: %v", err)
	}
	if subs[0].Enabled {
		t.Fatal("expected revoked subscription to be disabled")
	}
}
