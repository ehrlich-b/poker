package solver

import (
	"github.com/behrlich/poker-solver/pkg/tree"
)

// CFR implements vanilla Counterfactual Regret Minimization
type CFR struct {
	profile *StrategyProfile
}

// NewCFR creates a new CFR solver
func NewCFR() *CFR {
	return &CFR{
		profile: NewStrategyProfile(),
	}
}

// Train runs CFR for the specified number of iterations
// Returns the strategy profile after training
func (c *CFR) Train(root *tree.TreeNode, iterations int) *StrategyProfile {
	for i := 0; i < iterations; i++ {
		c.Iterate(root)
	}
	return c.profile
}

// Iterate runs a single CFR iteration
// This is useful for progress tracking in WASM/UI contexts
func (c *CFR) Iterate(root *tree.TreeNode) {
	c.cfr(root, 1.0, 1.0)
}

// cfr recursively traverses the game tree and updates regrets
// reachProb0 is the probability that player 0 reaches this node
// reachProb1 is the probability that player 1 reaches this node
// Returns the expected value for each player
func (c *CFR) cfr(node *tree.TreeNode, reachProb0, reachProb1 float64) [2]float64 {
	// Terminal node: return payoffs
	if node.IsTerminal {
		return node.Payoff
	}

	// Chance node: compute expected value over all outcomes
	if node.IsChance {
		nodeValue := [2]float64{0, 0}
		for childKey, child := range node.Children {
			prob := node.ChanceProbabilities[childKey]
			childValue := c.cfr(child, reachProb0*prob, reachProb1*prob)
			nodeValue[0] += prob * childValue[0]
			nodeValue[1] += prob * childValue[1]
		}
		return nodeValue
	}

	// Decision node: compute counterfactual values
	player := node.Player
	infoSet := node.InfoSet

	// Get or create strategy for this infoset
	strategy := c.profile.GetOrCreate(infoSet, node.Actions)

	// Get current strategy using regret matching
	currentStrategy := strategy.GetStrategy()

	// Track counterfactual values for each action
	numActions := len(node.Actions)
	actionValues := make([][2]float64, numActions)
	nodeValue := [2]float64{0, 0}

	// Recursively compute values for each action
	for i, action := range node.Actions {
		actionKey := tree.ActionKey(action)
		child, exists := node.Children[actionKey]
		if !exists {
			// Should not happen if tree is built correctly
			continue
		}

		// Update reach probabilities based on who's acting
		var childValue [2]float64
		if player == 0 {
			childValue = c.cfr(child, reachProb0*currentStrategy[i], reachProb1)
		} else {
			childValue = c.cfr(child, reachProb0, reachProb1*currentStrategy[i])
		}

		actionValues[i] = childValue
		// Accumulate expected value weighted by strategy
		nodeValue[0] += currentStrategy[i] * childValue[0]
		nodeValue[1] += currentStrategy[i] * childValue[1]
	}

	// Compute regrets and update strategy
	regrets := make([]float64, numActions)
	cfValue := nodeValue[player] // Counterfactual value at this node

	for i := 0; i < numActions; i++ {
		// Regret = value of action - value of current strategy
		actionCFValue := actionValues[i][player]
		regrets[i] = actionCFValue - cfValue
	}

	// Update regrets weighted by opponent's reach probability
	// (opponent's reach prob = probability this is a counterfactual scenario)
	var cfReachProb float64
	if player == 0 {
		cfReachProb = reachProb1
	} else {
		cfReachProb = reachProb0
	}

	scaledRegrets := make([]float64, numActions)
	for i := 0; i < numActions; i++ {
		scaledRegrets[i] = regrets[i] * cfReachProb
	}
	strategy.UpdateRegrets(scaledRegrets)

	// Update strategy sum weighted by own reach probability
	var ownReachProb float64
	if player == 0 {
		ownReachProb = reachProb0
	} else {
		ownReachProb = reachProb1
	}
	strategy.UpdateStrategy(currentStrategy, ownReachProb)

	return nodeValue
}

// GetProfile returns the current strategy profile
func (c *CFR) GetProfile() *StrategyProfile {
	return c.profile
}
