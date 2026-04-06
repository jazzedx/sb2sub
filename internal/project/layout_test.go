package project

import (
	"path/filepath"
	"testing"
)

func TestDefaultLayout(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "tmp", "sb2sub")
	layout := DefaultLayout(base)

	if layout.ConfigFile != filepath.Join(base, "etc", "config.yaml") {
		t.Fatalf("unexpected config path: %s", layout.ConfigFile)
	}
	if layout.DatabaseFile != filepath.Join(base, "var", "sb2sub.db") {
		t.Fatalf("unexpected database path: %s", layout.DatabaseFile)
	}
	if layout.RenderDir != filepath.Join(base, "var", "rendered") {
		t.Fatalf("unexpected render path: %s", layout.RenderDir)
	}
}
