package format

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
)

// disabled reports whether color output is suppressed.
func disabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return true
	}
	return false
}

// Data is the template context for --format.
type Data struct {
	Iface       string
	DownBps     float64
	UpBps       float64
	Down        string
	Up          string
	PeakDownBps float64
	PeakUpBps   float64
	TodayDown   string
	TodayUp     string
	PingMs      float64
	LossPct     float64
	JitterMs    float64
	Time        string
	// DownSamples holds recent raw rates for the spark function.
	DownSamples []float64
}

// funcs is the small documented function set available in templates.
var funcs = template.FuncMap{
	"rate": func(bps float64) string {
		return humanRate(bps, false)
	},
	"bytes": func(b float64) string {
		return humanBytes(b)
	},
	"bits": func(bps float64) string {
		return humanRate(bps, true)
	},
	"pad": func(n int, s string) string {
		r := []rune(s)
		if len(r) >= n {
			return string(r[:n])
		}
		return s + strings.Repeat(" ", n-len(r))
	},
	"color": func(name, s string) string {
		if disabled() {
			return s
		}
		codes := map[string]string{
			"red": "31", "green": "32", "yellow": "33",
			"blue": "34", "magenta": "35", "cyan": "36",
			"white": "37", "dim": "2", "bold": "1",
		}
		code, ok := codes[strings.ToLower(name)]
		if !ok {
			return s
		}
		return "\x1b[" + code + "m" + s + "\x1b[0m"
	},
	"spark": func(samples []float64) string {
		if len(samples) == 0 {
			return ""
		}
		blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
		max := 0.0
		for _, v := range samples {
			if v > max {
				max = v
			}
		}
		if max <= 0 {
			return strings.Repeat(string(blocks[0]), len(samples))
		}
		var sb strings.Builder
		for _, v := range samples {
			idx := int(v / max * float64(len(blocks)-1))
			sb.WriteRune(blocks[idx])
		}
		return sb.String()
	},
}

func humanRate(bps float64, bits bool) string {
	if bits {
		bps *= 8
	}
	unit := "B/s"
	if bits {
		unit = "b/s"
	}
	switch {
	case bps >= 1_000_000_000:
		return fmt.Sprintf("%.2f G%s", bps/1_000_000_000, unit)
	case bps >= 1_000_000:
		return fmt.Sprintf("%.1f M%s", bps/1_000_000, unit)
	case bps >= 1_000:
		return fmt.Sprintf("%.0f K%s", bps/1_000, unit)
	default:
		return fmt.Sprintf("%.0f %s", bps, unit)
	}
}

// Bytes formats a byte count for humans ("16.9 MB").
func Bytes(b float64) string {
	return humanBytes(b)
}

func humanBytes(b float64) string {
	switch {
	case b >= 1_000_000_000:
		return fmt.Sprintf("%.1f GB", b/1_000_000_000)
	case b >= 1_000_000:
		return fmt.Sprintf("%.1f MB", b/1_000_000)
	case b >= 1_000:
		return fmt.Sprintf("%.0f KB", b/1_000)
	default:
		return fmt.Sprintf("%.0f B", b)
	}
}

// RenderTemplate executes a user template against data. Template errors are
// returned so the caller can exit with code 2 and a clear stderr message.
func RenderTemplate(tmplStr string, data Data) (string, error) {
	tmpl, err := template.New("format").Funcs(funcs).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// FormatTime renders t in RFC3339Nano, always UTC.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
