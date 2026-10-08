package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLOW_DATA", filepath.Dir(dir))

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
	// FLOW_DATA points the stats file somewhere empty, so Load must report a
	// missing file without touching the real one.
	t.Setenv("FLOW_DATA", t.TempDir())

	tracker := NewTracker()
	if err := tracker.Load(); err == nil {
		t.Error("expected an error loading a missing stats file")
	}
}

func TestStatsPathHonoursFlowData(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLOW_DATA", dir)
	got, err := statsPath()
	if err != nil {
		t.Fatalf("statsPath: %v", err)
	}
	want := filepath.Join(dir, "flow", "stats.json")
	if got != want {
		t.Errorf("statsPath = %q, want %q", got, want)
	}
}

func TestSaveLoadRoundTripUsesFlowData(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLOW_DATA", dir)

	tr := NewTracker()
	tr.TodayDown, tr.TodayUp = 111, 222
	if err := tr.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "flow", "stats.json")); err != nil {
		t.Errorf("stats file not written under FLOW_DATA: %v", err)
	}

	loaded := NewTracker()
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.TodayDown != 111 || loaded.TodayUp != 222 {
		t.Errorf("round trip = %v/%v, want 111/222", loaded.TodayDown, loaded.TodayUp)
	}

	if err := Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "flow", "stats.json")); !os.IsNotExist(err) {
		t.Error("Reset did not remove the stats file")
	}
}
