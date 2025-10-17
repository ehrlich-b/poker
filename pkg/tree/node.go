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

	// Player index (0 or 1) whose turn it is to act
	// Only meaningful for non-terminal nodes
	Player int

	// Current pot size in BB
	Pot float64

	// Legal actions available from this node
	Actions []notation.Action

	// Children nodes indexed by action taken
	Children map[string]*TreeNode

	// Terminal node flags
	IsTerminal bool       // True if this is a terminal node (showdown or fold)
	Payoff     [2]float64 // Payoffs for each player at terminal nodes

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

// NewTerminalNode creates a terminal node (showdown or fold)
func NewTerminalNode(pot float64, payoffs [2]float64, board []cards.Card, stacks [2]float64) *TreeNode {
	return &TreeNode{
		InfoSet:    "", // Terminal nodes don't have info sets
		Player:     -1,
		Pot:        pot,
		Actions:    nil,
		Children:   nil,
		IsTerminal: true,
		Payoff:     payoffs,
		Board:      board,
		Stacks:     stacks,
	}
}

// NewDecisionNode creates a decision node where a player must act
func NewDecisionNode(infoSet string, player int, pot float64, actions []notation.Action, board []cards.Card, stacks [2]float64) *TreeNode {
	return &TreeNode{
		InfoSet:    infoSet,
		Player:     player,
		Pot:        pot,
		Actions:    actions,
		Children:   make(map[string]*TreeNode),
		IsTerminal: false,
		Payoff:     [2]float64{0, 0},
		Board:      board,
		Stacks:     stacks,
	}
}

// String returns a human-readable representation of the node
func (n *TreeNode) String() string {
	if n.IsTerminal {
		return fmt.Sprintf("Terminal{pot=%.1fbb, payoffs=[%.1f, %.1f]}", n.Pot, n.Payoff[0], n.Payoff[1])
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
