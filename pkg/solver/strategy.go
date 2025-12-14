package solver

import (
	"fmt"
	"math"

	"github.com/behrlich/poker-solver/pkg/notation"
)

// Strategy stores the strategy for a single information set
type Strategy struct {
	InfoSet string            // Information set key
	Actions []notation.Action // Available actions

	// CFR algorithm state
	RegretSum   []float64 // Cumulative regret for each action
	StrategySum []float64 // Cumulative strategy (for averaging)
}

// NewStrategy creates a new strategy for an information set
func NewStrategy(infoSet string, actions []notation.Action) *Strategy {
	n := len(actions)
	return &Strategy{
		InfoSet:     infoSet,
		Actions:     actions,
		RegretSum:   make([]float64, n),
		StrategySum: make([]float64, n),
	}
}

// GetStrategy computes the current strategy using regret matching
// Returns probability distribution over actions
func (s *Strategy) GetStrategy() []float64 {
	n := len(s.Actions)
	strategy := make([]float64, n)

	// Sum positive regrets
	normalizingSum := 0.0
	for i := 0; i < n; i++ {
		if s.RegretSum[i] > 0 {
			strategy[i] = s.RegretSum[i]
			normalizingSum += s.RegretSum[i]
		}
	}

	// Normalize to probability distribution
	if normalizingSum > 0 {
		for i := 0; i < n; i++ {
			strategy[i] /= normalizingSum
		}
	} else {
		// If no positive regrets, use uniform distribution
		uniform := 1.0 / float64(n)
		for i := 0; i < n; i++ {
			strategy[i] = uniform
		}
	}

	return strategy
}

// GetAverageStrategy returns the average strategy over all iterations
// This converges to the Nash equilibrium
func (s *Strategy) GetAverageStrategy() []float64 {
	n := len(s.Actions)
	avgStrategy := make([]float64, n)

	// Sum all strategy values
	normalizingSum := 0.0
	for i := 0; i < n; i++ {
		normalizingSum += s.StrategySum[i]
	}

	// Normalize
	if normalizingSum > 0 {
		for i := 0; i < n; i++ {
			avgStrategy[i] = s.StrategySum[i] / normalizingSum
		}
	} else {
		// If no data, return uniform
		uniform := 1.0 / float64(n)
		for i := 0; i < n; i++ {
			avgStrategy[i] = uniform
		}
	}

	return avgStrategy
}

// UpdateRegrets adds regrets for each action
func (s *Strategy) UpdateRegrets(regrets []float64) {
	for i := 0; i < len(s.Actions); i++ {
		s.RegretSum[i] += regrets[i]
	}
}

// UpdateStrategy adds current strategy to strategy sum (for averaging)
// reachProb is the probability of reaching this infoset
func (s *Strategy) UpdateStrategy(strategy []float64, reachProb float64) {
	for i := 0; i < len(s.Actions); i++ {
		s.StrategySum[i] += reachProb * strategy[i]
	}
}

// String returns a human-readable representation
func (s *Strategy) String() string {
	avgStrat := s.GetAverageStrategy()
	result := fmt.Sprintf("InfoSet: %s\n", s.InfoSet)
	for i, action := range s.Actions {
		result += fmt.Sprintf("  %s: %.1f%% (regret: %.2f)\n",
			action.String(), avgStrat[i]*100, s.RegretSum[i])
	}
	return result
}

// StrategyProfile stores strategies for all information sets
type StrategyProfile struct {
	strategies map[string]*Strategy
}

// NewStrategyProfile creates a new strategy profile
func NewStrategyProfile() *StrategyProfile {
	return &StrategyProfile{
		strategies: make(map[string]*Strategy),
	}
}

// GetOrCreate gets an existing strategy or creates a new one
func (sp *StrategyProfile) GetOrCreate(infoSet string, actions []notation.Action) *Strategy {
	if s, exists := sp.strategies[infoSet]; exists {
		return s
	}

	s := NewStrategy(infoSet, actions)
	sp.strategies[infoSet] = s
	return s
}

// Get retrieves a strategy by infoset key
func (sp *StrategyProfile) Get(infoSet string) (*Strategy, bool) {
	s, exists := sp.strategies[infoSet]
	return s, exists
}

// All returns all strategies
func (sp *StrategyProfile) All() map[string]*Strategy {
	return sp.strategies
}

// NumInfoSets returns the number of information sets
func (sp *StrategyProfile) NumInfoSets() int {
	return len(sp.strategies)
}

// GetAverageStrategies returns the average strategy for all infosets
func (sp *StrategyProfile) GetAverageStrategies() map[string][]float64 {
	result := make(map[string][]float64)
	for infoSet, strat := range sp.strategies {
		result[infoSet] = strat.GetAverageStrategy()
	}
	return result
}

// Exploitability calculates how exploitable the average strategy is
// This is a placeholder - full implementation requires best response calculation
func (sp *StrategyProfile) Exploitability() float64 {
	// TODO: Implement best response calculation
	// For now, return a metric based on regret magnitude
	totalAbsRegret := 0.0
	count := 0
	for _, strat := range sp.strategies {
		for _, regret := range strat.RegretSum {
			totalAbsRegret += math.Abs(regret)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return totalAbsRegret / float64(count)
}
