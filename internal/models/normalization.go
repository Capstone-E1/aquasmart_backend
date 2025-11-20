package models

import "math"

// SensorReadingWithNormalizedTurbidity extends SensorReading with normalized turbidity
type SensorReadingWithNormalizedTurbidity struct {
	SensorReading
	NormalizedTurbidity float64 `json:"normalized_turbidity"`
}

// NormalizeTurbidity converts turbidity (0-1000 NTU) to 0-1 scale where 0 = good (clear), 1 = bad (very cloudy)
// Optimal: 0-1 NTU (0-0.1 normalized), Normal: 0-5 NTU (0-0.3 normalized)
// Warning: 5-50 NTU (0.3-0.7 normalized), Danger: 50+ NTU (0.7-1.0 normalized)
func NormalizeTurbidity(rawTurbidity float64) float64 {
	const (
		maxTurbidity       = 1000.0
		normalThreshold    = 5.0
		warningThreshold   = 50.0
	)

	// Within normal range (0-5 NTU)
	if rawTurbidity <= normalThreshold {
		return (rawTurbidity / normalThreshold) * 0.3 // 0 to 0.3
	}

	// Warning range (5-50 NTU)
	if rawTurbidity <= warningThreshold {
		progress := (rawTurbidity - normalThreshold) / (warningThreshold - normalThreshold)
		return 0.3 + progress*0.4 // 0.3 to 0.7
	}

	// Danger range (50+ NTU)
	progress := math.Min((rawTurbidity-warningThreshold)/(maxTurbidity-warningThreshold), 1.0)
	return 0.7 + progress*0.3 // 0.7 to 1.0
}

// AddNormalizedTurbidity adds normalized turbidity to a SensorReading
func (s *SensorReading) AddNormalizedTurbidity() SensorReadingWithNormalizedTurbidity {
	return SensorReadingWithNormalizedTurbidity{
		SensorReading:       *s,
		NormalizedTurbidity: NormalizeTurbidity(s.Turbidity),
	}
}

// GetTurbidityStatus returns turbidity status based on normalized value (0 = good, 1 = bad)
// Returns: "normal", "warning", or "danger"
func GetTurbidityStatus(normalizedTurbidity float64) string {
	switch {
	case normalizedTurbidity <= 0.3:
		return "normal"
	case normalizedTurbidity <= 0.7:
		return "warning"
	default:
		return "danger"
	}
}

// GetTurbidityStatusLevel returns a more detailed turbidity status level
// Returns: "excellent", "good", "fair", "warning", or "danger"
func GetTurbidityStatusLevel(normalizedTurbidity float64) string {
	switch {
	case normalizedTurbidity <= 0.1:
		return "excellent"
	case normalizedTurbidity <= 0.3:
		return "good"
	case normalizedTurbidity <= 0.5:
		return "fair"
	case normalizedTurbidity <= 0.7:
		return "warning"
	default:
		return "danger"
	}
}
