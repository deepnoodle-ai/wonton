package tui

import "math"

// Easing represents an easing function that maps input [0,1] to output [0,1].
// Easing functions control the rate of change over time for animations.
type Easing func(t float64) float64

// Standard easing functions for smooth animations

// EaseLinear provides constant speed (no easing).
func EaseLinear(t float64) float64 {
	return t
}

// EaseInQuad accelerates from zero velocity (quadratic).
func EaseInQuad(t float64) float64 {
	return t * t
}

// EaseOutQuad decelerates to zero velocity (quadratic).
func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

// EaseInOutQuad accelerates until halfway, then decelerates (quadratic).
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// EaseInCubic accelerates from zero velocity (cubic).
func EaseInCubic(t float64) float64 {
	return t * t * t
}

// EaseOutCubic decelerates to zero velocity (cubic).
func EaseOutCubic(t float64) float64 {
	t--
	return t*t*t + 1
}

// EaseInOutCubic accelerates until halfway, then decelerates (cubic).
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	t = 2*t - 2
	return (t*t*t + 2) / 2
}

// EaseInQuart accelerates from zero velocity (quartic).
func EaseInQuart(t float64) float64 {
	return t * t * t * t
}

// EaseOutQuart decelerates to zero velocity (quartic).
func EaseOutQuart(t float64) float64 {
	t--
	return 1 - t*t*t*t
}

// EaseInOutQuart accelerates until halfway, then decelerates (quartic).
func EaseInOutQuart(t float64) float64 {
	if t < 0.5 {
		return 8 * t * t * t * t
	}
	t--
	return 1 - 8*t*t*t*t
}

// EaseInSine accelerates using sine wave.
func EaseInSine(t float64) float64 {
	return 1 - math.Cos(t*math.Pi/2)
}

// EaseOutSine decelerates using sine wave.
func EaseOutSine(t float64) float64 {
	return math.Sin(t * math.Pi / 2)
}

// EaseInOutSine accelerates and decelerates using sine wave.
func EaseInOutSine(t float64) float64 {
	return -(math.Cos(math.Pi*t) - 1) / 2
}

// EaseInExpo accelerates exponentially.
func EaseInExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Pow(2, 10*(t-1))
}

// EaseOutExpo decelerates exponentially.
func EaseOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

// EaseInOutExpo accelerates and decelerates exponentially.
func EaseInOutExpo(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	if t < 0.5 {
		return math.Pow(2, 20*t-10) / 2
	}
	return (2 - math.Pow(2, -20*t+10)) / 2
}

// EaseInCirc accelerates using circular function.
func EaseInCirc(t float64) float64 {
	return 1 - math.Sqrt(1-t*t)
}

// EaseOutCirc decelerates using circular function.
func EaseOutCirc(t float64) float64 {
	t--
	return math.Sqrt(1 - t*t)
}

// EaseInOutCirc accelerates and decelerates using circular function.
func EaseInOutCirc(t float64) float64 {
	if t < 0.5 {
		return (1 - math.Sqrt(1-4*t*t)) / 2
	}
	t = 2*t - 2
	return (math.Sqrt(1-t*t) + 1) / 2
}

// EaseInBack overshoots then returns.
func EaseInBack(t float64) float64 {
	const c1 = 1.70158
	const c3 = c1 + 1
	return c3*t*t*t - c1*t*t
}

// EaseOutBack overshoots then settles.
func EaseOutBack(t float64) float64 {
	const c1 = 1.70158
	const c3 = c1 + 1
	t--
	return 1 + c3*t*t*t + c1*t*t
}

// EaseInOutBack overshoots in both directions.
func EaseInOutBack(t float64) float64 {
	const c1 = 1.70158
	const c2 = c1 * 1.525
	if t < 0.5 {
		return (math.Pow(2*t, 2) * ((c2+1)*2*t - c2)) / 2
	}
	t = 2*t - 2
	return (math.Pow(t, 2)*((c2+1)*t+c2) + 2) / 2
}

// EaseInElastic creates an elastic/spring effect at the start.
func EaseInElastic(t float64) float64 {
	const c4 = (2 * math.Pi) / 3
	if t == 0 || t == 1 {
		return t
	}
	return -math.Pow(2, 10*t-10) * math.Sin((t*10-10.75)*c4)
}

// EaseOutElastic creates an elastic/spring effect at the end.
func EaseOutElastic(t float64) float64 {
	const c4 = (2 * math.Pi) / 3
	if t == 0 || t == 1 {
		return t
	}
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*c4) + 1
}

// EaseInOutElastic creates an elastic effect in both directions.
func EaseInOutElastic(t float64) float64 {
	const c5 = (2 * math.Pi) / 4.5
	if t == 0 || t == 1 {
		return t
	}
	if t < 0.5 {
		return -(math.Pow(2, 20*t-10) * math.Sin((20*t-11.125)*c5)) / 2
	}
	return (math.Pow(2, -20*t+10)*math.Sin((20*t-11.125)*c5))/2 + 1
}

// EaseInBounce creates a bouncing effect at the start.
func EaseInBounce(t float64) float64 {
	return 1 - EaseOutBounce(1-t)
}

// EaseOutBounce creates a bouncing effect at the end.
func EaseOutBounce(t float64) float64 {
	const n1 = 7.5625
	const d1 = 2.75
	if t < 1/d1 {
		return n1 * t * t
	} else if t < 2/d1 {
		t -= 1.5 / d1
		return n1*t*t + 0.75
	} else if t < 2.5/d1 {
		t -= 2.25 / d1
		return n1*t*t + 0.9375
	}
	t -= 2.625 / d1
	return n1*t*t + 0.984375
}

// EaseInOutBounce creates a bouncing effect in both directions.
func EaseInOutBounce(t float64) float64 {
	if t < 0.5 {
		return (1 - EaseOutBounce(1-2*t)) / 2
	}
	return (1 + EaseOutBounce(2*t-1)) / 2
}

// Custom easing function builders

// EaseChain chains multiple easing functions sequentially.
// Each function is applied to an equal portion of the total time.
func EaseChain(easings ...Easing) Easing {
	if len(easings) == 0 {
		return EaseLinear
	}
	return func(t float64) float64 {
		n := len(easings)
		section := int(t * float64(n))
		if section >= n {
			section = n - 1
		}
		sectionT := (t*float64(n) - float64(section))
		return (float64(section) + easings[section](sectionT)) / float64(n)
	}
}

// EaseReverse reverses an easing function (mirrors it).
func EaseReverse(easing Easing) Easing {
	return func(t float64) float64 {
		return 1 - easing(1-t)
	}
}

// EaseMirror creates a mirrored easing (goes up then down).
func EaseMirror(easing Easing) Easing {
	return func(t float64) float64 {
		if t < 0.5 {
			return easing(t * 2)
		}
		return easing((1 - t) * 2)
	}
}

// EaseScale scales the output of an easing function.
func EaseScale(easing Easing, scale float64) Easing {
	return func(t float64) float64 {
		return easing(t) * scale
	}
}

// EaseClamp ensures the output is clamped between 0 and 1.
func EaseClamp(easing Easing) Easing {
	return func(t float64) float64 {
		v := easing(t)
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
}
