package history

import (
	"path/filepath"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	orig := statsPath
	statsPath = func() (string, error) {
		return filepath.Join(dir, "stats.json"), nil
	}
	defer func() { statsPath = orig }()

	tracker := NewTracker()
	tracker.TodayDown = 12345
	tracker.TodayUp = 67890

	if err := tracker.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := NewTracker()
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.TodayDown != 12345 {
		t.Errorf("TodayDown = %f; want 12345", loaded.TodayDown)
	}
	if loaded.TodayUp != 67890 {
		t.Errorf("TodayUp = %f; want 67890", loaded.TodayUp)
	}
}

func TestLoadMissing(t *testing.T) {
	tracker := NewTracker()
	err := tracker.Load()
	if err == nil {
		t.Error("expected error loading missing file")
	}
}
