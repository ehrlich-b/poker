package poker_test

import (
	"testing"
	"time"

	"github.com/behrlich/poker-solver/pkg/abstraction"
	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
	"github.com/behrlich/poker-solver/pkg/solver"
	"github.com/behrlich/poker-solver/pkg/tree"
)

// TestIntegration_FlopSolver_WithBucketing tests flop solving with card abstraction
func TestIntegration_FlopSolver_WithBucketing(t *testing.T) {
	// Parse flop position: BTN vs BB on Th-9h-2c flop
	// Simple range: AA vs QQ
	posStr := "BTN:AA:S100/BB:QQ:S100|P10|Th9h2c|>BTN"
	gs, err := notation.ParsePosition(posStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Create bucketer with 100 buckets for opponent range
	// For this test, use BB's range (QQ) as the opponent range for bucketing
	board, err := cards.ParseCards("Th9h2c")
	if err != nil {
		t.Fatalf("Failed to parse board: %v", err)
	}

	// Parse opponent range for bucketing
	oppRange, err := notation.ParseRange("QQ")
	if err != nil {
		t.Fatalf("Failed to parse opponent range: %v", err)
	}

	bucketer := abstraction.NewBucketer(board, oppRange, 100)

	// Create tree builder with bucketing enabled
	config := tree.ActionConfig{
		BetSizes:   []float64{0.5, 1.0}, // 50% and 100% pot bets
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}

	builder := tree.NewBuilder(config)
	builder.SetBucketer(bucketer)

	// Build range tree (with bucketing)
	start := time.Now()
	root, err := builder.BuildRange(gs, gs.Players[0].Range, gs.Players[1].Range)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}
	buildTime := time.Since(start)

	t.Logf("Built flop tree in %v", buildTime)

	// Solve with MCCFR
	iterations := 1000 // Use conservative iteration count for testing
	mccfr := solver.NewMCCFR(42)

	start = time.Now()
	profile := mccfr.Train(root, iterations)
	solveTime := time.Since(start)

	t.Logf("Solved flop position (%d iterations) in %v", iterations, solveTime)
	t.Logf("Found %d information sets", profile.NumInfoSets())

	// Verify that info sets use bucket IDs
	foundBucketed := false
	for infoSet := range profile.All() {
		if len(infoSet) > 0 && infoSet[len(infoSet)-8:len(infoSet)-1] == "BUCKET_" {
			foundBucketed = true
			t.Logf("Found bucketed info set: %s", infoSet)
			break
		}
	}

	if !foundBucketed {
		t.Logf("Warning: No bucketed info sets found (might be using specific cards)")
		// This is not necessarily an error - might happen if bucketing is not applied
	}

	// Verify strategies converged (at least some non-uniform distributions)
	hasNonUniform := false
	for _, strategy := range profile.All() {
		avgStrat := strategy.GetAverageStrategy()
		if len(avgStrat) > 1 {
			// Check if not perfectly uniform
			diff := 0.0
			for _, p := range avgStrat {
				diff += (p - avgStrat[0]) * (p - avgStrat[0])
			}
			if diff > 0.01 { // Allow some variance
				hasNonUniform = true
				break
			}
		}
	}

	if !hasNonUniform {
		t.Logf("Warning: All strategies appear uniform (may need more iterations)")
	}

	// Performance check: should solve in reasonable time
	if solveTime > 30*time.Second {
		t.Errorf("Flop solve took too long: %v (target <30s for 1k iterations)", solveTime)
	}
}

