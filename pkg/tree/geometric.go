package tree

import (
	"fmt"
	"math"
)

// GeometricSizing calculates bet sizes that achieve geometric pot growth
// This is useful for multi-street scenarios where you want consistent pot growth
type GeometricSizing struct {
	// TargetPot is the desired pot size at the end (in BB)
	TargetPot float64

	// NumStreets is the number of betting rounds remaining (1-3)
	// Flop: 3 streets (flop, turn, river)
	// Turn: 2 streets (turn, river)
	// River: 1 street (river only)
	NumStreets int

	// AllIn stack size (in BB) - maximum bet is capped at this
	AllIn float64
}

// NewGeometricSizing creates a geometric sizing calculator
func NewGeometricSizing(targetPot float64, numStreets int, allIn float64) *GeometricSizing {
	return &GeometricSizing{
		TargetPot:  targetPot,
		NumStreets: numStreets,
		AllIn:      allIn,
	}
}

// CalculateBetSize calculates the bet size needed to achieve geometric growth
// Returns the bet size as a fraction of the current pot
//
// Example:
//   currentPot = 10bb, targetPot = 100bb, numStreets = 2
//   Growth factor G = (100/10)^(1/2) = sqrt(10) ≈ 3.16
//   Bet fraction F = (G - 1) / 2 = (3.16 - 1) / 2 ≈ 1.08 (108% pot)
func (g *GeometricSizing) CalculateBetSize(currentPot float64) float64 {
	if g.NumStreets <= 0 {
		return 0
	}

	if currentPot <= 0 {
		return 0
	}

	// Calculate growth factor needed per street
	// targetPot = currentPot × G^numStreets
	// G = (targetPot / currentPot)^(1/numStreets)
	growthFactor := math.Pow(g.TargetPot/currentPot, 1.0/float64(g.NumStreets))

	// Bet size fraction that achieves this growth (assuming opponent calls)
	// After bet+call: pot = currentPot × (1 + 2×fraction) = currentPot × growthFactor
	// 1 + 2×fraction = growthFactor
	// fraction = (growthFactor - 1) / 2
	betFraction := (growthFactor - 1.0) / 2.0

	// Cap at all-in
	if betFraction*currentPot > g.AllIn {
		betFraction = g.AllIn / currentPot
	}

	return betFraction
}

// CalculateBetSizes calculates multiple bet sizes around the geometric mean
// Returns a slice of bet fractions (e.g., [0.5×geo, geo, 1.5×geo])
// This gives solver flexibility while maintaining geometric structure
func (g *GeometricSizing) CalculateBetSizes(currentPot float64, numSizes int) []float64 {
	if numSizes <= 0 {
		return []float64{}
	}

	// Calculate geometric bet size
	geoSize := g.CalculateBetSize(currentPot)

	if numSizes == 1 {
		return []float64{geoSize}
	}

	// Create multiple sizes around the geometric mean
	// For numSizes=3: [0.66×geo, geo, 1.5×geo]
	sizes := make([]float64, numSizes)

	if numSizes == 2 {
		// Two sizes: 0.75× and 1.25× geometric
		sizes[0] = geoSize * 0.75
		sizes[1] = geoSize * 1.25
	} else if numSizes == 3 {
		// Three sizes: 0.66×, 1×, 1.5× geometric
		sizes[0] = geoSize * 0.66
		sizes[1] = geoSize
		sizes[2] = geoSize * 1.5
	} else {
		// General case: spread evenly around geometric mean
		// Range from 0.5× to 1.5× geometric
		for i := 0; i < numSizes; i++ {
			ratio := 0.5 + (1.0 * float64(i) / float64(numSizes-1))
			sizes[i] = geoSize * ratio
		}
	}

	// Cap all sizes at all-in
	for i := range sizes {
		if sizes[i]*currentPot > g.AllIn {
			sizes[i] = g.AllIn / currentPot
		}
	}

	return sizes
}

// Validate checks if the geometric sizing parameters are valid
func (g *GeometricSizing) Validate() error {
	if g.TargetPot <= 0 {
		return fmt.Errorf("target pot must be positive, got %.2f", g.TargetPot)
	}

	if g.NumStreets < 1 || g.NumStreets > 3 {
		return fmt.Errorf("numStreets must be 1-3, got %d", g.NumStreets)
	}

	if g.AllIn <= 0 {
		return fmt.Errorf("allIn must be positive, got %.2f", g.AllIn)
	}

	return nil
}

// String returns a human-readable description of the geometric sizing
func (g *GeometricSizing) String() string {
	return fmt.Sprintf("GeometricSizing{target=%.1fbb, streets=%d, allIn=%.1fbb}",
		g.TargetPot, g.NumStreets, g.AllIn)
}
