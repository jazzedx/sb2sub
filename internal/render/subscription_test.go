package render

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"sb2sub/internal/config"
	"sb2sub/internal/model"
)

func TestRenderClash(t *testing.T) {
	cfg := config.DefaultConfig()
	user := model.User{
		Username:          "alice",
		VLESSUUID:         "11111111-1111-1111-1111-111111111111",
		Hysteria2Password: "alice-pass",
		Enabled:           true,
		VLESSEnabled:      true,
		Hysteria2Enabled:  true,
	}

	raw, err := RenderClash(cfg, user)
	if err != nil {
		t.Fatalf("RenderClash returned error: %v", err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("expected valid YAML: %v", err)
	}

	rawGroups, ok := doc["proxy-groups"].([]any)
	if !ok {
		t.Fatalf("expected proxy-groups array, got %T", doc["proxy-groups"])
	}

	expectedNames := []string{"手动切换", "国内直连", "国外分流", "广告拦截", "漏网之鱼"}
	if len(rawGroups) != len(expectedNames) {
		t.Fatalf("expected %d proxy groups, got %d", len(expectedNames), len(rawGroups))
	}
	for i, rawGroup := range rawGroups {
		group, ok := rawGroup.(map[string]any)
		if !ok {
			t.Fatalf("expected proxy group map, got %T", rawGroup)
		}
		if group["name"] != expectedNames[i] {
			t.Fatalf("expected group %q, got %v", expectedNames[i], group["name"])
		}
	}

	tun, ok := doc["tun"].(map[string]any)
	if !ok || tun["enable"] != true {
		t.Fatalf("expected tun to be enabled")
	}
	if tun["stack"] != "mixed" {
		t.Fatalf("expected tun stack to be mixed, got %v", tun["stack"])
	}
	dns, ok := doc["dns"].(map[string]any)
	if !ok || dns["enable"] != true {
		t.Fatalf("expected dns to be enabled")
	}
	if dns["ipv6"] != false {
		t.Fatalf("expected ipv6 disabled in dns config")
	}
	if dns["fake-ip-range"] != "198.18.0.1/16" {
		t.Fatalf("expected fake-ip-range to be set")
	}
	sniffer, ok := doc["sniffer"].(map[string]any)
	if !ok || sniffer["enable"] != true {
		t.Fatalf("expected sniffer to be enabled")
	}
	sniff, ok := sniffer["sniff"].(map[string]any)
	if !ok {
		t.Fatalf("expected sniff config")
	}
	httpSniff, ok := sniff["HTTP"].(map[string]any)
	if !ok || httpSniff["override-destination"] != true {
		t.Fatalf("expected HTTP sniff override-destination enabled")
	}
	if _, ok := sniff["QUIC"].(map[string]any); !ok {
		t.Fatalf("expected QUIC sniff config")
	}
	if doc["external-controller"] != "127.0.0.1:9090" {
		t.Fatalf("expected external-controller to be set")
	}
	if doc["log-level"] != "info" {
		t.Fatalf("expected log-level to be info")
	}
	if doc["find-process-mode"] != "strict" {
		t.Fatalf("expected find-process-mode to be strict")
	}

	text := string(raw)
	if !strings.Contains(text, "Loyalsoldier/clash-rules") {
		t.Fatal("expected Loyalsoldier rule provider")
	}
	if !strings.Contains(text, "blackmatrix7/ios_rule_script") {
		t.Fatal("expected blackmatrix7 rule provider")
	}

	orderedSections := []string{
		"allow-lan:",
		"\ndns:",
		"\nexternal-controller:",
		"\nfind-process-mode:",
		"\nlog-level:",
		"\nmixed-port:",
		"\nmode:",
		"\nunified-delay:",
		"\ntun:",
		"\nsniffer:",
		"\nproxies:",
		"\nproxy-groups:",
		"\nrule-providers:",
		"\nrules:",
	}
	lastIndex := -1
	for _, section := range orderedSections {
		index := strings.Index(text, section)
		if index == -1 {
			t.Fatalf("expected section %q in clash output", section)
		}
		if index <= lastIndex {
			t.Fatalf("expected section %q after previous section", section)
		}
		lastIndex = index
	}
}

func TestRenderShadowrocket(t *testing.T) {
	cfg := config.DefaultConfig()
	user := model.User{
		Username:          "bob",
		VLESSUUID:         "22222222-2222-2222-2222-222222222222",
		Hysteria2Password: "bob-pass",
		Enabled:           true,
		VLESSEnabled:      true,
		Hysteria2Enabled:  false,
	}

	raw, err := RenderShadowrocket(cfg, user)
	if err != nil {
		t.Fatalf("RenderShadowrocket returned error: %v", err)
	}

	text := string(raw)
	if !strings.Contains(text, "vless://") {
		t.Fatal("expected vless node in Shadowrocket output")
	}
	if strings.Contains(text, "hysteria2://") {
		t.Fatal("did not expect hysteria2 node when disabled for user")
	}
}
