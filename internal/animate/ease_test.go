package animate

import (
	"math"
	"testing"
)

func TestClamp01(t *testing.T) {
	if got := Clamp01(-0.5); got != 0 {
		t.Errorf("Clamp01(-0.5) = %f; want 0", got)
	}
	if got := Clamp01(1.5); got != 1 {
		t.Errorf("Clamp01(1.5) = %f; want 1", got)
	}
	if got := Clamp01(0.5); got != 0.5 {
		t.Errorf("Clamp01(0.5) = %f; want 0.5", got)
	}
}

func TestLerp(t *testing.T) {
	if got := Lerp(0, 100, 0.5); got != 50 {
		t.Errorf("Lerp(0, 100, 0.5) = %f; want 50", got)
	}
}

func TestSpring(t *testing.T) {
	var vel float64
	val := Spring(0, 100, &vel, 0.016)
	if val <= 0 {
		t.Errorf("Spring(0, 100) = %f; want > 0 after one step", val)
	}
}

func TestColorLerp(t *testing.T) {
	r, g, b := ColorLerp(0, 0, 0, 255, 255, 255, 0.5)
	if r != 128 || g != 128 || b != 128 {
		t.Errorf("ColorLerp(black, white, 0.5) = (%d,%d,%d); want (128,128,128)", r, g, b)
	}
}

func TestSpringConverges(t *testing.T) {
	var vel float64
	val := 0.0
	for i := 0; i < 1000; i++ {
		val = Spring(val, 100, &vel, 0.016)
	}
	if math.Abs(val-100) > 1 {
		t.Errorf("Spring did not converge: %f (want ~100)", val)
	}
}
