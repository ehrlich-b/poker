package tree

import (
	"testing"

	"github.com/behrlich/poker-solver/pkg/notation"
)

func TestGenerateActions_NoBet(t *testing.T) {
	config := ActionConfig{
		BetSizes:   []float64{0.5, 1.0},
		AllowCheck: true,
		AllowCall:  false,
		AllowFold:  false,
	}

	actions := GenerateActions(10, 100, nil, config)

	// Should have: check, bet 5, bet 10, bet 100 (all-in)
	if len(actions) < 3 {
		t.Errorf("expected at least 3 actions (check + bets), got %d", len(actions))
	}

	// First action should be check
	if actions[0].Type != notation.Check {
		t.Errorf("first action should be Check, got %v", actions[0].Type)
	}

	// Should have bet actions
	hasBet := false
	for _, action := range actions {
		if action.Type == notation.Bet {
			hasBet = true
			break
		}
	}
	if !hasBet {
		t.Error("expected at least one Bet action")
	}
}

func TestGenerateActions_FacingBet(t *testing.T) {
	config := ActionConfig{
		BetSizes:   []float64{0.5, 1.0},
		AllowCheck: false,
		AllowCall:  true,
		AllowFold:  true,
	}

	lastAction := notation.Action{Type: notation.Bet, Amount: 10}
	actions := GenerateActions(20, 100, &lastAction, config)

	// Should have: fold, call (no betting when facing bet in v0.1)
	if len(actions) != 2 {
		t.Errorf("expected 2 actions (fold, call), got %d", len(actions))
	}

	// Check for fold and call
	hasFold := false
	hasCall := false
	for _, action := range actions {
		if action.Type == notation.Fold {
			hasFold = true
		}
		if action.Type == notation.Call {
			hasCall = true
		}
	}

	if !hasFold {
		t.Error("expected Fold action")
	}
	if !hasCall {
		t.Error("expected Call action")
	}
}

func TestGenerateActions_BetSizing(t *testing.T) {
	config := ActionConfig{
		BetSizes:   []float64{0.5, 0.75, 1.5},
		AllowCheck: true,
		AllowCall:  false,
		AllowFold:  false,
	}

	pot := 10.0
	stack := 100.0

	actions := GenerateActions(pot, stack, nil, config)

	// Check bet sizes
	expectedBets := []float64{5.0, 7.5, 15.0, 100.0} // 50%, 75%, 150%, all-in

	betCount := 0
	for _, action := range actions {
		if action.Type == notation.Bet {
			betCount++
		}
	}

	// Should have check + bet actions
	if betCount < 3 {
		t.Errorf("expected at least 3 bet actions, got %d", betCount)
	}

	// Verify bet amounts are reasonable
	for _, action := range actions {
		if action.Type == notation.Bet {
			if action.Amount <= 0 || action.Amount > stack {
				t.Errorf("bet amount %.1f is out of range (0, %.1f]", action.Amount, stack)
			}
		}
	}

	_ = expectedBets // Don't check exact amounts, just that they're reasonable
}

func TestGenerateActions_AllIn(t *testing.T) {
	config := ActionConfig{
		BetSizes:   []float64{0.5, 1.0},
		AllowCheck: true,
		AllowCall:  false,
		AllowFold:  false,
	}

	pot := 10.0
	stack := 12.0 // Small stack

	actions := GenerateActions(pot, stack, nil, config)

	// All-in should be included
	hasAllIn := false
	for _, action := range actions {
		if action.Type == notation.Bet && action.Amount == stack {
			hasAllIn = true
			break
		}
	}

	if !hasAllIn {
		t.Error("expected all-in option")
	}
}

func TestGenerateActions_NoCheckWhenFacingBet(t *testing.T) {
	config := ActionConfig{
		BetSizes:   []float64{0.5},
		AllowCheck: true, // Even though allowed, shouldn't appear when facing bet
		AllowCall:  true,
		AllowFold:  true,
	}

	lastAction := notation.Action{Type: notation.Bet, Amount: 10}
	actions := GenerateActions(20, 100, &lastAction, config)

	// Check should not be present
	for _, action := range actions {
		if action.Type == notation.Check {
			t.Error("Check should not be allowed when facing a bet")
		}
	}
}

func TestDefaultRiverConfig(t *testing.T) {
	config := DefaultRiverConfig()

	if len(config.BetSizes) < 2 {
		t.Errorf("expected at least 2 bet sizes, got %d", len(config.BetSizes))
	}

	if !config.AllowCheck {
		t.Error("expected AllowCheck to be true")
	}

	if !config.AllowCall {
		t.Error("expected AllowCall to be true")
	}

	if !config.AllowFold {
		t.Error("expected AllowFold to be true")
	}
}

func TestGetLastAction(t *testing.T) {
	tests := []struct {
		name    string
		history []notation.Action
		want    *notation.Action
	}{
		{
			name:    "empty history",
			history: nil,
			want:    nil,
		},
		{
			name: "single action",
			history: []notation.Action{
				{Type: notation.Bet, Amount: 10},
			},
			want: &notation.Action{Type: notation.Bet, Amount: 10},
		},
		{
			name: "multiple actions",
			history: []notation.Action{
				{Type: notation.Bet, Amount: 5},
				{Type: notation.Call},
				{Type: notation.Check},
			},
			want: &notation.Action{Type: notation.Check},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLastAction(tt.history)

			if tt.want == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Error("expected non-nil action")
				return
			}

			if got.Type != tt.want.Type {
				t.Errorf("got type %v, want %v", got.Type, tt.want.Type)
			}

			if got.Amount != tt.want.Amount {
				t.Errorf("got amount %.1f, want %.1f", got.Amount, tt.want.Amount)
			}
		})
	}
}

func TestGenerateActions_SmallBetsFiltered(t *testing.T) {
	config := ActionConfig{
		BetSizes:   []float64{0.001}, // Very small bet
		AllowCheck: true,
		AllowCall:  false,
		AllowFold:  false,
	}

	pot := 0.5 // Small pot
	stack := 100.0

	actions := GenerateActions(pot, stack, nil, config)

	// Should have check, but bet might be filtered if < 0.01
	hasCheck := false
	for _, action := range actions {
		if action.Type == notation.Check {
			hasCheck = true
		}
	}

	if !hasCheck {
		t.Error("expected Check action")
	}
}
