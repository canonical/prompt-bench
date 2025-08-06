package main

import (
	"log/slog"
	"math"
)

// timeStats returns the average, max, min, and standard deviation of the given times.
func timeStats(times []uint64) (avg, maxT, minT, dev uint64) {
retry:
	for {
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

		// Ignore any values that are more than 50% away from the average. Suspicious under combined action
		// from snapd still pending, causing this value in general to be really low.
		filteredTimes := make([]uint64, 0, len(times))
		for _, t := range times {
			if t < avg/2 || t > avg*3/2 {
				slog.Info("ignoring suspicious time value", slog.Uint64("time", t), slog.Uint64("avg", avg))
				continue
			}
			filteredTimes = append(filteredTimes, t)
			times = filteredTimes
			continue retry
		}

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
}
