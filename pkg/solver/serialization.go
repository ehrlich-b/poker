package solver

import (
	"encoding/json"
	"os"

	"github.com/behrlich/poker-solver/pkg/notation"
)

// SerializableStrategy is a JSON-friendly representation of a Strategy
type SerializableStrategy struct {
	InfoSet     string                  `json:"infoset"`
	Actions     []SerializableAction    `json:"actions"`
	RegretSum   []float64               `json:"regret_sum"`
	StrategySum []float64               `json:"strategy_sum"`
}

// SerializableAction is a JSON-friendly representation of an Action
type SerializableAction struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount,omitempty"`
}

// SerializableProfile is a JSON-friendly representation of a StrategyProfile
type SerializableProfile struct {
	Strategies []SerializableStrategy `json:"strategies"`
	Version    string                 `json:"version"` // For future compatibility
}

// actionTypeToString converts ActionType to string for JSON
func actionTypeToString(t notation.ActionType) string {
	switch t {
	case notation.Check:
		return "check"
	case notation.Call:
		return "call"
	case notation.Bet:
		return "bet"
	case notation.Raise:
		return "raise"
	case notation.Fold:
		return "fold"
	default:
		return "unknown"
	}
}

// stringToActionType converts string to ActionType from JSON
func stringToActionType(s string) notation.ActionType {
	switch s {
	case "check":
		return notation.Check
	case "call":
		return notation.Call
	case "bet":
		return notation.Bet
	case "raise":
		return notation.Raise
	case "fold":
		return notation.Fold
	default:
		return notation.Check // default fallback
	}
}

// ToJSON serializes the StrategyProfile to JSON bytes
func (sp *StrategyProfile) ToJSON() ([]byte, error) {
	profile := SerializableProfile{
		Version:    "1.0",
		Strategies: make([]SerializableStrategy, 0, len(sp.strategies)),
	}

	for infoSet, strat := range sp.strategies {
		// Convert actions
		actions := make([]SerializableAction, len(strat.Actions))
		for i, action := range strat.Actions {
			actions[i] = SerializableAction{
				Type:   actionTypeToString(action.Type),
				Amount: action.Amount,
			}
		}

		// Add strategy
		profile.Strategies = append(profile.Strategies, SerializableStrategy{
			InfoSet:     infoSet,
			Actions:     actions,
			RegretSum:   strat.RegretSum,
			StrategySum: strat.StrategySum,
		})
	}

	return json.MarshalIndent(profile, "", "  ")
}

// FromJSON deserializes JSON bytes into a StrategyProfile
func FromJSON(data []byte) (*StrategyProfile, error) {
	var profile SerializableProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	sp := NewStrategyProfile()

	for _, serStrat := range profile.Strategies {
		// Convert actions back
		actions := make([]notation.Action, len(serStrat.Actions))
		for i, serAction := range serStrat.Actions {
			actions[i] = notation.Action{
				Type:   stringToActionType(serAction.Type),
				Amount: serAction.Amount,
			}
		}

		// Create strategy
		strat := NewStrategy(serStrat.InfoSet, actions)
		strat.RegretSum = serStrat.RegretSum
		strat.StrategySum = serStrat.StrategySum

		sp.strategies[serStrat.InfoSet] = strat
	}

	return sp, nil
}

// SaveToFile saves the StrategyProfile to a JSON file
func (sp *StrategyProfile) SaveToFile(filename string) error {
	data, err := sp.ToJSON()
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadFromFile loads a StrategyProfile from a JSON file
func LoadFromFile(filename string) (*StrategyProfile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return FromJSON(data)
}
