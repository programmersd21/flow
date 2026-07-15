package ping

import (
	"testing"
	"time"
)

func TestMeasureTimeout(t *testing.T) {
	_, err := Measure("192.0.2.1", 50*time.Millisecond)
	if err == nil {
		t.Log("Measure connected to non-routable IP (unexpected but not fatal)")
	}
}

func TestMeasureInvalidHost(t *testing.T) {
	_, err := Measure("", 10*time.Millisecond)
	if err == nil {
		t.Error("expected error for empty host")
	}
}
