package stats

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"sb2sub/internal/db"
	"sb2sub/internal/model"
	"sb2sub/internal/service"
)

type fakeSource struct {
	records []Usage
}

func (f fakeSource) Collect(context.Context) ([]Usage, error) {
	return f.records, nil
}

func TestCollectorRefresh(t *testing.T) {
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
		VLESSUUID:         "11111111-1111-1111-1111-111111111111",
		Hysteria2Password: "alice-pass",
		VLESSEnabled:      true,
		Hysteria2Enabled:  true,
	})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	collector := NewCollector(fakeSource{
		records: []Usage{
			{Username: "alice", UploadBytes: 123, DownloadBytes: 456},
		},
	}, svc)

	if err := collector.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	refreshedUser, err := svc.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID returned error: %v", err)
	}
	if refreshedUser.UsedUploadBytes != 123 {
		t.Fatalf("expected upload 123, got %d", refreshedUser.UsedUploadBytes)
	}
	if refreshedUser.UsedDownloadBytes != 456 {
		t.Fatalf("expected download 456, got %d", refreshedUser.UsedDownloadBytes)
	}
}
