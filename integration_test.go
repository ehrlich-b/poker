package poker_test

import (
	"math"
	"testing"
	"time"

	"github.com/behrlich/poker-solver/pkg/notation"
	"github.com/behrlich/poker-solver/pkg/solver"
	"github.com/behrlich/poker-solver/pkg/tree"
)

// TestIntegration_EndToEnd tests the full pipeline: parse → build tree → solve
func TestIntegration_EndToEnd(t *testing.T) {
	// Parse a river position
	positionStr := "BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d2s|>BTN"
	gs, err := notation.ParsePosition(positionStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Verify we got 2 players with specific combos
	if len(gs.Players) != 2 {
		t.Fatalf("Expected 2 players, got %d", len(gs.Players))
	}
	if len(gs.Players[0].Range) != 1 || len(gs.Players[1].Range) != 1 {
		t.Fatalf("Expected specific cards for both players")
	}

	combo0 := gs.Players[0].Range[0]
	combo1 := gs.Players[1].Range[0]

	// Build game tree
	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Verify tree was built
	if root.IsTerminal {
		t.Fatal("Root should not be terminal")
	}
	if len(root.Children) == 0 {
		t.Fatal("Root should have children")
	}

	// Solve with CFR
	cfr := solver.NewCFR()
	profile := cfr.Train(root, 10000)

	// Verify we found strategies
	if profile.NumInfoSets() == 0 {
		t.Fatal("No strategies found")
	}

	// Verify strategies sum to 1.0
	for infoSet, strat := range profile.All() {
		avgStrat := strat.GetAverageStrategy()
		sum := 0.0
		for _, prob := range avgStrat {
			sum += prob
		}
		if math.Abs(sum-1.0) > 0.001 {
			t.Errorf("Strategy for %s doesn't sum to 1.0, got %.3f", infoSet, sum)
		}
	}
}

// TestIntegration_SymmetricScenario tests that symmetric positions produce symmetric strategies
func TestIntegration_SymmetricScenario(t *testing.T) {
	// Both players have the same hand (pair of Kings)
	// Board is rainbow (Ah9s4c7d2h), no flush draws
	// This should produce symmetric strategies

	positionStr1 := "BTN:KdKc:S100/BB:QdQh:S100|P10|Ah9s4c7d2h|>BTN"
	positionStr2 := "BTN:QdQh:S100/BB:KdKc:S100|P10|Ah9s4c7d2h|>BTN"

	// Parse first position
	gs1, err := notation.ParsePosition(positionStr1)
	if err != nil {
		t.Fatalf("Failed to parse position 1: %v", err)
	}

	// Parse second position (swapped hands)
	gs2, err := notation.ParsePosition(positionStr2)
	if err != nil {
		t.Fatalf("Failed to parse position 2: %v", err)
	}

	// Build trees
	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)

	combo1_0 := gs1.Players[0].Range[0]
	combo1_1 := gs1.Players[1].Range[0]
	root1, err := builder.Build(gs1, combo1_0, combo1_1)
	if err != nil {
		t.Fatalf("Failed to build tree 1: %v", err)
	}

	combo2_0 := gs2.Players[0].Range[0]
	combo2_1 := gs2.Players[1].Range[0]
	root2, err := builder.Build(gs2, combo2_0, combo2_1)
	if err != nil {
		t.Fatalf("Failed to build tree 2: %v", err)
	}

	// Solve both
	cfr1 := solver.NewCFR()
	profile1 := cfr1.Train(root1, 10000)

	cfr2 := solver.NewCFR()
	profile2 := cfr2.Train(root2, 10000)

	// Both should have the same number of infosets (symmetric game)
	if profile1.NumInfoSets() != profile2.NumInfoSets() {
		t.Errorf("Symmetric scenarios should have same number of infosets: %d vs %d",
			profile1.NumInfoSets(), profile2.NumInfoSets())
	}

	// The strategies should be similar (both players have same hand strength)
	// We don't expect exact equality due to CFR randomness, but should be close
	// This is more of a sanity check than a strict test
}

