package main

import (
	"math"
)

// timeStats returns the average, max, min, and standard deviation of the given times.
func timeStats(times []uint64) (avg, maxT, minT, dev uint64) {
	if len(times) == 0 {
		return 0, 0, 0, 0
	}

	maxT = times[0]
	minT = times[0]
	sum := uint64(0)
	for _, t := range times {
		maxT = max(t, maxT)
		minT = min(t, minT)
		sum += t
	}
	avg = sum / uint64(len(times))

	// Compute standard deviation
	var variance uint64 = 0
	if len(times) > 1 {
		varianceSum := uint64(0)
		for _, t := range times {
			varianceSum += (t - avg) * (t - avg)
		}
		variance = varianceSum / uint64(len(times)-1)
	}
	dev = uint64(math.Sqrt(float64(variance)))

	return avg, maxT, minT, dev
}
