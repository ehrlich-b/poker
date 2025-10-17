package tree

import (
	"testing"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
)

func TestBuilder_Build_SimpleRiver(t *testing.T) {
	// Simple river scenario: BTN acts first, pot=10bb, stacks=100bb
	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot: 10,
		Board: []cards.Card{
			cards.NewCard(cards.King, cards.Hearts),
			cards.NewCard(cards.Nine, cards.Spades),
			cards.NewCard(cards.Four, cards.Clubs),
			cards.NewCard(cards.Seven, cards.Diamonds),
			cards.NewCard(cards.Two, cards.Spades),
		},
		ActionHistory: nil,
		ToAct:         0,
		Street:        notation.River,
	}

	combo0 := notation.Combo{
		Card1: cards.NewCard(cards.Ace, cards.Diamonds),
		Card2: cards.NewCard(cards.Ace, cards.Clubs),
	}
	combo1 := notation.Combo{
		Card1: cards.NewCard(cards.Queen, cards.Diamonds),
		Card2: cards.NewCard(cards.Queen, cards.Hearts),
	}

	config := ActionConfig{
		BetSizes:   []float64{0.5, 1.0},
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}

	builder := NewBuilder(config)
	root, err := builder.Build(gs, combo0, combo1)

	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if root == nil {
		t.Fatal("Build() returned nil root")
	}

	// Root should be a decision node for player 0
	if root.IsTerminal {
		t.Error("root should not be terminal")
	}

	if root.Player != 0 {
		t.Errorf("root player should be 0, got %d", root.Player)
	}

	if root.Pot != 10 {
		t.Errorf("root pot should be 10, got %.1f", root.Pot)
	}

	// Should have check and bet actions
	if len(root.Actions) < 2 {
		t.Errorf("expected at least 2 actions, got %d", len(root.Actions))
	}

	// Check should have children
	if len(root.Children) == 0 {
		t.Error("root should have children")
	}
}

func TestBuilder_Build_InvalidInputs(t *testing.T) {
	config := DefaultRiverConfig()
	builder := NewBuilder(config)

	combo0 := notation.Combo{
		Card1: cards.NewCard(cards.Ace, cards.Spades),
		Card2: cards.NewCard(cards.King, cards.Spades),
	}
	combo1 := notation.Combo{
		Card1: cards.NewCard(cards.Queen, cards.Diamonds),
		Card2: cards.NewCard(cards.Jack, cards.Diamonds),
	}

	// Test: not 2 players
	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
		},
		Pot:           10,
		Board:         makeRiverBoard(),
		ActionHistory: nil,
		ToAct:         0,
	}

	_, err := builder.Build(gs, combo0, combo1)
	if err == nil {
		t.Error("expected error for non-2-player game")
	}

	// Test: not postflop
	gs2 := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:           10,
		Board:         []cards.Card{}, // Empty board
		ActionHistory: nil,
		ToAct:         0,
	}

	_, err = builder.Build(gs2, combo0, combo1)
	if err == nil {
		t.Error("expected error for preflop game")
	}
}

func TestBuilder_Build_DuplicateCards(t *testing.T) {
	config := DefaultRiverConfig()
	builder := NewBuilder(config)

	board := makeRiverBoard()

	// Create combo with card that's on the board
	combo0 := notation.Combo{
		Card1: board[0], // Duplicate!
		Card2: cards.NewCard(cards.Ace, cards.Clubs),
	}
	combo1 := notation.Combo{
		Card1: cards.NewCard(cards.Queen, cards.Diamonds),
		Card2: cards.NewCard(cards.Jack, cards.Diamonds),
	}

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:           10,
		Board:         board,
		ActionHistory: nil,
		ToAct:         0,
	}

	_, err := builder.Build(gs, combo0, combo1)
	if err == nil {
		t.Error("expected error for duplicate cards")
	}
}

func TestBuilder_ShowdownPayoffs(t *testing.T) {
	config := DefaultRiverConfig()
	builder := NewBuilder(config)

	board := []cards.Card{
		cards.NewCard(cards.King, cards.Hearts),
		cards.NewCard(cards.King, cards.Spades),
		cards.NewCard(cards.King, cards.Diamonds),
		cards.NewCard(cards.Seven, cards.Clubs),
		cards.NewCard(cards.Two, cards.Spades),
	}

	// Player 0 has AA (loses to board KKK77)
	// Player 1 has QQ (also loses to board)
	combo0 := notation.Combo{
		Card1: cards.NewCard(cards.Ace, cards.Diamonds),
		Card2: cards.NewCard(cards.Ace, cards.Clubs),
	}
	combo1 := notation.Combo{
		Card1: cards.NewCard(cards.Queen, cards.Diamonds),
		Card2: cards.NewCard(cards.Queen, cards.Hearts),
	}

	// Calculate showdown payoffs
	payoffs := builder.calculateShowdownPayoffs(board, [2]notation.Combo{combo0, combo1}, 100)

	// Verify it's zero-sum (or sums to pot)
	sum := payoffs[0] + payoffs[1]
	if sum != 100 {
		t.Errorf("payoffs should sum to pot (100), got %.1f", sum)
	}

	// One player should win
	if payoffs[0] != 100 && payoffs[1] != 100 && payoffs[0] != 50 {
		// Either someone wins or it's a split
		t.Errorf("unexpected payoffs: [%.1f, %.1f]", payoffs[0], payoffs[1])
	}
}

