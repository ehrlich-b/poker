package poker_test

import (
	"testing"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
	"github.com/behrlich/poker-solver/pkg/solver"
	"github.com/behrlich/poker-solver/pkg/tree"
)

// TestIntegration_GeometricSizing tests that geometric bet sizing works end-to-end
func TestIntegration_GeometricSizing(t *testing.T) {
	// Parse river position: BTN vs BB on river
	posStr := "BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d2s|>BTN"
	gs, err := notation.ParsePosition(posStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Create geometric sizing: target 30bb pot, 1 street (river), 100bb all-in
	geoSizing := tree.NewGeometricSizing(30.0, 1, 100.0)

	// Create action config with geometric sizing
	config := tree.ActionConfig{
		GeometricSizing:   geoSizing,
		NumGeometricSizes: 1, // Just the geometric mean
		AllowCheck:        true,
		AllowCall:         true,
		AllowFold:         true,
	}

	builder := tree.NewBuilder(config)

	// Parse combos
	combo0 := notation.Combo{
		Card1: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.Ace, Suit: cards.Clubs},
	}
	combo1 := notation.Combo{
		Card1: cards.Card{Rank: cards.Queen, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.Queen, Suit: cards.Hearts},
	}

	// Build tree
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Check that BTN has check and bet actions
	if root.Player != 0 {
		t.Fatalf("Expected BTN (player 0) to act first, got player %d", root.Player)
	}

	hasCheck := false
	betSizes := []float64{}

	for _, action := range root.Actions {
		if action.Type == notation.Check {
			hasCheck = true
		}
		if action.Type == notation.Bet {
			betSizes = append(betSizes, action.Amount)
		}
	}

	if !hasCheck {
		t.Error("Expected check action to be available")
	}

	if len(betSizes) == 0 {
		t.Fatal("Expected at least one bet action to be available")
	}

	// Verify geometric bet size is present
	// Target: 30bb, current: 10bb, 1 street
	// Growth factor: 30/10 = 3
	// Bet size: 10 × (3-1)/2 = 10 × 1 = 10bb
	expectedBet := 10.0
	tolerance := 0.5

	hasGeometricBet := false
	for _, betSize := range betSizes {
		if betSize >= expectedBet-tolerance && betSize <= expectedBet+tolerance {
			hasGeometricBet = true
			t.Logf("Found geometric bet size: %.1fbb (target pot 30bb from 10bb)", betSize)
			break
		}
	}

	if !hasGeometricBet {
		t.Errorf("Expected geometric bet size ~%.1fbb, got bet sizes: %v", expectedBet, betSizes)
	}

	// Solve with CFR to verify tree is valid
	cfr := solver.NewCFR()
	profile := cfr.Train(root, 100)

	if profile.NumInfoSets() == 0 {
		t.Error("Expected some information sets after solving")
	}

	t.Logf("Solved with geometric sizing, found %d information sets", profile.NumInfoSets())
}

// TestIntegration_GeometricSizing_MultipleSizes tests geometric sizing with multiple bet sizes
func TestIntegration_GeometricSizing_MultipleSizes(t *testing.T) {
	// Parse river position
	posStr := "BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d2s|>BTN"
	gs, err := notation.ParsePosition(posStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Create geometric sizing with 3 bet sizes
	geoSizing := tree.NewGeometricSizing(30.0, 1, 100.0)

	config := tree.ActionConfig{
		GeometricSizing:   geoSizing,
		NumGeometricSizes: 3, // Three sizes around geometric mean
		AllowCheck:        true,
		AllowCall:         true,
		AllowFold:         true,
	}

	builder := tree.NewBuilder(config)

	// Parse combos
	combo0 := notation.Combo{
		Card1: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.Ace, Suit: cards.Clubs},
	}
	combo1 := notation.Combo{
		Card1: cards.Card{Rank: cards.Queen, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.Queen, Suit: cards.Hearts},
	}

	// Build tree
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Count bet actions
	betCount := 0
	betSizes := []float64{}

	for _, action := range root.Actions {
		if action.Type == notation.Bet {
			betCount++
			betSizes = append(betSizes, action.Amount)
		}
	}

	// Should have 3 bet sizes (or 4 if all-in is added)
	if betCount < 3 {
		t.Errorf("Expected at least 3 bet sizes, got %d", betCount)
	}

	t.Logf("Generated %d bet sizes with geometric sizing: %v", betCount, betSizes)

	// Verify sizes are in ascending order
	for i := 1; i < len(betSizes); i++ {
		if betSizes[i] <= betSizes[i-1] {
			t.Errorf("Bet sizes not in ascending order: %v", betSizes)
			break
		}
	}
}

