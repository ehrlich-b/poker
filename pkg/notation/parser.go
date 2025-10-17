package notation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/behrlich/poker-solver/pkg/cards"
)

// ParsePosition parses a position FEN string into a GameState
// Format: <players>|<pot>|<board>|<history>|<action>
// Example: "BTN:AsKd:S98/BB:??:S97|P3|Th9h2c|>BTN"
// Example with range: "BTN:AA,KK/BB:QQ-JJ|P20|Kh9s4c7d2s|>BTN"
func ParsePosition(fen string) (*GameState, error) {
	fen = strings.TrimSpace(fen)
	if fen == "" {
		return nil, fmt.Errorf("empty position FEN")
	}

	// Split by | to get main components
	parts := strings.Split(fen, "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid FEN format: expected at least 4 parts separated by |, got %d", len(parts))
	}

	playersStr := parts[0]
	potStr := parts[1]
	boardStr := parts[2]

	// History is optional (can be empty)
	var historyStr string
	var actionStr string

	if len(parts) == 4 {
		// No history, just action indicator
		actionStr = parts[3]
	} else if len(parts) == 5 {
		// History present
		historyStr = parts[3]
		actionStr = parts[4]
	} else {
		return nil, fmt.Errorf("invalid FEN format: too many parts (%d)", len(parts))
	}

	// Parse each component
	players, err := parsePlayers(playersStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing players: %w", err)
	}

	pot, err := parsePot(potStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing pot: %w", err)
	}

	board, err := parseBoard(boardStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing board: %w", err)
	}

	history, err := parseHistory(historyStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing history: %w", err)
	}

	toAct, err := parseAction(actionStr, players)
	if err != nil {
		return nil, fmt.Errorf("error parsing action: %w", err)
	}

	street := GetStreet(len(board))

	return &GameState{
		Players:       players,
		Pot:           pot,
		Board:         board,
		ActionHistory: history,
		ToAct:         toAct,
		Street:        street,
	}, nil
}

// parsePlayers parses the players section: "POS:CARDS:STACK/POS:CARDS:STACK/..."
func parsePlayers(playersStr string) ([]PlayerRange, error) {
	playersStr = strings.TrimSpace(playersStr)
	if playersStr == "" {
		return nil, fmt.Errorf("empty players string")
	}

	playerParts := strings.Split(playersStr, "/")
	players := make([]PlayerRange, 0, len(playerParts))

	for _, playerStr := range playerParts {
		player, err := parsePlayer(playerStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing player %q: %w", playerStr, err)
		}
		players = append(players, player)
	}

	return players, nil
}

// parsePlayer parses a single player: "POS:CARDS:STACK"
// CARDS can be:
//   - Specific cards: "AsKd"
//   - Range: "AA,KK,AKs"
//   - Unknown: "??"
func parsePlayer(playerStr string) (PlayerRange, error) {
	playerStr = strings.TrimSpace(playerStr)
	parts := strings.Split(playerStr, ":")

	if len(parts) != 3 {
		return PlayerRange{}, fmt.Errorf("invalid player format %q (expected POS:CARDS:STACK)", playerStr)
	}

	position := Position(strings.TrimSpace(parts[0]))
	cardsStr := strings.TrimSpace(parts[1])
	stackStr := strings.TrimSpace(parts[2])

	// Parse stack (format: S100 for 100bb)
	if len(stackStr) < 2 || stackStr[0] != 'S' {
		return PlayerRange{}, fmt.Errorf("invalid stack format %q (expected S{amount})", stackStr)
	}

	stack, err := strconv.ParseFloat(stackStr[1:], 64)
	if err != nil {
		return PlayerRange{}, fmt.Errorf("invalid stack amount %q: %w", stackStr, err)
	}

	// Parse cards/range
	var combos []Combo

	if cardsStr == "??" {
		// Unknown range - leave empty for now
		// In actual solving, this would be filled by context
		combos = nil
	} else if len(cardsStr) == 4 && isSpecificCards(cardsStr) {
		// Specific hole cards (e.g., "AsKd")
		card1, err := cards.ParseCard(cardsStr[0:2])
		if err != nil {
			return PlayerRange{}, fmt.Errorf("error parsing card1 from %q: %w", cardsStr, err)
		}
		card2, err := cards.ParseCard(cardsStr[2:4])
		if err != nil {
			return PlayerRange{}, fmt.Errorf("error parsing card2 from %q: %w", cardsStr, err)
		}
		combos = []Combo{{Card1: card1, Card2: card2}}
	} else {
		// Range notation (e.g., "AA,KK,AKs")
		var err error
		combos, err = ParseRange(cardsStr)
		if err != nil {
			return PlayerRange{}, fmt.Errorf("error parsing range %q: %w", cardsStr, err)
		}
	}

	return PlayerRange{
		Position: position,
		Range:    combos,
		Stack:    stack,
	}, nil
}

