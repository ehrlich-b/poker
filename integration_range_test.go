package poker_test

import (
	"math"
	"testing"

	"github.com/behrlich/poker-solver/pkg/abstraction"
	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
	"github.com/behrlich/poker-solver/pkg/solver"
	"github.com/behrlich/poker-solver/pkg/tree"
)

// TestIntegration_RangeVsRange validates that range-vs-range solving works correctly
func TestIntegration_RangeVsRange_Simple(t *testing.T) {
	// Parse position with ranges
	// BTN has AA, BB has QQ
	// On board Kh9s4c7d2s (no pair conflicts)
	board, _ := cards.ParseCards("Kh9s4c7d2s")

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:   10,
		Board: board,
		ToAct: 0,
	}

	// Parse ranges
	range0, _ := notation.ParseRange("AA")
	range1, _ := notation.ParseRange("QQ")

	// Build tree with range root
	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	root, err := builder.BuildRange(gs, range0, range1)
	if err != nil {
		t.Fatalf("BuildRange failed: %v", err)
	}

	// Solve with CFR
	cfr := solver.NewCFR()
	profile := cfr.Train(root, 10000)

	// Verify strategies exist and sum to 1.0
	for infoSet, strat := range profile.All() {
		avgStrat := strat.GetAverageStrategy()
		sum := 0.0
		for _, prob := range avgStrat {
			sum += prob
			if prob < -0.001 || prob > 1.001 {
				t.Errorf("Invalid probability %.3f in infoset %s", prob, infoSet)
			}
		}
		if math.Abs(sum-1.0) > 0.001 {
			t.Errorf("Strategy for %s doesn't sum to 1.0, got %.3f", infoSet, sum)
		}
	}

	// Check that we have multiple information sets (for different combos)
	numInfoSets := len(profile.All())
	if numInfoSets == 0 {
		t.Errorf("Expected non-zero information sets")
	}

	t.Logf("Solved range-vs-range with %d information sets", numInfoSets)
}

// TestIntegration_RangeVsRange_MultipleHands tests with wider ranges
func TestIntegration_RangeVsRange_MultipleHands(t *testing.T) {
	// BTN has {AA, KK}, BB has {QQ, JJ}
	// This creates 144 combo pairs (12 × 12)
	board, _ := cards.ParseCards("Th9h2c5d8s")

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:   10,
		Board: board,
		ToAct: 0,
	}

	// Parse ranges
	range0, _ := notation.ParseRange("AA,KK")
	range1, _ := notation.ParseRange("QQ,JJ")

	// Build tree
	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	root, err := builder.BuildRange(gs, range0, range1)
	if err != nil {
		t.Fatalf("BuildRange failed: %v", err)
	}

	// Verify root has correct number of children
	expectedCombos := 144 // (6+6) × (6+6) = 12 × 12
	if len(root.Children) != expectedCombos {
		t.Errorf("Expected %d combo pairs, got %d", expectedCombos, len(root.Children))
	}

	// Solve with CFR (fewer iterations since tree is larger)
	cfr := solver.NewCFR()
	profile := cfr.Train(root, 5000)

	// Verify all strategies are valid
	for infoSet, strat := range profile.All() {
		avgStrat := strat.GetAverageStrategy()
		sum := 0.0
		for _, prob := range avgStrat {
			sum += prob
		}
		if math.Abs(sum-1.0) > 0.01 { // Slightly looser tolerance for larger tree
			t.Errorf("Strategy for %s doesn't sum to 1.0, got %.3f", infoSet, sum)
		}
	}

	numInfoSets := len(profile.All())
	t.Logf("Solved %d combo pairs, found %d information sets", expectedCombos, numInfoSets)

	// Check that we have info sets for different hands
	// BTN should have info sets for both AA and KK
	foundAA := false
	foundKK := false
	for infoSet := range profile.All() {
		// InfoSet format: "board|history|>player|cards"
		// Check if this is a BTN info set
		if len(infoSet) > 5 && infoSet[len(infoSet)-4] == 'A' && infoSet[len(infoSet)-2] == 'A' {
			foundAA = true
		}
		if len(infoSet) > 5 && infoSet[len(infoSet)-4] == 'K' && infoSet[len(infoSet)-2] == 'K' {
			foundKK = true
		}
	}

	if !foundAA {
		t.Errorf("Expected to find AA in information sets")
	}
	if !foundKK {
		t.Errorf("Expected to find KK in information sets")
	}
}

