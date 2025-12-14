package tree

import (
	"math"
	"testing"
)

func TestGeometricSizing_CalculateBetSize(t *testing.T) {
	tests := []struct {
		name       string
		targetPot  float64
		numStreets int
		allIn      float64
		currentPot float64
		wantMin    float64 // Expected minimum bet fraction
		wantMax    float64 // Expected maximum bet fraction
	}{
		{
			name:       "river: 10bb to 100bb in 1 street",
			targetPot:  100,
			numStreets: 1,
			allIn:      100,
			currentPot: 10,
			wantMin:    4.4, // (10 - 1) / 2 = 4.5, close enough
			wantMax:    4.6,
		},
		{
			name:       "turn: 10bb to 100bb in 2 streets",
			targetPot:  100,
			numStreets: 2,
			allIn:      100,
			currentPot: 10,
			wantMin:    1.0, // sqrt(10) ≈ 3.16, (3.16-1)/2 ≈ 1.08
			wantMax:    1.2,
		},
		{
			name:       "flop: 10bb to 100bb in 3 streets",
			targetPot:  100,
			numStreets: 3,
			allIn:      100,
			currentPot: 10,
			wantMin:    0.5, // 10^(1/3) ≈ 2.15, (2.15-1)/2 ≈ 0.58
			wantMax:    0.7,
		},
		{
			name:       "flop: 3bb to 30bb in 3 streets (typical preflop raiser)",
			targetPot:  30,
			numStreets: 3,
			allIn:      100,
			currentPot: 3,
			wantMin:    0.5,
			wantMax:    0.7,
		},
		{
			name:       "all-in capped",
			targetPot:  1000,
			numStreets: 1,
			allIn:      50, // Can only bet 50bb
			currentPot: 10,
			wantMin:    4.9, // Capped at 50/10 = 5.0
			wantMax:    5.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGeometricSizing(tt.targetPot, tt.numStreets, tt.allIn)
			betFraction := gs.CalculateBetSize(tt.currentPot)

			if betFraction < tt.wantMin || betFraction > tt.wantMax {
				t.Errorf("CalculateBetSize() = %.3f, want between %.3f and %.3f",
					betFraction, tt.wantMin, tt.wantMax)
			}

			// Verify pot growth (only if not capped by all-in)
			betAmount := betFraction * tt.currentPot
			expectedGrowth := math.Pow(tt.targetPot/tt.currentPot, 1.0/float64(tt.numStreets))
			expectedBet := tt.currentPot * (expectedGrowth - 1.0) / 2.0

			// Only verify growth if bet wasn't capped
			if expectedBet <= tt.allIn {
				newPot := tt.currentPot + 2*betAmount
				growthFactor := newPot / tt.currentPot
				tolerance := 0.01

				if math.Abs(growthFactor-expectedGrowth) > tolerance {
					t.Errorf("Growth factor = %.3f, want %.3f", growthFactor, expectedGrowth)
				}
			}
		})
	}
}

func TestGeometricSizing_CalculateBetSizes(t *testing.T) {
	tests := []struct {
		name       string
		targetPot  float64
		numStreets int
		allIn      float64
		currentPot float64
		numSizes   int
		wantLen    int
	}{
		{
			name:       "single size",
			targetPot:  100,
			numStreets: 2,
			allIn:      100,
			currentPot: 10,
			numSizes:   1,
			wantLen:    1,
		},
		{
			name:       "two sizes",
			targetPot:  100,
			numStreets: 2,
			allIn:      100,
			currentPot: 10,
			numSizes:   2,
			wantLen:    2,
		},
		{
			name:       "three sizes",
			targetPot:  100,
			numStreets: 2,
			allIn:      100,
			currentPot: 10,
			numSizes:   3,
			wantLen:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGeometricSizing(tt.targetPot, tt.numStreets, tt.allIn)
			sizes := gs.CalculateBetSizes(tt.currentPot, tt.numSizes)

			if len(sizes) != tt.wantLen {
				t.Errorf("CalculateBetSizes() returned %d sizes, want %d", len(sizes), tt.wantLen)
			}

			// Verify sizes are in ascending order
			for i := 1; i < len(sizes); i++ {
				if sizes[i] <= sizes[i-1] {
					t.Errorf("Sizes not in ascending order: %v", sizes)
					break
				}
			}

			// Verify middle size is close to geometric mean (for 3 sizes)
			if tt.numSizes == 3 {
				geoSize := gs.CalculateBetSize(tt.currentPot)
				middleSize := sizes[1]
				tolerance := 0.01

				if math.Abs(middleSize-geoSize) > tolerance {
					t.Errorf("Middle size %.3f not close to geometric size %.3f", middleSize, geoSize)
				}
			}
		})
	}
}