// isSpecificCards checks if a string represents specific hole cards (e.g., "AsKd")
func isSpecificCards(s string) bool {
	if len(s) != 4 {
		return false
	}
	// Check if it looks like two cards: rank+suit+rank+suit
	// Valid ranks: A,K,Q,J,T,9-2
	// Valid suits: s,h,d,c
	ranks := "AKQJT98765432"
	suits := "shdc"

	return strings.ContainsRune(ranks, rune(s[0])) &&
		strings.ContainsRune(suits, rune(s[1])) &&
		strings.ContainsRune(ranks, rune(s[2])) &&
		strings.ContainsRune(suits, rune(s[3]))
}

// parsePot parses pot string: "P3" → 3.0
func parsePot(potStr string) (float64, error) {
	potStr = strings.TrimSpace(potStr)
	if len(potStr) < 2 || potStr[0] != 'P' {
		return 0, fmt.Errorf("invalid pot format %q (expected P{amount})", potStr)
	}

	pot, err := strconv.ParseFloat(potStr[1:], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid pot amount %q: %w", potStr, err)
	}

	return pot, nil
}

// parseBoard parses board string: "Th9h2c" (flop), "Th9h2c/Js" (turn), "Th9h2c/Js/3d" (river)
// Empty string or "-" for preflop
func parseBoard(boardStr string) ([]cards.Card, error) {
	boardStr = strings.TrimSpace(boardStr)

	if boardStr == "" || boardStr == "-" {
		return nil, nil // Preflop
	}

	// Remove slashes to get all cards in one string
	boardStr = strings.ReplaceAll(boardStr, "/", "")

	// Must be multiple of 2 (each card is 2 characters)
	if len(boardStr)%2 != 0 {
		return nil, fmt.Errorf("invalid board length %q (must be even)", boardStr)
	}

	numCards := len(boardStr) / 2
	if numCards != 3 && numCards != 4 && numCards != 5 {
		return nil, fmt.Errorf("invalid board %q (must have 3, 4, or 5 cards)", boardStr)
	}

	board := make([]cards.Card, numCards)
	for i := 0; i < numCards; i++ {
		card, err := cards.ParseCard(boardStr[i*2 : i*2+2])
		if err != nil {
			return nil, fmt.Errorf("error parsing board card %q: %w", boardStr[i*2:i*2+2], err)
		}
		board[i] = card
	}

	return board, nil
}

// parseHistory parses action history: "b3.5c" → [bet 3.5, call]
// Empty string returns empty slice
func parseHistory(historyStr string) ([]Action, error) {
	historyStr = strings.TrimSpace(historyStr)

	if historyStr == "" {
		return nil, nil
	}

	var actions []Action
	i := 0

	for i < len(historyStr) {
		char := historyStr[i]

		switch char {
		case 'x', 'X':
			// Check
			actions = append(actions, Action{Type: Check})
			i++

		case 'c', 'C':
			// Call
			actions = append(actions, Action{Type: Call})
			i++

		case 'f', 'F':
			// Fold
			actions = append(actions, Action{Type: Fold})
			i++

		case 'b', 'B':
			// Bet with amount
			amount, consumed, err := parseActionAmount(historyStr[i+1:])
			if err != nil {
				return nil, fmt.Errorf("error parsing bet amount at position %d: %w", i, err)
			}
			actions = append(actions, Action{Type: Bet, Amount: amount})
			i += 1 + consumed

		case 'r', 'R':
			// Raise with amount
			amount, consumed, err := parseActionAmount(historyStr[i+1:])
			if err != nil {
				return nil, fmt.Errorf("error parsing raise amount at position %d: %w", i, err)
			}
			actions = append(actions, Action{Type: Raise, Amount: amount})
			i += 1 + consumed

		default:
			return nil, fmt.Errorf("invalid action character %q at position %d", char, i)
		}
	}

	return actions, nil
}

// parseActionAmount parses the numeric amount following a bet/raise action
// Returns (amount, charactersConsumed, error)
func parseActionAmount(s string) (float64, int, error) {
	if len(s) == 0 {
		return 0, 0, fmt.Errorf("missing amount after bet/raise")
	}

	// Find where the number ends (at next action letter or end of string)
	end := 0
	for end < len(s) {
		c := s[end]
		if (c >= '0' && c <= '9') || c == '.' {
			end++
		} else {
			break
		}
	}

	if end == 0 {
		return 0, 0, fmt.Errorf("missing numeric amount")
	}

	amountStr := s[0:end]
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid amount %q: %w", amountStr, err)
	}

	return amount, end, nil
}

// parseAction parses action indicator: ">BTN" → player index
func parseAction(actionStr string, players []PlayerRange) (int, error) {
	actionStr = strings.TrimSpace(actionStr)

	if len(actionStr) < 2 || actionStr[0] != '>' {
		return 0, fmt.Errorf("invalid action format %q (expected >{POSITION})", actionStr)
	}

	position := Position(actionStr[1:])

	// Find player index with this position
	for i, player := range players {
		if player.Position == position {
			return i, nil
		}
	}

	return 0, fmt.Errorf("position %q not found in players", position)
}
