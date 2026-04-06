package render

import (
	"encoding/json"
	"testing"

	"sb2sub/internal/config"
)

func TestRenderSingBox(t *testing.T) {
	t.Run("renders both approved protocols with stats", func(t *testing.T) {
		cfg := config.DefaultConfig()
		users := []RuntimeUser{
			{
				Name:              "alice",
				UUID:              "11111111-1111-1111-1111-111111111111",
				Hysteria2Password: "alice-pass",
				Enabled:           true,
				VLESSEnabled:      true,
				Hysteria2Enabled:  true,
			},
		}

		raw, err := RenderSingBox(cfg, users)
		if err != nil {
			t.Fatalf("RenderSingBox returned error: %v", err)
		}

		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("expected valid JSON: %v", err)
		}

		inbounds, ok := doc["inbounds"].([]any)
		if !ok {
			t.Fatalf("expected inbounds array, got %T", doc["inbounds"])
		}
		if len(inbounds) != 2 {
			t.Fatalf("expected two inbounds, got %d", len(inbounds))
		}

		experimental, ok := doc["experimental"].(map[string]any)
		if !ok {
			t.Fatalf("expected experimental config, got %T", doc["experimental"])
		}
		v2rayAPI, ok := experimental["v2ray_api"].(map[string]any)
		if !ok {
			t.Fatalf("expected v2ray_api config, got %T", experimental["v2ray_api"])
		}
		stats, ok := v2rayAPI["stats"].(map[string]any)
		if !ok || stats["enabled"] != true {
			t.Fatalf("expected enabled stats section")
		}
	})

	t.Run("omits disabled protocol", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Protocols.Hysteria2.Enabled = false

		users := []RuntimeUser{
			{
				Name:         "bob",
				UUID:         "22222222-2222-2222-2222-222222222222",
				Enabled:      true,
				VLESSEnabled: true,
			},
		}

		raw, err := RenderSingBox(cfg, users)
		if err != nil {
			t.Fatalf("RenderSingBox returned error: %v", err)
		}

		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("expected valid JSON: %v", err)
		}

		inbounds, ok := doc["inbounds"].([]any)
		if !ok {
			t.Fatalf("expected inbounds array, got %T", doc["inbounds"])
		}
		if len(inbounds) != 1 {
			t.Fatalf("expected one inbound, got %d", len(inbounds))
		}
	})
}