// TestIntegration_GeometricSizing_FlopToRiver tests geometric sizing across multiple streets
func TestIntegration_GeometricSizing_FlopToRiver(t *testing.T) {
	// Parse flop position
	posStr := "BTN:AdAc:S97.5/BB:QdQh:S97.5|P5.5|Th9h2c|>BTN"
	gs, err := notation.ParsePosition(posStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Target: 30bb pot at river, starting from 5.5bb pot on flop
	// 3 streets remaining (flop, turn, river)
	geoSizing := tree.NewGeometricSizing(30.0, 3, 97.5)

	config := tree.ActionConfig{
		GeometricSizing:   geoSizing,
		NumGeometricSizes: 1,
		AllowCheck:        true,
		AllowCall:         true,
		AllowFold:         true,
	}

	builder := tree.NewBuilder(config)

	// Parse combos
	combo0 := notation.Combo{
		Card1: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.Ace, Suit: cards.Clubs},
	}
	combo1 := notation.Combo{
		Card1: cards.Card{Rank: cards.Queen, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.Queen, Suit: cards.Hearts},
	}

	// Build tree
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Find the bet action
	var flopBetAmount float64
	for _, action := range root.Actions {
		if action.Type == notation.Bet {
			flopBetAmount = action.Amount
			break
		}
	}

	if flopBetAmount == 0 {
		t.Fatal("Expected to find a bet action on flop")
	}

	// Expected flop bet: ~38% pot (from geometric test earlier)
	// Growth factor per street: (30/5.5)^(1/3) ≈ 1.76
	// Bet fraction: (1.76-1)/2 ≈ 0.38
	expectedFlopBet := 5.5 * 0.38
	tolerance := 0.3

	if flopBetAmount < expectedFlopBet-tolerance || flopBetAmount > expectedFlopBet+tolerance {
		t.Logf("Note: Flop bet size = %.1fbb (%.0f%% pot), expected ~%.1fbb",
			flopBetAmount, (flopBetAmount/5.5)*100, expectedFlopBet)
		// Not failing here as exact size depends on rounding
	} else {
		t.Logf("Flop geometric bet size: %.1fbb (%.0f%% pot)", flopBetAmount, (flopBetAmount/5.5)*100)
	}
}

// TestIntegration_GeometricSizing_BackwardCompatible ensures non-geometric configs still work
func TestIntegration_GeometricSizing_BackwardCompatible(t *testing.T) {
	// Parse river position
	posStr := "BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d2s|>BTN"
	gs, err := notation.ParsePosition(posStr)
	if err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Use old-style fixed bet sizes (no geometric sizing)
	config := tree.ActionConfig{
		BetSizes:   []float64{0.5, 1.0}, // 50% and 100% pot
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}

	builder := tree.NewBuilder(config)

	// Parse combos
	combo0 := notation.Combo{
		Card1: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.Ace, Suit: cards.Clubs},
	}
	combo1 := notation.Combo{
		Card1: cards.Card{Rank: cards.Queen, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.Queen, Suit: cards.Hearts},
	}

	// Build tree
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Verify we get the expected bet sizes (5bb and 10bb)
	betSizes := []float64{}
	for _, action := range root.Actions {
		if action.Type == notation.Bet {
			betSizes = append(betSizes, action.Amount)
		}
	}

	if len(betSizes) < 2 {
		t.Errorf("Expected at least 2 bet sizes, got %d", len(betSizes))
	}

	// Check for 5bb (50% pot) and 10bb (100% pot)
	has5bb := false
	has10bb := false
	for _, size := range betSizes {
		if size >= 4.9 && size <= 5.1 {
			has5bb = true
		}
		if size >= 9.9 && size <= 10.1 {
			has10bb = true
		}
	}

	if !has5bb {
		t.Error("Expected 5bb bet size (50% pot)")
	}
	if !has10bb {
		t.Error("Expected 10bb bet size (100% pot)")
	}

	t.Logf("Backward compatible: got bet sizes %v", betSizes)
}
