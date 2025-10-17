package tree

import (
	"github.com/behrlich/poker-solver/pkg/notation"
)

// ActionConfig specifies what actions are available at each decision point
type ActionConfig struct {
	// BetSizes are pot-relative bet sizes (e.g., 0.5 = 50% pot, 1.0 = pot-sized)
	// Empty slice means no betting allowed (e.g., facing a bet)
	BetSizes []float64

	// AllowCheck is true if checking is a legal action
	AllowCheck bool

	// AllowCall is true if calling is a legal action (facing a bet)
	AllowCall bool

	// AllowFold is true if folding is a legal action (facing a bet)
	AllowFold bool
}

// GenerateActions generates all legal actions for a given game state
// This is the action abstraction - we choose which bet sizes to include
func GenerateActions(pot float64, stack float64, lastAction *notation.Action, config ActionConfig) []notation.Action {
	var actions []notation.Action

	// If facing a bet/raise, can fold or call
	if lastAction != nil && (lastAction.Type == notation.Bet || lastAction.Type == notation.Raise) {
		if config.AllowFold {
			actions = append(actions, notation.Action{Type: notation.Fold})
		}
		if config.AllowCall {
			actions = append(actions, notation.Action{Type: notation.Call})
		}
		// Note: We don't implement raises in v0.1 river solver (keep tree small)
		// Will add in v0.2
		return actions
	}

	// If nobody has bet yet, can check or bet
	if config.AllowCheck {
		actions = append(actions, notation.Action{Type: notation.Check})
	}

	// Generate bet actions based on pot-relative sizes
	for _, sizeFraction := range config.BetSizes {
		betAmount := pot * sizeFraction

		// Cap bet at remaining stack (all-in)
		if betAmount >= stack {
			betAmount = stack
		}

		// Skip if this bet size is too small (< 0.01 bb)
		if betAmount < 0.01 {
			continue
		}

		actions = append(actions, notation.Action{
			Type:   notation.Bet,
			Amount: betAmount,
		})
	}

	// Always include all-in as an option if stack > 0 and we have bet sizes
	if stack > 0.01 && len(config.BetSizes) > 0 {
		// Check if all-in is already included (avoid duplicate)
		hasAllIn := false
		for _, action := range actions {
			if action.Type == notation.Bet && action.Amount >= stack-0.01 {
				hasAllIn = true
				break
			}
		}

		if !hasAllIn {
			actions = append(actions, notation.Action{
				Type:   notation.Bet,
				Amount: stack,
			})
		}
	}

	return actions
}

// DefaultRiverConfig returns a reasonable default action config for river play
// Allows check or bet with 2-3 standard sizes
func DefaultRiverConfig() ActionConfig {
	return ActionConfig{
		BetSizes:   []float64{0.5, 0.75, 1.5}, // 50%, 75%, 150% pot
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}
}

// GetLastAction returns the last action from action history, or nil if empty
func GetLastAction(history []notation.Action) *notation.Action {
	if len(history) == 0 {
		return nil
	}
	return &history[len(history)-1]
}
