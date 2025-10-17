package notation

import (
	"testing"

	"github.com/behrlich/poker-solver/pkg/cards"
)

func TestParsePosition_SimpleRiver(t *testing.T) {
	fen := "BTN:AsKd:S98/BB:QhQd:S97|P3|Th9h2c7d2s|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	// Check players
	if len(gs.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(gs.Players))
	}

	// Check BTN
	if gs.Players[0].Position != BTN {
		t.Errorf("expected BTN, got %v", gs.Players[0].Position)
	}
	if gs.Players[0].Stack != 98 {
		t.Errorf("expected stack 98, got %.1f", gs.Players[0].Stack)
	}
	if len(gs.Players[0].Range) != 1 {
		t.Errorf("expected 1 combo, got %d", len(gs.Players[0].Range))
	}

	// Check BB
	if gs.Players[1].Position != BB {
		t.Errorf("expected BB, got %v", gs.Players[1].Position)
	}

	// Check pot
	if gs.Pot != 3 {
		t.Errorf("expected pot 3, got %.1f", gs.Pot)
	}

	// Check board (river = 5 cards)
	if len(gs.Board) != 5 {
		t.Errorf("expected 5 board cards, got %d", len(gs.Board))
	}

	// Check street
	if gs.Street != River {
		t.Errorf("expected River, got %v", gs.Street)
	}

	// Check ToAct
	if gs.ToAct != 0 {
		t.Errorf("expected ToAct 0 (BTN), got %d", gs.ToAct)
	}
}

func TestParsePosition_WithRange(t *testing.T) {
	fen := "BTN:AA,KK,AKs:S100/BB:QQ-JJ:S100|P20|Kh9s4c7d2s|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	// BTN should have 6+6+4=16 combos
	expectedBTNCombos := 6 + 6 + 4 // AA=6, KK=6, AKs=4
	if len(gs.Players[0].Range) != expectedBTNCombos {
		t.Errorf("expected %d BTN combos, got %d", expectedBTNCombos, len(gs.Players[0].Range))
	}

	// BB should have 6+6=12 combos (QQ=6, JJ=6)
	expectedBBCombos := 6 + 6
	if len(gs.Players[1].Range) != expectedBBCombos {
		t.Errorf("expected %d BB combos, got %d", expectedBBCombos, len(gs.Players[1].Range))
	}

	if gs.Pot != 20 {
		t.Errorf("expected pot 20, got %.1f", gs.Pot)
	}
}

func TestParsePosition_WithUnknownRange(t *testing.T) {
	fen := "BTN:AsKd:S100/BB:??:S100|P3|Th9h2c|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	// BTN has specific cards
	if len(gs.Players[0].Range) != 1 {
		t.Errorf("expected 1 combo for BTN, got %d", len(gs.Players[0].Range))
	}

	// BB has unknown range (??)
	if gs.Players[1].Range != nil && len(gs.Players[1].Range) != 0 {
		t.Errorf("expected nil or empty range for BB, got %d combos", len(gs.Players[1].Range))
	}

	// Check flop (3 cards)
	if len(gs.Board) != 3 {
		t.Errorf("expected 3 board cards (flop), got %d", len(gs.Board))
	}

	if gs.Street != Flop {
		t.Errorf("expected Flop, got %v", gs.Street)
	}
}

func TestParsePosition_Turn(t *testing.T) {
	fen := "BTN:AA:S100/BB:KK:S100|P10|Ah7h3c/5s|>BB"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	// Check board (turn = 4 cards)
	if len(gs.Board) != 4 {
		t.Errorf("expected 4 board cards (turn), got %d", len(gs.Board))
	}

	if gs.Street != Turn {
		t.Errorf("expected Turn, got %v", gs.Street)
	}

	// Check ToAct is BB (index 1)
	if gs.ToAct != 1 {
		t.Errorf("expected ToAct 1 (BB), got %d", gs.ToAct)
	}
}

func TestParsePosition_WithHistory(t *testing.T) {
	fen := "BTN:AA:S100/BB:KK:S100|P30|Kh9s4c7d2s|b10c|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	// Check action history: bet 10, call
	if len(gs.ActionHistory) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(gs.ActionHistory))
	}

	if gs.ActionHistory[0].Type != Bet {
		t.Errorf("expected first action to be Bet, got %v", gs.ActionHistory[0].Type)
	}
	if gs.ActionHistory[0].Amount != 10 {
		t.Errorf("expected bet amount 10, got %.1f", gs.ActionHistory[0].Amount)
	}

	if gs.ActionHistory[1].Type != Call {
		t.Errorf("expected second action to be Call, got %v", gs.ActionHistory[1].Type)
	}
}