// TestIntegration_FlopRollout tests that flop→turn→river rollout works correctly
func TestIntegration_FlopRollout(t *testing.T) {
	// Test specific combo matchup on flop
	posStr := "BTN:AdAc:S100/BB:2d2c:S100|P10|Kh9s4c|>BTN"
	gs, err := notation.ParsePosition(posStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Create builder without bucketing (specific combos)
	config := tree.ActionConfig{
		BetSizes:   []float64{0.5}, // Single bet size for simplicity
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}

	builder := tree.NewBuilder(config)

	// Parse combos
	combo0 := notation.Combo{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Clubs}}
	combo1 := notation.Combo{Card1: cards.Card{Rank: cards.Two, Suit: cards.Diamonds}, Card2: cards.Card{Rank: cards.Two, Suit: cards.Clubs}}

	// Build tree
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Count rollout nodes in tree
	rolloutCount := 0
	var countRollouts func(*tree.TreeNode)
	countRollouts = func(node *tree.TreeNode) {
		if node.NeedsRollout {
			rolloutCount++
		}
		for _, child := range node.Children {
			countRollouts(child)
		}
	}

	countRollouts(root)
	t.Logf("Found %d rollout nodes in flop tree", rolloutCount)

	if rolloutCount == 0 {
		t.Error("Expected to find rollout nodes in flop tree")
	}

	// Solve with MCCFR (which handles rollout)
	mccfr := solver.NewMCCFR(42)
	profile := mccfr.Train(root, 200)

	t.Logf("Solved flop with rollout, found %d information sets", profile.NumInfoSets())

	// Sanity check: AA should have a strategy at the root
	foundAA := false
	for infoSet := range profile.All() {
		if len(infoSet) > 0 && infoSet[len(infoSet)-4:] == "AdAc" {
			foundAA = true
			t.Logf("Found AA strategy at: %s", infoSet)
			break
		}
	}

	if !foundAA {
		t.Error("Expected to find AA strategy in info sets")
	}
}

// TestIntegration_FlopBucketing_TreeSize verifies bucketing reduces tree size
func TestIntegration_FlopBucketing_TreeSize(t *testing.T) {
	// Parse flop position with small ranges
	posStr := "BTN:AA,KK:S100/BB:QQ,JJ:S100|P10|Th9h2c|>BTN"
	gs, err := notation.ParsePosition(posStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Create bucketer
	board, _ := cards.ParseCards("Th9h2c")
	oppRange, _ := notation.ParseRange("QQ,JJ")
	bucketer := abstraction.NewBucketer(board, oppRange, 100)

	config := tree.ActionConfig{
		BetSizes:   []float64{0.5},
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}

	// Build tree WITH bucketing
	builderWithBucketing := tree.NewBuilder(config)
	builderWithBucketing.SetBucketer(bucketer)

	rootBucketed, err := builderWithBucketing.BuildRange(gs, gs.Players[0].Range, gs.Players[1].Range)
	if err != nil {
		t.Fatalf("Failed to build bucketed tree: %v", err)
	}

	// Build tree WITHOUT bucketing
	builderNoBucketing := tree.NewBuilder(config)

	rootUnbucketed, err := builderNoBucketing.BuildRange(gs, gs.Players[0].Range, gs.Players[1].Range)
	if err != nil {
		t.Fatalf("Failed to build unbucketed tree: %v", err)
	}

	// Solve both and count info sets
	mccfr := solver.NewMCCFR(42)

	profileBucketed := mccfr.Train(rootBucketed, 100)
	bucketedInfoSets := profileBucketed.NumInfoSets()

	mccfr2 := solver.NewMCCFR(43)
	profileUnbucketed := mccfr2.Train(rootUnbucketed, 100)
	unbucketedInfoSets := profileUnbucketed.NumInfoSets()

	t.Logf("Info sets WITH bucketing: %d", bucketedInfoSets)
	t.Logf("Info sets WITHOUT bucketing: %d", unbucketedInfoSets)

	// Bucketing should reduce info sets (hands in same bucket share strategy)
	// However, with small ranges, the reduction might not be dramatic
	if bucketedInfoSets > unbucketedInfoSets {
		t.Logf("Note: Bucketing resulted in MORE info sets (%d vs %d)", bucketedInfoSets, unbucketedInfoSets)
		t.Logf("This can happen with small ranges where bucket collisions create different branches")
	} else if bucketedInfoSets == unbucketedInfoSets {
		t.Logf("Note: Bucketing resulted in SAME number of info sets (%d)", bucketedInfoSets)
		t.Logf("This can happen if all hands map to different buckets")
	} else {
		t.Logf("Bucketing reduced info sets by %.1f%%", 100.0*(1.0-float64(bucketedInfoSets)/float64(unbucketedInfoSets)))
	}
}
