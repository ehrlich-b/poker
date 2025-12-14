package tree

import (
	"fmt"
	"strings"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
)

// TreeNode represents a single node in the game tree
type TreeNode struct {
	// InfoSet is the information set key: what this player knows
	// Format: "board|history|>player|cards"
	// Example: "Kh9s4c7d2s|b10c|>BTN|AhKh"
	InfoSet string

	// Player index: 0 or 1 for decision nodes, -1 for chance/terminal nodes
	// Only meaningful for non-terminal, non-chance nodes
	Player int

	// Current pot size in BB
	Pot float64

	// Legal actions available from this node (decision nodes)
	Actions []notation.Action

	// Children nodes indexed by action key (decision nodes) or combo pair key (chance nodes)
	Children map[string]*TreeNode

	// Chance node support (for range-vs-range)
	IsChance            bool               // True if this is a chance node (deals hands from ranges)
	ChanceProbabilities map[string]float64 // Probability of each child (for chance nodes)

	// Terminal node flags
	IsTerminal bool       // True if this is a terminal node (showdown or fold)
	Payoff     [2]float64 // Payoffs for each player at terminal nodes

	// Rollout support (for turn→river, flop→turn→river)
	NeedsRollout bool              // True if this terminal needs future card rollout
	PlayerCombos [2]notation.Combo // Player combos (for rollout evaluation)

	// Game state information
	Board  []cards.Card // Community cards
	Stacks [2]float64   // Remaining stacks for each player
}

// ActionKey returns a string key for an action (for use in Children map)
func ActionKey(action notation.Action) string {
	return action.String()
}

// GetInfoSet generates the information set key for a game state and specific hole cards
// InfoSet format: "board|action_history|>acting_player|hole_cards"
// This represents what a single player knows at a decision point
func GetInfoSet(board []cards.Card, history []notation.Action, actingPlayer notation.Position, holeCards []cards.Card) string {
	var parts []string

	// Board cards
	boardStr := ""
	for _, card := range board {
		boardStr += card.String()
	}
	parts = append(parts, boardStr)

	// Action history (empty string if no actions)
	historyStr := ""
	for _, action := range history {
		historyStr += action.String()
	}
	parts = append(parts, historyStr)

	// Acting player indicator
	parts = append(parts, ">"+string(actingPlayer))

	// Hole cards (what this player knows)
	holeCardsStr := ""
	for _, card := range holeCards {
		holeCardsStr += card.String()
	}
	parts = append(parts, holeCardsStr)

	return strings.Join(parts, "|")
}

// GetInfoSetBucketed generates the information set key using a bucket ID instead of specific cards
// InfoSet format: "board|action_history|>acting_player|BUCKET_35"
// This is used for card abstraction - hands in the same bucket are treated identically
func GetInfoSetBucketed(board []cards.Card, history []notation.Action, actingPlayer notation.Position, bucketID int) string {
	var parts []string

	// Board cards
	boardStr := ""
	for _, card := range board {
		boardStr += card.String()
	}
	parts = append(parts, boardStr)

	// Action history (empty string if no actions)
	historyStr := ""
	for _, action := range history {
		historyStr += action.String()
	}
	parts = append(parts, historyStr)

	// Acting player indicator
	parts = append(parts, ">"+string(actingPlayer))

	// Bucket ID (replaces specific hole cards)
	bucketStr := fmt.Sprintf("BUCKET_%d", bucketID)
	parts = append(parts, bucketStr)

	return strings.Join(parts, "|")
}

// NewTerminalNode creates a terminal node (showdown or fold)
func NewTerminalNode(pot float64, payoffs [2]float64, board []cards.Card, stacks [2]float64) *TreeNode {
	return &TreeNode{
		InfoSet:             "", // Terminal nodes don't have info sets
		Player:              -1,
		Pot:                 pot,
		Actions:             nil,
		Children:            nil,
		IsChance:            false,
		ChanceProbabilities: nil,
		IsTerminal:          true,
		Payoff:              payoffs,
		NeedsRollout:        false,
		PlayerCombos:        [2]notation.Combo{},
		Board:               board,
		Stacks:              stacks,
	}
}

// NewRolloutNode creates a terminal node that requires future card rollout
// Used for turn showdowns (need to sample river) or flop showdowns (need turn+river)
func NewRolloutNode(pot float64, board []cards.Card, stacks [2]float64, combos [2]notation.Combo) *TreeNode {
	return &TreeNode{
		InfoSet:             "",
		Player:              -1,
		Pot:                 pot,
		Actions:             nil,
		Children:            nil,
		IsChance:            false,
		ChanceProbabilities: nil,
		IsTerminal:          true,
		Payoff:              [2]float64{0, 0}, // Will be computed during rollout
		NeedsRollout:        true,
		PlayerCombos:        combos,
		Board:               board,
		Stacks:              stacks,
	}
}

// NewDecisionNode creates a decision node where a player must act
func NewDecisionNode(infoSet string, player int, pot float64, actions []notation.Action, board []cards.Card, stacks [2]float64) *TreeNode {
	return &TreeNode{
		InfoSet:             infoSet,
		Player:              player,
		Pot:                 pot,
		Actions:             actions,
		Children:            make(map[string]*TreeNode),
		IsChance:            false,
		ChanceProbabilities: nil,
		IsTerminal:          false,
		Payoff:              [2]float64{0, 0},
		NeedsRollout:        false,
		PlayerCombos:        [2]notation.Combo{},
		Board:               board,
		Stacks:              stacks,
	}
}

// NewChanceNode creates a chance node that samples hands from ranges
func NewChanceNode(pot float64, board []cards.Card, stacks [2]float64) *TreeNode {
	return &TreeNode{
		InfoSet:             "",
		Player:              -1,
		Pot:                 pot,
		Actions:             nil,
		Children:            make(map[string]*TreeNode),
		IsChance:            true,
		ChanceProbabilities: make(map[string]float64),
		IsTerminal:          false,
		Payoff:              [2]float64{0, 0},
		NeedsRollout:        false,
		PlayerCombos:        [2]notation.Combo{},
		Board:               board,
		Stacks:              stacks,
	}
}

// String returns a human-readable representation of the node
func (n *TreeNode) String() string {
	if n.IsTerminal {
		return fmt.Sprintf("Terminal{pot=%.1fbb, payoffs=[%.1f, %.1f]}", n.Pot, n.Payoff[0], n.Payoff[1])
	}
	if n.IsChance {
		return fmt.Sprintf("Chance{pot=%.1fbb, outcomes=%d}", n.Pot, len(n.Children))
	}
	return fmt.Sprintf("Decision{player=%d, pot=%.1fbb, actions=%d, infoset=%s}", n.Player, n.Pot, len(n.Actions), n.InfoSet)
}

// IsShowdown returns true if this is a terminal showdown node
func (n *TreeNode) IsShowdown() bool {
	return n.IsTerminal && n.Payoff[0]+n.Payoff[1] == 0 // Zero-sum means showdown, not fold
}

// NumChildren returns the number of child nodes
func (n *TreeNode) NumChildren() int {
	return len(n.Children)
}