func TestBuilder_FoldPayoffs(t *testing.T) {
	config := ActionConfig{
		BetSizes:   []float64{0.5},
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}
	builder := NewBuilder(config)

	board := makeRiverBoard()
	combo0 := notation.Combo{
		Card1: cards.NewCard(cards.Ace, cards.Diamonds),
		Card2: cards.NewCard(cards.Ace, cards.Clubs),
	}
	combo1 := notation.Combo{
		Card1: cards.NewCard(cards.Queen, cards.Diamonds),
		Card2: cards.NewCard(cards.Queen, cards.Hearts),
	}

	// Build tree where BTN bets and BB folds
	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:   10,
		Board: board,
		ActionHistory: []notation.Action{
			{Type: notation.Bet, Amount: 5},
			{Type: notation.Fold},
		},
		ToAct: 0, // Doesn't matter, should be terminal
	}

	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	// Should be terminal with payoff going to BTN
	if !root.IsTerminal {
		t.Error("expected terminal node after fold")
	}

	// BTN (player 0) should win the pot
	if root.Payoff[0] <= 0 {
		t.Errorf("expected BTN to win pot, got payoffs [%.1f, %.1f]", root.Payoff[0], root.Payoff[1])
	}
}

func TestBuilder_IsShowdown(t *testing.T) {
	config := DefaultRiverConfig()
	builder := NewBuilder(config)

	tests := []struct {
		name     string
		history  []notation.Action
		expected bool
	}{
		{
			name:     "both checked",
			history:  []notation.Action{{Type: notation.Check}, {Type: notation.Check}},
			expected: true,
		},
		{
			name:     "bet and call",
			history:  []notation.Action{{Type: notation.Bet, Amount: 10}, {Type: notation.Call}},
			expected: true,
		},
		{
			name:     "just one check",
			history:  []notation.Action{{Type: notation.Check}},
			expected: false,
		},
		{
			name:     "bet (not called yet)",
			history:  []notation.Action{{Type: notation.Bet, Amount: 10}},
			expected: false,
		},
		{
			name:     "empty history",
			history:  []notation.Action{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.isShowdown(tt.history)
			if got != tt.expected {
				t.Errorf("isShowdown() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuilder_GetCallAmount(t *testing.T) {
	config := DefaultRiverConfig()
	builder := NewBuilder(config)

	tests := []struct {
		name    string
		history []notation.Action
		stack   float64
		want    float64
	}{
		{
			name:    "empty history",
			history: []notation.Action{},
			stack:   100,
			want:    0,
		},
		{
			name:    "after bet",
			history: []notation.Action{{Type: notation.Bet, Amount: 10}},
			stack:   100,
			want:    10,
		},
		{
			name:    "after bet and check",
			history: []notation.Action{{Type: notation.Bet, Amount: 15}, {Type: notation.Check}},
			stack:   100,
			want:    15, // Still the bet amount
		},
		{
			name:    "capped by stack",
			history: []notation.Action{{Type: notation.Bet, Amount: 50}},
			stack:   20,
			want:    20, // Can only call stack amount
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.getCallAmount(tt.history, 20, tt.stack)
			if got != tt.want {
				t.Errorf("getCallAmount() = %.1f, want %.1f", got, tt.want)
			}
		})
	}
}

func TestBuilder_TreeStructure(t *testing.T) {
	// Test that tree structure is correct: check/bet → opponent actions → terminals
	config := ActionConfig{
		BetSizes:   []float64{0.5},
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}
	builder := NewBuilder(config)

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:           10,
		Board:         makeRiverBoard(),
		ActionHistory: nil,
		ToAct:         0,
	}

	combo0 := notation.Combo{
		Card1: cards.NewCard(cards.Ace, cards.Diamonds),
		Card2: cards.NewCard(cards.Ace, cards.Clubs),
	}
	combo1 := notation.Combo{
		Card1: cards.NewCard(cards.Queen, cards.Diamonds),
		Card2: cards.NewCard(cards.Queen, cards.Hearts),
	}

	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	// Check that root has check action
	checkAction := notation.Action{Type: notation.Check}
	checkKey := ActionKey(checkAction)
	checkChild, hasCheck := root.Children[checkKey]

	if !hasCheck {
		t.Fatal("root should have check action")
	}

	// Check child should be a decision node for player 1 (BB)
	if checkChild.IsTerminal {
		t.Error("check child should not be terminal (BB must act)")
	}

	if checkChild.Player != 1 {
		t.Errorf("check child should be for player 1, got %d", checkChild.Player)
	}

	// BB should be able to check (leading to showdown) or bet
	if len(checkChild.Actions) < 2 {
		t.Errorf("BB should have at least 2 actions, got %d", len(checkChild.Actions))
	}

	// If BB checks, should reach showdown (terminal)
	bbCheckKey := ActionKey(notation.Action{Type: notation.Check})
	if bbCheckChild, ok := checkChild.Children[bbCheckKey]; ok {
		if !bbCheckChild.IsTerminal {
			t.Error("after both checks, should be terminal (showdown)")
		}
	}
}

// Helper function to create a standard river board
func makeRiverBoard() []cards.Card {
	return []cards.Card{
		cards.NewCard(cards.King, cards.Hearts),
		cards.NewCard(cards.Nine, cards.Spades),
		cards.NewCard(cards.Four, cards.Clubs),
		cards.NewCard(cards.Seven, cards.Diamonds),
		cards.NewCard(cards.Two, cards.Spades),
	}
}
