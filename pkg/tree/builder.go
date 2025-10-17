package tree

import (
	"fmt"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
)

// Builder constructs game trees from GameState
type Builder struct {
	Config ActionConfig
}

// NewBuilder creates a new tree builder with the given action config
func NewBuilder(config ActionConfig) *Builder {
	return &Builder{Config: config}
}

// Build constructs a game tree for a specific combo vs combo matchup
// This builds the full tree for these two specific hands
func (b *Builder) Build(gs *notation.GameState, combo0 notation.Combo, combo1 notation.Combo) (*TreeNode, error) {
	// Validate inputs
	if len(gs.Players) != 2 {
		return nil, fmt.Errorf("only 2-player games supported")
	}

	if len(gs.Board) != 5 && len(gs.Board) != 4 && len(gs.Board) != 3 {
		return nil, fmt.Errorf("only postflop (3-5 board cards) supported")
	}

	// Check for card conflicts
	if err := b.validateCards(gs.Board, combo0, combo1); err != nil {
		return nil, err
	}

	// Build tree recursively
	stacks := [2]float64{gs.Players[0].Stack, gs.Players[1].Stack}
	combos := [2]notation.Combo{combo0, combo1}

	return b.buildNode(gs.Board, gs.ActionHistory, gs.Pot, stacks, gs.ToAct, combos), nil
}

// buildNode recursively builds a node in the game tree
func (b *Builder) buildNode(
	board []cards.Card,
	history []notation.Action,
	pot float64,
	stacks [2]float64,
	toAct int,
	combos [2]notation.Combo,
) *TreeNode {
	// Check if we've reached a terminal node
	lastAction := GetLastAction(history)

	// Terminal: fold
	if lastAction != nil && lastAction.Type == notation.Fold {
		// Player who didn't fold wins the pot
		payoffs := [2]float64{0, 0}
		if toAct == 0 {
			// Player 1 folded, player 0 wins
			payoffs[0] = pot
		} else {
			// Player 0 folded, player 1 wins
			payoffs[1] = pot
		}
		return NewTerminalNode(pot, payoffs, board, stacks)
	}

	// Terminal: showdown (both players checked or someone called)
	if b.isShowdown(history) {
		payoffs := b.calculateShowdownPayoffs(board, combos, pot)
		return NewTerminalNode(pot, payoffs, board, stacks)
	}

	// Decision node: current player must act
	playerCombo := combos[toAct]
	holeCards := []cards.Card{playerCombo.Card1, playerCombo.Card2}

	// Generate info set key for this player
	playerPos := []notation.Position{notation.BTN, notation.BB}[toAct]
	infoSet := GetInfoSet(board, history, playerPos, holeCards)

	// Generate legal actions
	actions := GenerateActions(pot, stacks[toAct], lastAction, b.Config)

	// Create decision node
	node := NewDecisionNode(infoSet, toAct, pot, actions, board, stacks)

	// Build children for each action
	for _, action := range actions {
		newHistory := append([]notation.Action{}, history...)
		newHistory = append(newHistory, action)
		newPot := pot
		newStacks := stacks

		// Update pot and stacks based on action
		switch action.Type {
		case notation.Bet, notation.Raise:
			newPot += action.Amount
			newStacks[toAct] -= action.Amount

		case notation.Call:
			// Figure out how much to call
			callAmount := b.getCallAmount(history, pot, stacks[toAct])
			newPot += callAmount
			newStacks[toAct] -= callAmount

		case notation.Check, notation.Fold:
			// No pot/stack changes
		}

		// Determine next player
		var nextToAct int
		if action.Type == notation.Check {
			// After a check, opponent acts (unless both checked)
			nextToAct = 1 - toAct
		} else if action.Type == notation.Bet || action.Type == notation.Raise {
			// After a bet/raise, opponent acts
			nextToAct = 1 - toAct
		} else if action.Type == notation.Call || action.Type == notation.Fold {
			// After call/fold, game is over (will be caught by terminal checks)
			nextToAct = toAct // doesn't matter, will be terminal
		}

		// Recursively build child node
		child := b.buildNode(board, newHistory, newPot, newStacks, nextToAct, combos)
		node.Children[ActionKey(action)] = child
	}

	return node
}

// isShowdown returns true if we've reached a showdown
func (b *Builder) isShowdown(history []notation.Action) bool {
	if len(history) < 2 {
		return false
	}

	last := history[len(history)-1]
	secondLast := history[len(history)-2]

	// Both players checked
	if last.Type == notation.Check && secondLast.Type == notation.Check {
		return true
	}

	// Someone bet and opponent called
	if last.Type == notation.Call && (secondLast.Type == notation.Bet || secondLast.Type == notation.Raise) {
		return true
	}

	return false
}

// calculateShowdownPayoffs determines payoffs at showdown
func (b *Builder) calculateShowdownPayoffs(board []cards.Card, combos [2]notation.Combo, pot float64) [2]float64 {
	// Evaluate both hands
	hand0 := append([]cards.Card{combos[0].Card1, combos[0].Card2}, board...)
	hand1 := append([]cards.Card{combos[1].Card1, combos[1].Card2}, board...)

	rank0 := cards.Evaluate(hand0)
	rank1 := cards.Evaluate(hand1)

	cmp := rank0.Compare(rank1)

	if cmp > 0 {
		// Player 0 wins
		return [2]float64{pot, 0}
	} else if cmp < 0 {
		// Player 1 wins
		return [2]float64{0, pot}
	} else {
		// Tie (split pot)
		return [2]float64{pot / 2, pot / 2}
	}
}

// getCallAmount calculates how much the current player needs to call
func (b *Builder) getCallAmount(history []notation.Action, pot float64, stack float64) float64 {
	if len(history) == 0 {
		return 0
	}

	// Find the last bet/raise amount
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Type == notation.Bet || history[i].Type == notation.Raise {
			callAmount := history[i].Amount
			// Cap at remaining stack
			if callAmount > stack {
				return stack
			}
			return callAmount
		}
	}

	return 0
}

// validateCards checks for duplicate cards
func (b *Builder) validateCards(board []cards.Card, combo0, combo1 notation.Combo) error {
	allCards := make(map[cards.Card]bool)

	// Check board cards
	for _, card := range board {
		if allCards[card] {
			return fmt.Errorf("duplicate card in board: %v", card)
		}
		allCards[card] = true
	}

	// Check combo 0
	for _, card := range []cards.Card{combo0.Card1, combo0.Card2} {
		if allCards[card] {
			return fmt.Errorf("duplicate card: %v", card)
		}
		allCards[card] = true
	}

	// Check combo 1
	for _, card := range []cards.Card{combo1.Card1, combo1.Card2} {
		if allCards[card] {
			return fmt.Errorf("duplicate card: %v", card)
		}
		allCards[card] = true
	}

	return nil
}
