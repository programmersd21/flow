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

var statsPath = func() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, err2 := os.UserHomeDir()
		if err2 != nil {
			return "", err2
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "flow")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "stats.json"), nil
}

func (t *Tracker) Save() error {
	path, err := statsPath()
	if err != nil {
		return err
	}
	data := statsData{
		TodayDown: t.TodayDown,
		TodayUp:   t.TodayUp,
		Year:      t.Year,
		Month:     int(t.Month),
		Day:       t.Day,
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	return json.NewEncoder(f).Encode(data)
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
