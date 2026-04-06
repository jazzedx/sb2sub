package config

import (
	"path/filepath"
	"testing"
)

func TestLoadAndSaveConfig(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")

	cfg := DefaultConfig()
	cfg.Server.Domain = "example.com"
	cfg.Protocols.VLESS.ListenPort = 9443
	cfg.Protocols.Hysteria2.Enabled = false

	if err := Save(file, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := Load(file)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if loaded.Server.Domain != "example.com" {
		t.Fatalf("expected domain example.com, got %s", loaded.Server.Domain)
	}
	if loaded.Protocols.VLESS.ListenPort != 9443 {
		t.Fatalf("expected vless port 9443, got %d", loaded.Protocols.VLESS.ListenPort)
	}
	if loaded.Protocols.Hysteria2.Enabled {
		t.Fatal("expected hysteria2 to be disabled")
	}
}