func TestGeometricSizing_Validate(t *testing.T) {
	tests := []struct {
		name       string
		targetPot  float64
		numStreets int
		allIn      float64
		wantError  bool
	}{
		{
			name:       "valid",
			targetPot:  100,
			numStreets: 2,
			allIn:      100,
			wantError:  false,
		},
		{
			name:       "invalid: negative target pot",
			targetPot:  -10,
			numStreets: 2,
			allIn:      100,
			wantError:  true,
		},
		{
			name:       "invalid: zero target pot",
			targetPot:  0,
			numStreets: 2,
			allIn:      100,
			wantError:  true,
		},
		{
			name:       "invalid: zero streets",
			targetPot:  100,
			numStreets: 0,
			allIn:      100,
			wantError:  true,
		},
		{
			name:       "invalid: too many streets",
			targetPot:  100,
			numStreets: 4,
			allIn:      100,
			wantError:  true,
		},
		{
			name:       "invalid: negative all-in",
			targetPot:  100,
			numStreets: 2,
			allIn:      -10,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGeometricSizing(tt.targetPot, tt.numStreets, tt.allIn)
			err := gs.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestGeometricSizing_String(t *testing.T) {
	gs := NewGeometricSizing(100, 2, 100)
	str := gs.String()

	if str == "" {
		t.Error("String() returned empty string")
	}

	t.Logf("GeometricSizing.String() = %s", str)
}

// TestGeometricSizing_RealWorldScenario tests a realistic poker scenario
func TestGeometricSizing_RealWorldScenario(t *testing.T) {
	// Scenario: BTN raises 2.5bb pre, BB calls
	// Pot: 5.5bb (including blinds)
	// Stacks: 97.5bb each
	// Want pot to be ~30bb at river for a pot-sized bet

	targetPot := 30.0
	numStreets := 3 // flop, turn, river
	allIn := 97.5
	currentPot := 5.5

	gs := NewGeometricSizing(targetPot, numStreets, allIn)

	// Calculate bet sizes for each street
	flopPot := currentPot
	flopBetFraction := gs.CalculateBetSize(flopPot)
	flopBetAmount := flopBetFraction * flopPot

	t.Logf("Flop: pot=%.1fbb, bet=%.1fbb (%.0f%% pot)", flopPot, flopBetAmount, flopBetFraction*100)

	// After flop bet+call
	turnPot := flopPot + 2*flopBetAmount
	gs.NumStreets = 2
	turnBetFraction := gs.CalculateBetSize(turnPot)
	turnBetAmount := turnBetFraction * turnPot

	t.Logf("Turn: pot=%.1fbb, bet=%.1fbb (%.0f%% pot)", turnPot, turnBetAmount, turnBetFraction*100)

	// After turn bet+call
	riverPot := turnPot + 2*turnBetAmount
	gs.NumStreets = 1
	riverBetFraction := gs.CalculateBetSize(riverPot)
	riverBetAmount := riverBetFraction * riverPot

	t.Logf("River: pot=%.1fbb, bet=%.1fbb (%.0f%% pot)", riverPot, riverBetAmount, riverBetFraction*100)

	// Final pot after river bet+call
	finalPot := riverPot + 2*riverBetAmount

	t.Logf("Final pot: %.1fbb (target was %.1fbb)", finalPot, targetPot)

	// Verify we hit target (within 5% tolerance)
	tolerance := targetPot * 0.05
	if math.Abs(finalPot-targetPot) > tolerance {
		t.Errorf("Final pot %.1fbb differs from target %.1fbb by more than %.1fbb",
			finalPot, targetPot, tolerance)
	}
}
