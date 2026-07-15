package animate

import "math"

const (
	stiffness = 180.0
	damping   = 12.0
)

func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func ColorLerp(r1, g1, b1, r2, g2, b2 uint8, t float64) (uint8, uint8, uint8) {
	t = Clamp01(t)
	r := uint8(math.Round(Lerp(float64(r1), float64(r2), t)))
	g := uint8(math.Round(Lerp(float64(g1), float64(g2), t)))
	b := uint8(math.Round(Lerp(float64(b1), float64(b2), t)))
	return r, g, b
}

func Clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

func Spring(current, target float64, velocity *float64, dt float64) float64 {
	force := stiffness * (target - current)
	*velocity += force * dt
	*velocity *= 1.0 - damping*dt
	if *velocity < 0.0001 && *velocity > -0.0001 {
		*velocity = 0
	}
	return current + *velocity*dt
}