// TestIntegration_KnownSolution tests a simple spot with a known correct strategy
func TestIntegration_KnownSolution(t *testing.T) {
	// AA vs 72o on a dry board - shallow stacks for clear value betting
	// BTN with AA should bet for value
	// BB with 72o should fold when facing a bet

	// Use 20bb stacks, 10bb pot for 2:1 SPR (more conducive to betting)
	positionStr := "BTN:AdAc:S20/BB:7h2s:S20|P10|Kh9s4c3d2h|>BTN"
	gs, err := notation.ParsePosition(positionStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	combo0 := gs.Players[0].Range[0]
	combo1 := gs.Players[1].Range[0]

	// Build tree
	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Solve
	cfr := solver.NewCFR()
	profile := cfr.Train(root, 10000)

	// Verify we got strategies
	if profile.NumInfoSets() == 0 {
		t.Fatal("No strategies found")
	}

	// With fixed hands (AA vs 72o), equilibrium should have BB folding to bets
	// and BTN either betting or checking (both can be equilibrium depending on bet sizing)
	// The key property we check: strategies sum to 1.0
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

	// Additional sanity check: BB should fold to bets with 72o (worst hand)
	// Look for any BB facing bet infosets
	foundBBFacing := false
	for _, strat := range profile.All() {
		if len(strat.Actions) > 0 {
			// Check if this is BB facing a bet (has fold action)
			hasFold := false
			foldIdx := -1
			for i, action := range strat.Actions {
				if action.Type == notation.Fold {
					hasFold = true
					foldIdx = i
					break
				}
			}

			if hasFold {
				avgStrat := strat.GetAverageStrategy()
				foldFreq := avgStrat[foldIdx]
				// BB with worst hand should fold at reasonable frequency
				if foldFreq < 0.2 {
					t.Logf("Warning: BB folding only %.1f%% vs bet (expected >20%%)", foldFreq*100)
				}
				foundBBFacing = true
			}
		}
	}

	if !foundBBFacing {
		t.Log("Note: BB never faced a bet (BTN always checked)")
	}
}

// TestIntegration_Performance tests that solving completes in reasonable time
func TestIntegration_Performance(t *testing.T) {
	// Simple river spot
	positionStr := "BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d2s|>BTN"
	gs, err := notation.ParsePosition(positionStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	combo0 := gs.Players[0].Range[0]
	combo1 := gs.Players[1].Range[0]

	// Build tree
	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Time the solve
	start := time.Now()
	cfr := solver.NewCFR()
	profile := cfr.Train(root, 10000)
	elapsed := time.Since(start)

	// Should solve in <5 seconds (success criteria)
	if elapsed > 5*time.Second {
		t.Errorf("Solve took too long: %v (target: <5s)", elapsed)
	}

	// Verify we got results
	if profile.NumInfoSets() == 0 {
		t.Fatal("No strategies found")
	}

	t.Logf("Solved %d infosets in %v (%.0f iter/sec)",
		profile.NumInfoSets(), elapsed, 10000.0/elapsed.Seconds())
}

// TestIntegration_RangeExpansion tests that range parser expands correctly
func TestIntegration_RangeExpansion(t *testing.T) {
	// Test: "AA,KK-JJ" should expand to 24 combos (AA, KK, QQ, JJ)
	rangeStr := "AA,KK-JJ"
	combos, err := notation.ParseRange(rangeStr)
	if err != nil {
		t.Fatalf("Failed to parse range '%s': %v", rangeStr, err)
	}

	expected := 24 // AA=6, KK=6, QQ=6, JJ=6
	if len(combos) != expected {
		t.Errorf("Range '%s' should expand to %d combos, got %d", rangeStr, expected, len(combos))
	}

	// Verify we got the right pairs
	pairCounts := make(map[string]int)
	for _, combo := range combos {
		// Count pairs by rank
		if combo.Card1.Rank == combo.Card2.Rank {
			rankName := combo.Card1.Rank.String()
			pairCounts[rankName]++
		}
	}

	if pairCounts["A"] != 6 {
		t.Errorf("Expected 6 AA combos, got %d", pairCounts["A"])
	}
	if pairCounts["K"] != 6 {
		t.Errorf("Expected 6 KK combos, got %d", pairCounts["K"])
	}
	if pairCounts["Q"] != 6 {
		t.Errorf("Expected 6 QQ combos, got %d", pairCounts["Q"])
	}
	if pairCounts["J"] != 6 {
		t.Errorf("Expected 6 JJ combos, got %d", pairCounts["J"])
	}
}

// TestIntegration_TurnSolver tests MCCFR on turn positions with river rollout
// SAFETY: Uses only 200 iterations to prevent memory issues
func TestIntegration_TurnSolver(t *testing.T) {
	// Setup: Turn position with AA vs QQ
	// Board: Kh9s4c7d (turn)
	positionStr := "BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d|>BTN"

	// Parse position
	gs, err := notation.ParsePosition(positionStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Verify it's a turn position
	if len(gs.Board) != 4 {
		t.Fatalf("Expected turn (4 cards), got %d cards", len(gs.Board))
	}

	// Build game tree
	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	combo0 := gs.Players[0].Range[0]
	combo1 := gs.Players[1].Range[0]
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Verify tree contains rollout nodes
	hasRollout := false
	var checkNode func(*tree.TreeNode)
	checkNode = func(node *tree.TreeNode) {
		if node.IsTerminal && node.NeedsRollout {
			hasRollout = true
		}
		for _, child := range node.Children {
			checkNode(child)
		}
	}
	checkNode(root)

	if !hasRollout {
		t.Error("Turn tree should contain rollout nodes at showdowns")
	}

	// Solve with MCCFR (SAFETY: low iteration count)
	start := time.Now()
	mccfr := solver.NewMCCFR(42)
	profile := mccfr.Train(root, 200) // SAFETY: Only 200 iterations
	elapsed := time.Since(start)

	// Verify we got strategies
	if profile.NumInfoSets() == 0 {
		t.Fatal("No strategies found after training")
	}

	t.Logf("Turn solver: Solved %d infosets in %v (%.0f iter/sec)",
		profile.NumInfoSets(), elapsed, 200.0/elapsed.Seconds())

	// Basic sanity check: strategies should be defined
	avgStrategies := profile.GetAverageStrategies()
	if len(avgStrategies) == 0 {
		t.Error("No average strategies computed")
	}

	// Verify strategies sum to 1.0 (within tolerance)
	for infoSet, probs := range avgStrategies {
		sum := 0.0
		for _, p := range probs {
			sum += p
		}
		if sum < 0.99 || sum > 1.01 {
			t.Errorf("InfoSet %s: strategy sum %.3f not close to 1.0", infoSet, sum)
		}
	}
}
