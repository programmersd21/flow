package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type statsData struct {
	TodayDown float64 `json:"today_down"`
	TodayUp   float64 `json:"today_up"`
	Year      int     `json:"year"`
	Month     int     `json:"month"`
	Day       int     `json:"day"`
}

// statsPath returns the file used to persist daily totals.
//
// FLOW_DATA overrides the location. Without it the path follows the platform
// convention under the user config dir. The override exists for the same reason
// as FLOW_CONFIG: so tests, and anyone who wants flow's data kept apart from
// its config, can point it somewhere else.
//
// The directory is created by Save, not here: Load and Reset must not have the
// side effect of creating storage just because they asked where it lives.
func statsPath() (string, error) {
	base := os.Getenv("FLOW_DATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			home, err2 := os.UserHomeDir()
			if err2 != nil {
				return "", err2
			}
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, "flow", "stats.json"), nil
}

// Reset deletes the persisted stats file. Missing file is not an error.
func Reset() error {
	path, err := statsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Save writes the tracker atomically (temp file + rename) so a crash, a full
// disk, or a concurrent reader never sees a truncated stats file. The old
// version truncated in place with os.Create and ignored Close errors.
func (t *Tracker) Save() error {
	path, err := statsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data := statsData{
		TodayDown: t.TodayDown,
		TodayUp:   t.TodayUp,
		Year:      t.Year,
		Month:     int(t.Month),
		Day:       t.Day,
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (t *Tracker) Load() error {
	path, err := statsPath()
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	var data statsData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return err
	}
	t.TodayDown = data.TodayDown
	t.TodayUp = data.TodayUp
	t.Year = data.Year
	t.Month = time.Month(data.Month)
	t.Day = data.Day
	return nil
}
