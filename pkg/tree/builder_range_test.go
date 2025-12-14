package tree

import (
	"testing"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
)

func TestBuilder_BuildRange_Simple(t *testing.T) {
	// Set up game state
	board, err := cards.ParseCards("Kh9s4c7d2s")
	if err != nil {
		t.Fatalf("Failed to parse board: %v", err)
	}

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
	range0, err := notation.ParseRange("AA")
	if err != nil {
		t.Fatalf("Failed to parse range0: %v", err)
	}

	range1, err := notation.ParseRange("KK")
	if err != nil {
		t.Fatalf("Failed to parse range1: %v", err)
	}

	// Build tree
	builder := NewBuilder(DefaultRiverConfig())
	root, err := builder.BuildRange(gs, range0, range1)
	if err != nil {
		t.Fatalf("BuildRange failed: %v", err)
	}

	// Verify root is a chance node
	if !root.IsChance {
		t.Errorf("Root should be a chance node")
	}

	if root.IsTerminal {
		t.Errorf("Root should not be terminal")
	}

	// Verify there are children for valid combo pairs
	// AA has 6 combos, KK has 6 combos
	// But some may conflict with board (Kh9s4c7d2s has Kh, 9s, 4c, 7d, 2s)
	// AA combos that don't use any board cards: all 6
	// KK combos that don't use Kh: 5 (excluding any with Kh)
	// Actually KK combos: KsKd, KsKc, KdKc (3 combos not using Kh)
	expectedCombos := 6 * 3 // 6 AA combos × 3 KK combos (not using Kh)

	if len(root.Children) != expectedCombos {
		t.Errorf("Expected %d children, got %d", expectedCombos, len(root.Children))
	}

	// Verify probabilities sum to 1.0
	totalProb := 0.0
	for _, prob := range root.ChanceProbabilities {
		totalProb += prob
	}
	if totalProb < 0.999 || totalProb > 1.001 {
		t.Errorf("Probabilities should sum to 1.0, got %.6f", totalProb)
	}

	// Verify each probability is uniform
	expectedProb := 1.0 / float64(expectedCombos)
	for key, prob := range root.ChanceProbabilities {
		if prob < expectedProb-0.0001 || prob > expectedProb+0.0001 {
			t.Errorf("Probability for %s should be %.6f, got %.6f", key, expectedProb, prob)
		}
	}
}

func TestBuilder_BuildRange_NoValidCombos(t *testing.T) {
	// Set up game state with board that blocks all combos
	board, err := cards.ParseCards("AhAdAcAsKh")
	if err != nil {
		t.Fatalf("Failed to parse board: %v", err)
	}

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:   10,
		Board: board,
		ToAct: 0,
	}

	// Parse ranges (all aces are on board, so AA range has no valid combos)
	range0, err := notation.ParseRange("AA")
	if err != nil {
		t.Fatalf("Failed to parse range0: %v", err)
	}

	range1, err := notation.ParseRange("KK")
	if err != nil {
		t.Fatalf("Failed to parse range1: %v", err)
	}

	// Build tree should fail
	builder := NewBuilder(DefaultRiverConfig())
	_, err = builder.BuildRange(gs, range0, range1)
	if err == nil {
		t.Errorf("Expected error for no valid combos, got nil")
	}
}

func TestBuilder_BuildRange_MultipleRanges(t *testing.T) {
	// Set up game state
	board, err := cards.ParseCards("Th9h2c5d8s")
	if err != nil {
		t.Fatalf("Failed to parse board: %v", err)
	}

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
	range0, err := notation.ParseRange("AA,KK")
	if err != nil {
		t.Fatalf("Failed to parse range0: %v", err)
	}

	range1, err := notation.ParseRange("QQ,JJ")
	if err != nil {
		t.Fatalf("Failed to parse range1: %v", err)
	}

	// Build tree
	builder := NewBuilder(DefaultRiverConfig())
	root, err := builder.BuildRange(gs, range0, range1)
	if err != nil {
		t.Fatalf("BuildRange failed: %v", err)
	}

	// Verify root is a chance node
	if !root.IsChance {
		t.Errorf("Root should be a chance node")
	}

	// AA: 6 combos, KK: 6 combos, QQ: 6 combos, JJ: 6 combos
	// All should be valid (no conflicts with board)
	expectedCombos := (6 + 6) * (6 + 6) // (AA + KK) × (QQ + JJ) = 12 × 12 = 144

	if len(root.Children) != expectedCombos {
		t.Errorf("Expected %d children, got %d", expectedCombos, len(root.Children))
	}

	// Verify each child is a valid game tree (decision node)
	for key, child := range root.Children {
		if child.IsTerminal {
			t.Errorf("Child %s should not be terminal immediately", key)
		}
		if child.IsChance {
			t.Errorf("Child %s should not be a chance node", key)
		}
		if child.Player != 0 && child.Player != 1 {
			t.Errorf("Child %s should have a valid player (0 or 1), got %d", key, child.Player)
		}
	}
}
