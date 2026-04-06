package buildinfo

import "testing"

func TestInfoDefaults(t *testing.T) {
	info := Info()

	if info.Version == "" {
		t.Fatal("expected default version")
	}
	if info.Commit == "" {
		t.Fatal("expected default commit")
	}
	if info.BuiltAt == "" {
		t.Fatal("expected default build timestamp")
	}
}
