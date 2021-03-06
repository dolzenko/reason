package core

import (
	"math"

	"github.com/bsm/reason/internal/calc"
)

// NumSeries maintains information about a series of (weighted) numeric data
type NumSeries struct{ weight, sum, sumSquares float64 }

// Append adds a new value to the series, with a weight
func (s *NumSeries) Append(value, weight float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}

	wv := weight * value
	s.weight += weight
	s.sum += wv
	s.sumSquares += wv * value
}

// TotalWeight returns total observed weight of that series, usually equavalent
// to the count of observations
func (s *NumSeries) TotalWeight() float64 { return s.weight }

// IsZero returns true if there are no values in the series
func (s *NumSeries) IsZero() bool { return s.weight <= 0 }

// Sum returns the sum of all observed values
func (s *NumSeries) Sum() float64 { return s.sum }

// Mean returns a mean average
func (s *NumSeries) Mean() float64 {
	if s.weight != 0 {
		return s.sum / s.weight
	}
	return 0.0
}

// Variance is variance of the series
func (s *NumSeries) Variance() float64 {
	if s.weight > 0 {
		x := (s.sum * s.sum) / s.weight
		return (s.sumSquares - x) / s.weight
	}
	return 0.0
}

// StdDev is the standard deviation of the series
func (s *NumSeries) StdDev() float64 {
	return math.Sqrt(s.Variance())
}

// SampleVariance is the sample variance of the series
func (s *NumSeries) SampleVariance() float64 {
	if s.weight > 1 {
		x := (s.sum * s.sum) / s.weight
		return (s.sumSquares - x) / (s.weight - 1)
	}
	return 0.0
}

// SampleStdDev is the sample standard deviation of the series
func (s *NumSeries) SampleStdDev() float64 {
	return math.Sqrt(s.SampleVariance())
}

var gaussianNormalConstant = math.Sqrt(2 * math.Pi)

// ProbDensity calculates the gaussian probability density of a value
func (s *NumSeries) ProbDensity(value float64) float64 {
	if s.weight > 0 {
		mean := s.Mean()
		if stdDev := s.SampleStdDev(); stdDev > 0 {
			diff := value - mean
			return 1.0 / (gaussianNormalConstant * stdDev) * math.Exp(-(diff * diff / (2.0 * stdDev * stdDev)))
		} else if value == mean {
			return 1.0
		}
	}
	return 0.0
}

// Estimate estimates weight boundaries for a given value
func (s *NumSeries) Estimate(value float64) (lessThan float64, equalTo float64, greaterThan float64) {
	equalTo = s.ProbDensity(value) * s.TotalWeight()

	mean := s.Mean()
	if stdDev := s.SampleStdDev(); stdDev > 0 {
		lessThan = calc.NormProb((value-mean)/stdDev)*s.weight - equalTo
	} else if value < mean {
		lessThan = s.weight - equalTo
	}

	if greaterThan = s.weight - equalTo - lessThan; greaterThan < 0 {
		greaterThan = 0
	}
	return
}