func TestParsePosition_ComplexHistory(t *testing.T) {
	fen := "BTN:AA:S100/BB:KK:S100|P50|Kh9s4c7d2s|b15cr30c|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	// Check action history: bet 15, call, raise 30, call
	expectedActions := []struct {
		typ    ActionType
		amount float64
	}{
		{Bet, 15},
		{Call, 0},
		{Raise, 30},
		{Call, 0},
	}

	if len(gs.ActionHistory) != len(expectedActions) {
		t.Fatalf("expected %d actions, got %d", len(expectedActions), len(gs.ActionHistory))
	}

	for i, expected := range expectedActions {
		if gs.ActionHistory[i].Type != expected.typ {
			t.Errorf("action %d: expected type %v, got %v", i, expected.typ, gs.ActionHistory[i].Type)
		}
		if expected.typ == Bet || expected.typ == Raise {
			if gs.ActionHistory[i].Amount != expected.amount {
				t.Errorf("action %d: expected amount %.1f, got %.1f", i, expected.amount, gs.ActionHistory[i].Amount)
			}
		}
	}
}

func TestParsePosition_NoHistory(t *testing.T) {
	fen := "BTN:AA:S100/BB:KK:S100|P3|Kh9s4c|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	if len(gs.ActionHistory) != 0 {
		t.Errorf("expected empty action history, got %d actions", len(gs.ActionHistory))
	}
}

func TestParsePosition_DecimalAmounts(t *testing.T) {
	fen := "BTN:AA:S100/BB:KK:S100|P15.5|Kh9s4c7d2s|b3.5c|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	// Check pot with decimal
	if gs.Pot != 15.5 {
		t.Errorf("expected pot 15.5, got %.1f", gs.Pot)
	}

	// Check action amount with decimal
	if gs.ActionHistory[0].Amount != 3.5 {
		t.Errorf("expected bet amount 3.5, got %.1f", gs.ActionHistory[0].Amount)
	}
}

func TestParsePosition_AllActionTypes(t *testing.T) {
	fen := "BTN:AA:S100/BB:KK:S100|P20|Kh9s4c7d2s|xb5cr10f|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	// Check, bet 5, call, raise 10, fold
	expectedTypes := []ActionType{Check, Bet, Call, Raise, Fold}

	if len(gs.ActionHistory) != len(expectedTypes) {
		t.Fatalf("expected %d actions, got %d", len(expectedTypes), len(gs.ActionHistory))
	}

	for i, expectedType := range expectedTypes {
		if gs.ActionHistory[i].Type != expectedType {
			t.Errorf("action %d: expected %v, got %v", i, expectedType, gs.ActionHistory[i].Type)
		}
	}
}

func TestParsePosition_Preflop(t *testing.T) {
	fen := "BTN:AA:S100/BB:KK:S100|P1.5|-|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	if len(gs.Board) != 0 {
		t.Errorf("expected empty board for preflop, got %d cards", len(gs.Board))
	}

	if gs.Street != Preflop {
		t.Errorf("expected Preflop, got %v", gs.Street)
	}
}

func TestParsePosition_BoardWithSlashes(t *testing.T) {
	// Test that slashes in board are parsed correctly
	fen := "BTN:AA:S100/BB:KK:S100|P10|Ah7h3c/5s/2d|>BTN"
	gs, err := ParsePosition(fen)

	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	if len(gs.Board) != 5 {
		t.Errorf("expected 5 cards, got %d", len(gs.Board))
	}

	// Verify the actual cards
	expectedCards := []cards.Card{
		cards.NewCard(cards.Ace, cards.Hearts),
		cards.NewCard(cards.Seven, cards.Hearts),
		cards.NewCard(cards.Three, cards.Clubs),
		cards.NewCard(cards.Five, cards.Spades),
		cards.NewCard(cards.Two, cards.Diamonds),
	}

	for i, expected := range expectedCards {
		if gs.Board[i] != expected {
			t.Errorf("card %d: expected %v, got %v", i, expected, gs.Board[i])
		}
	}
}

func TestParsePosition_Errors(t *testing.T) {
	tests := []struct {
		name string
		fen  string
	}{
		{"empty string", ""},
		{"too few parts", "BTN:AA:S100|P3"},
		{"invalid player format", "BTN-AA-S100/BB:KK:S100|P3|Kh9s4c|>BTN"},
		{"invalid stack format", "BTN:AA:100/BB:KK:S100|P3|Kh9s4c|>BTN"},
		{"invalid pot format", "BTN:AA:S100/BB:KK:S100|3|Kh9s4c|>BTN"},
		{"invalid board length", "BTN:AA:S100/BB:KK:S100|P3|Kh9s|>BTN"},
		{"invalid card in board", "BTN:AA:S100/BB:KK:S100|P3|Xh9s4c|>BTN"},
		{"invalid action", "BTN:AA:S100/BB:KK:S100|P3|Kh9s4c|BTN"},
		{"position not found", "BTN:AA:S100/BB:KK:S100|P3|Kh9s4c|>CO"},
		{"invalid action in history", "BTN:AA:S100/BB:KK:S100|P3|Kh9s4c|z|>BTN"},
		{"bet without amount", "BTN:AA:S100/BB:KK:S100|P3|Kh9s4c|b|>BTN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePosition(tt.fen)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.fen)
			}
		})
	}
}

