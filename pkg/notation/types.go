package notation

import (
	"fmt"

	"github.com/behrlich/poker-solver/pkg/cards"
)

// ActionType represents a poker action
type ActionType uint8

const (
	Check ActionType = iota
	Call
	Bet
	Raise
	Fold
)

// String returns the action type as a string
func (a ActionType) String() string {
	switch a {
	case Check:
		return "check"
	case Call:
		return "call"
	case Bet:
		return "bet"
	case Raise:
		return "raise"
	case Fold:
		return "fold"
	default:
		return "unknown"
	}
}

// Action represents a poker action with an optional amount (for bets/raises)
type Action struct {
	Type   ActionType
	Amount float64 // In big blinds (0 for check/call/fold)
}

// String returns the action in notation format (e.g., "c", "b3.5", "r9")
func (a Action) String() string {
	switch a.Type {
	case Check:
		return "x"
	case Call:
		return "c"
	case Bet:
		return fmt.Sprintf("b%.1f", a.Amount)
	case Raise:
		return fmt.Sprintf("r%.1f", a.Amount)
	case Fold:
		return "f"
	default:
		return "?"
	}
}

// Position represents a player's position at the table
type Position string

const (
	BTN Position = "BTN" // Button
	SB  Position = "SB"  // Small blind
	BB  Position = "BB"  // Big blind
	UTG Position = "UTG" // Under the gun
	MP  Position = "MP"  // Middle position
	CO  Position = "CO"  // Cutoff
)

// PlayerRange represents a player's range and stack
type PlayerRange struct {
	Position Position
	Range    []Combo // All possible hole card combinations
	Stack    float64 // Stack size in big blinds
}

// Street represents which betting round we're on
type Street uint8

const (
	Preflop Street = iota
	Flop
	Turn
	River
)

// String returns the street name
func (s Street) String() string {
	switch s {
	case Preflop:
		return "preflop"
	case Flop:
		return "flop"
	case Turn:
		return "turn"
	case River:
		return "river"
	default:
		return "unknown"
	}
}

// GameState represents a complete poker game state
type GameState struct {
	// Players and their ranges
	Players []PlayerRange

	// Current pot size (in big blinds)
	Pot float64

	// Community cards (board)
	Board []cards.Card

	// Action history for this street
	ActionHistory []Action

	// Which player acts next (index into Players)
	ToAct int

	// Current street
	Street Street
}

// GetStreet determines the street based on board cards
func GetStreet(boardSize int) Street {
	switch boardSize {
	case 0:
		return Preflop
	case 3:
		return Flop
	case 4:
		return Turn
	case 5:
		return River
	default:
		return Preflop
	}
}

// Clone creates a deep copy of the GameState
func (gs *GameState) Clone() *GameState {
	clone := &GameState{
		Players:       make([]PlayerRange, len(gs.Players)),
		Pot:           gs.Pot,
		Board:         make([]cards.Card, len(gs.Board)),
		ActionHistory: make([]Action, len(gs.ActionHistory)),
		ToAct:         gs.ToAct,
		Street:        gs.Street,
	}

	// Copy players (ranges are references, but that's ok for our use case)
	for i, player := range gs.Players {
		clone.Players[i] = PlayerRange{
			Position: player.Position,
			Range:    player.Range, // Shallow copy is ok - ranges are immutable
			Stack:    player.Stack,
		}
	}

	// Copy board
	copy(clone.Board, gs.Board)

	// Copy action history
	copy(clone.ActionHistory, gs.ActionHistory)

	return clone
}

// String returns a human-readable representation of the game state
func (gs *GameState) String() string {
	return fmt.Sprintf("GameState{Players=%d, Pot=%.1fbb, Board=%v, ToAct=%s, Street=%s}",
		len(gs.Players),
		gs.Pot,
		gs.Board,
		gs.Players[gs.ToAct].Position,
		gs.Street,
	)
}
