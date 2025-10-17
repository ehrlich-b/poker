package notation

import (
	"testing"

	"github.com/behrlich/poker-solver/pkg/cards"
)

func TestActionType_String(t *testing.T) {
	tests := []struct {
		action ActionType
		want   string
	}{
		{Check, "check"},
		{Call, "call"},
		{Bet, "bet"},
		{Raise, "raise"},
		{Fold, "fold"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("ActionType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAction_String(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   string
	}{
		{"check", Action{Type: Check}, "x"},
		{"call", Action{Type: Call}, "c"},
		{"fold", Action{Type: Fold}, "f"},
		{"bet 3.5bb", Action{Type: Bet, Amount: 3.5}, "b3.5"},
		{"raise to 9bb", Action{Type: Raise, Amount: 9.0}, "r9.0"},
		{"bet 10bb", Action{Type: Bet, Amount: 10.0}, "b10.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("Action.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStreet(t *testing.T) {
	tests := []struct {
		boardSize int
		want      Street
	}{
		{0, Preflop},
		{3, Flop},
		{4, Turn},
		{5, River},
	}

	for _, tt := range tests {
		t.Run(tt.want.String(), func(t *testing.T) {
			if got := GetStreet(tt.boardSize); got != tt.want {
				t.Errorf("GetStreet(%d) = %v, want %v", tt.boardSize, got, tt.want)
			}
		})
	}
}

func TestStreet_String(t *testing.T) {
	tests := []struct {
		street Street
		want   string
	}{
		{Preflop, "preflop"},
		{Flop, "flop"},
		{Turn, "turn"},
		{River, "river"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.street.String(); got != tt.want {
				t.Errorf("Street.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameState_Clone(t *testing.T) {
	// Create an original game state
	original := &GameState{
		Players: []PlayerRange{
			{
				Position: BTN,
				Range:    []Combo{{cards.NewCard(cards.Ace, cards.Spades), cards.NewCard(cards.King, cards.Spades)}},
				Stack:    100.0,
			},
			{
				Position: BB,
				Range:    []Combo{{cards.NewCard(cards.Queen, cards.Hearts), cards.NewCard(cards.Queen, cards.Diamonds)}},
				Stack:    98.0,
			},
		},
		Pot:   3.0,
		Board: []cards.Card{cards.NewCard(cards.King, cards.Hearts), cards.NewCard(cards.Nine, cards.Spades), cards.NewCard(cards.Four, cards.Clubs)},
		ActionHistory: []Action{
			{Type: Bet, Amount: 2.0},
			{Type: Call},
		},
		ToAct:  0,
		Street: Flop,
	}

	// Clone it
	clone := original.Clone()

	// Verify deep copy
	if clone == original {
		t.Error("Clone returned same pointer")
	}

	// Verify values match
	if len(clone.Players) != len(original.Players) {
		t.Errorf("Players count: got %d, want %d", len(clone.Players), len(original.Players))
	}

	if clone.Pot != original.Pot {
		t.Errorf("Pot: got %.1f, want %.1f", clone.Pot, original.Pot)
	}

	if len(clone.Board) != len(original.Board) {
		t.Errorf("Board size: got %d, want %d", len(clone.Board), len(original.Board))
	}

	if len(clone.ActionHistory) != len(original.ActionHistory) {
		t.Errorf("ActionHistory size: got %d, want %d", len(clone.ActionHistory), len(original.ActionHistory))
	}

	if clone.ToAct != original.ToAct {
		t.Errorf("ToAct: got %d, want %d", clone.ToAct, original.ToAct)
	}

	if clone.Street != original.Street {
		t.Errorf("Street: got %v, want %v", clone.Street, original.Street)
	}

	// Modify clone and verify original is unchanged
	clone.Pot = 999.0
	if original.Pot == 999.0 {
		t.Error("Modifying clone affected original (Pot)")
	}

	clone.ToAct = 1
	if original.ToAct == 1 {
		t.Error("Modifying clone affected original (ToAct)")
	}
}

func TestGameState_String(t *testing.T) {
	gs := &GameState{
		Players: []PlayerRange{
			{Position: BTN, Stack: 100.0},
			{Position: BB, Stack: 98.0},
		},
		Pot:    3.0,
		Board:  []cards.Card{cards.NewCard(cards.Ace, cards.Spades)},
		ToAct:  0,
		Street: Flop,
	}

	got := gs.String()
	// Just verify it doesn't panic and contains some key info
	if got == "" {
		t.Error("GameState.String() returned empty string")
	}
}