func TestParsePlayer(t *testing.T) {
	tests := []struct {
		name          string
		playerStr     string
		wantPosition  Position
		wantStack     float64
		wantNumCombos int
		wantErr       bool
	}{
		{
			name:          "specific cards",
			playerStr:     "BTN:AsKd:S100",
			wantPosition:  BTN,
			wantStack:     100,
			wantNumCombos: 1,
			wantErr:       false,
		},
		{
			name:          "pair range",
			playerStr:     "BB:AA:S50",
			wantPosition:  BB,
			wantStack:     50,
			wantNumCombos: 6,
			wantErr:       false,
		},
		{
			name:          "unknown range",
			playerStr:     "CO:??:S75.5",
			wantPosition:  CO,
			wantStack:     75.5,
			wantNumCombos: 0,
			wantErr:       false,
		},
		{
			name:      "invalid format",
			playerStr: "BTN-AA-S100",
			wantErr:   true,
		},
		{
			name:      "invalid stack",
			playerStr: "BTN:AA:100",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player, err := parsePlayer(tt.playerStr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if player.Position != tt.wantPosition {
				t.Errorf("position: got %v, want %v", player.Position, tt.wantPosition)
			}

			if player.Stack != tt.wantStack {
				t.Errorf("stack: got %.1f, want %.1f", player.Stack, tt.wantStack)
			}

			gotCombos := len(player.Range)
			if gotCombos != tt.wantNumCombos {
				t.Errorf("num combos: got %d, want %d", gotCombos, tt.wantNumCombos)
			}
		})
	}
}

func TestParseBoard(t *testing.T) {
	tests := []struct {
		name      string
		boardStr  string
		wantCards int
		wantErr   bool
	}{
		{"flop", "Th9h2c", 3, false},
		{"turn", "Th9h2c/Js", 4, false},
		{"river", "Th9h2c/Js/3d", 5, false},
		{"preflop dash", "-", 0, false},
		{"preflop empty", "", 0, false},
		{"invalid length", "Th9h2", 0, true},
		{"wrong card count", "Th9h", 0, true},
		{"invalid card", "Xh9h2c", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := parseBoard(tt.boardStr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(board) != tt.wantCards {
				t.Errorf("got %d cards, want %d", len(board), tt.wantCards)
			}
		})
	}
}

func TestParseHistory(t *testing.T) {
	tests := []struct {
		name        string
		historyStr  string
		wantActions []Action
		wantErr     bool
	}{
		{
			name:        "empty",
			historyStr:  "",
			wantActions: nil,
			wantErr:     false,
		},
		{
			name:       "check",
			historyStr: "x",
			wantActions: []Action{
				{Type: Check},
			},
			wantErr: false,
		},
		{
			name:       "bet call",
			historyStr: "b10c",
			wantActions: []Action{
				{Type: Bet, Amount: 10},
				{Type: Call},
			},
			wantErr: false,
		},
		{
			name:       "complex sequence",
			historyStr: "b5cr12.5c",
			wantActions: []Action{
				{Type: Bet, Amount: 5},
				{Type: Call},
				{Type: Raise, Amount: 12.5},
				{Type: Call},
			},
			wantErr: false,
		},
		{
			name:       "all actions",
			historyStr: "xb3cr6f",
			wantActions: []Action{
				{Type: Check},
				{Type: Bet, Amount: 3},
				{Type: Call},
				{Type: Raise, Amount: 6},
				{Type: Fold},
			},
			wantErr: false,
		},
		{
			name:        "invalid action",
			historyStr:  "z",
			wantActions: nil,
			wantErr:     true,
		},
		{
			name:        "bet without amount",
			historyStr:  "b",
			wantActions: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := parseHistory(tt.historyStr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(actions) != len(tt.wantActions) {
				t.Fatalf("got %d actions, want %d", len(actions), len(tt.wantActions))
			}

			for i, want := range tt.wantActions {
				got := actions[i]
				if got.Type != want.Type {
					t.Errorf("action %d type: got %v, want %v", i, got.Type, want.Type)
				}
				if got.Amount != want.Amount {
					t.Errorf("action %d amount: got %.1f, want %.1f", i, got.Amount, want.Amount)
				}
			}
		})
	}
}
