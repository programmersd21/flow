package animate

import "math"

const (
	SpringStiffness float64 = 180.0
	SpringDamping   float64 = 12.0
)

func EaseOut(current, target, alpha float64) float64 {
	return current + (target-current)*alpha
}

func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func ColorLerp(r1, g1, b1, r2, g2, b2 uint8, t float64) (uint8, uint8, uint8) {
	t = clampF(t, 0, 1)
	r := uint8(math.Round(Lerp(float64(r1), float64(r2), t)))
	g := uint8(math.Round(Lerp(float64(g1), float64(g2), t)))
	b := uint8(math.Round(Lerp(float64(b1), float64(b2), t)))
	return r, g, b
}

func ThreeWayColorLerp(
	r1, g1, b1, r2, g2, b2, r3, g3, b3 uint8,
	t float64,
) (uint8, uint8, uint8) {
	t = clampF(t, 0, 1)
	if t <= 0.5 {
		return ColorLerp(r1, g1, b1, r2, g2, b2, t*2)
	}
	return ColorLerp(r2, g2, b2, r3, g3, b3, (t-0.5)*2)
}

func Clamp01(t float64) float64 { return clampF(t, 0, 1) }

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func Spring(current, target float64, velocity *float64, dt float64) float64 {
	force := SpringStiffness * (target - current)
	*velocity += force * dt
	*velocity *= 1.0 - SpringDamping*dt
	if *velocity < 0.0001 && *velocity > -0.0001 {
		*velocity = 0
	}
	return current + *velocity*dt
}