// TestIntegration_RangeVsRange_Performance ensures range solving is reasonably fast
func TestIntegration_RangeVsRange_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	board, _ := cards.ParseCards("Kh9s4c7d2s")

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:   10,
		Board: board,
		ToAct: 0,
	}

	// Small ranges for performance test
	range0, _ := notation.ParseRange("AA,KK")
	range1, _ := notation.ParseRange("QQ")

	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	root, err := builder.BuildRange(gs, range0, range1)
	if err != nil {
		t.Fatalf("BuildRange failed: %v", err)
	}

	// This should complete reasonably quickly
	cfr := solver.NewCFR()
	profile := cfr.Train(root, 10000)

	// Just verify it completed
	if len(profile.All()) == 0 {
		t.Errorf("Expected non-zero information sets")
	}

	t.Logf("Successfully solved range-vs-range scenario")
}

// TestIntegration_BucketedRangeOutput validates that bucketed range-vs-range output is correct
func TestIntegration_BucketedRangeOutput(t *testing.T) {
	// This test validates the fix for the "CBo" bug where bucketed range output
	// showed garbage like "CBo" instead of proper bucket IDs

	// Parse flop position with ranges
	board, _ := cards.ParseCards("Th9h2c")

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:   10,
		Board: board,
		ToAct: 0,
	}

	// Parse ranges
	range0, _ := notation.ParseRange("AA,KK")
	range1, _ := notation.ParseRange("QQ,JJ")

	// Create bucketer (100 buckets)
	oppRange := append([]notation.Combo{}, range1...)
	bucketer := abstraction.NewBucketer(board, oppRange, 100)

	// Build tree with bucketing
	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	builder.SetBucketer(bucketer)
	root, err := builder.BuildRange(gs, range0, range1)
	if err != nil {
		t.Fatalf("BuildRange with bucketing failed: %v", err)
	}

	// Solve with MCCFR (few iterations, just need to test output format)
	mccfr := solver.NewMCCFR(42)
	profile := mccfr.Train(root, 100)

	// Validate that info sets contain bucket IDs
	foundBucketedInfoSet := false
	for infoSet := range profile.All() {
		// InfoSet format with bucketing: "board|history|>player|BUCKET_35"
		if len(infoSet) > 7 {
			// Extract the cards part (after last |)
			lastPipe := -1
			for i := len(infoSet) - 1; i >= 0; i-- {
				if infoSet[i] == '|' {
					lastPipe = i
					break
				}
			}
			if lastPipe >= 0 {
				cardsPart := infoSet[lastPipe+1:]

				// Check if this is a bucketed info set
				if len(cardsPart) >= 7 && cardsPart[:7] == "BUCKET_" {
					foundBucketedInfoSet = true

					// Validate bucket ID format (should be BUCKET_NN where NN is a number)
					if len(cardsPart) < 8 {
						t.Errorf("Invalid bucket ID format: %s (too short)", cardsPart)
					}

					// Check that it doesn't contain garbage like "CBo"
					if cardsPart == "CBo" || cardsPart[:2] == "CB" {
						t.Errorf("Found garbage output 'CBo' instead of bucket ID in: %s", infoSet)
					}
				}
			}
		}
	}

	if !foundBucketedInfoSet {
		t.Errorf("Expected to find at least one bucketed info set (format: BUCKET_NN)")
	}

	t.Logf("Successfully validated bucketed range-vs-range output format")
}
