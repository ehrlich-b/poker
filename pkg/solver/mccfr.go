package solver

import (
	"math/rand"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/tree"
)

// MCCFR implements Monte Carlo Counterfactual Regret Minimization with outcome sampling
// This is more efficient than vanilla CFR for large trees (e.g., turn→river)
type MCCFR struct {
	profile *StrategyProfile
	rng     *rand.Rand
}

// NewMCCFR creates a new MCCFR solver with the given random seed
func NewMCCFR(seed int64) *MCCFR {
	return &MCCFR{
		profile: NewStrategyProfile(),
		rng:     rand.New(rand.NewSource(seed)),
	}
}

// Train runs MCCFR for the specified number of iterations
// Returns the strategy profile after training
// SAFETY: Maximum 100,000 iterations to prevent memory explosion
func (m *MCCFR) Train(root *tree.TreeNode, iterations int) *StrategyProfile {
	// SAFETY: Hard limit on iterations to prevent crashes
	const maxIterations = 100000
	if iterations > maxIterations {
		iterations = maxIterations
	}
	if iterations < 0 {
		iterations = 0
	}

	for i := 0; i < iterations; i++ {
		m.Iterate(root)
	}
	return m.profile
}

// Iterate runs a single MCCFR iteration
// This is useful for progress tracking in WASM/UI contexts
func (m *MCCFR) Iterate(root *tree.TreeNode) {
	m.mccfr(root, 1.0, 1.0, 1.0)
}

// mccfr recursively traverses the game tree using outcome sampling
// reachProb0 is the probability that player 0 reaches this node
// reachProb1 is the probability that player 1 reaches this node
// sampleProb is the probability of sampling this path (for importance sampling correction)
// Returns the sampled expected value for each player
func (m *MCCFR) mccfr(node *tree.TreeNode, reachProb0, reachProb1, sampleProb float64) [2]float64 {
	// Terminal node: return payoffs
	if node.IsTerminal {
		// Check if this terminal needs rollout (turn showdown)
		if node.NeedsRollout {
			return m.rollout(node)
		}
		return node.Payoff
	}

	// Chance node: sample one outcome
	if node.IsChance {
		return m.sampleChanceNode(node, reachProb0, reachProb1, sampleProb)
	}

	// Decision node: sample one action according to current strategy
	player := node.Player
	infoSet := node.InfoSet

	// Get or create strategy for this infoset
	strategy := m.profile.GetOrCreate(infoSet, node.Actions)

	// Get current strategy using regret matching
	currentStrategy := strategy.GetStrategy()

	// Sample action according to current strategy
	actionIdx := m.sampleAction(currentStrategy)
	action := node.Actions[actionIdx]
	actionKey := tree.ActionKey(action)

	child, exists := node.Children[actionKey]
	if !exists {
		// Should not happen if tree is built correctly
		return [2]float64{0, 0}
	}

	// Update reach probabilities and sample probability
	actionProb := currentStrategy[actionIdx]
	var childReachProb0, childReachProb1 float64
	if player == 0 {
		childReachProb0 = reachProb0 * actionProb
		childReachProb1 = reachProb1
	} else {
		childReachProb0 = reachProb0
		childReachProb1 = reachProb1 * actionProb
	}

	// In outcome sampling, we sample one action, so sample prob is multiplied by action prob
	childSampleProb := sampleProb * actionProb

	// Recursively compute value for sampled action
	childValue := m.mccfr(child, childReachProb0, childReachProb1, childSampleProb)

	// Compute counterfactual values for all actions
	// This is where MCCFR differs from vanilla CFR - we use importance sampling
	numActions := len(node.Actions)
	actionValues := make([][2]float64, numActions)

	// For the sampled action, we have the actual value
	actionValues[actionIdx] = childValue

	// For other actions, we could sample them too (not in basic outcome sampling)
	// In basic outcome sampling, we only update based on the sampled action

	// Node value is the sampled child value (weighted by strategy in expectation)
	nodeValue := childValue

	// Compute regrets for the sampled action
	// Regret for action i = Q(i) - V(node)
	// But we only sampled one action, so we need importance sampling correction
	regrets := make([]float64, numActions)
	cfValue := nodeValue[player]

	// For outcome sampling, regret for sampled action is:
	// (Q(action) - V) / sampleProb
	// For other actions, regret is 0 (or we could explore them too)
	actionCFValue := actionValues[actionIdx][player]
	regrets[actionIdx] = (actionCFValue - cfValue) / sampleProb

	// Update regrets weighted by opponent's reach probability
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

// sampleChanceNode samples one outcome from a chance node
func (m *MCCFR) sampleChanceNode(node *tree.TreeNode, reachProb0, reachProb1, sampleProb float64) [2]float64 {
	// Sample one outcome uniformly (for now - could use probabilities later)
	outcomes := make([]string, 0, len(node.Children))
	for key := range node.Children {
		outcomes = append(outcomes, key)
	}

	if len(outcomes) == 0 {
		return [2]float64{0, 0}
	}

	// Sample uniformly
	sampledKey := outcomes[m.rng.Intn(len(outcomes))]
	child := node.Children[sampledKey]
	prob := node.ChanceProbabilities[sampledKey]

	// Update sample probability (we sampled uniformly, but true prob is from ChanceProbabilities)
	// For importance sampling: childSampleProb = sampleProb * samplingProb
	// Here we sample uniformly, so samplingProb = 1/numOutcomes
	uniformProb := 1.0 / float64(len(outcomes))
	childSampleProb := sampleProb * uniformProb

	// Reach probabilities are multiplied by true chance probability
	childValue := m.mccfr(child, reachProb0*prob, reachProb1*prob, childSampleProb)

	// Importance sampling correction: multiply by (trueProb / samplingProb)
	correction := prob / uniformProb
	return [2]float64{childValue[0] * correction, childValue[1] * correction}
}

// rollout samples future cards and evaluates the hand
// Handles both turn→river (4 cards) and flop→turn→river (3 cards)
func (m *MCCFR) rollout(node *tree.TreeNode) [2]float64 {
	board := node.Board

	// Rollout only makes sense for flop (3 cards) or turn (4 cards)
	if len(board) != 3 && len(board) != 4 {
		// This shouldn't happen - rollout nodes should only be created on flop/turn
		return node.Payoff
	}

	// Get player combos
	combo0 := node.PlayerCombos[0]
	combo1 := node.PlayerCombos[1]

	// Build set of used cards
	usedCards := make(map[cards.Card]bool)
	for _, card := range board {
		usedCards[card] = true
	}
	usedCards[combo0.Card1] = true
	usedCards[combo0.Card2] = true
	usedCards[combo1.Card1] = true
	usedCards[combo1.Card2] = true

	// Generate all possible remaining cards
	// SAFETY: Use explicit rank list to avoid uint8 underflow
	possibleCards := make([]cards.Card, 0, 48)
	ranks := []cards.Rank{cards.Ace, cards.King, cards.Queen, cards.Jack, cards.Ten,
		cards.Nine, cards.Eight, cards.Seven, cards.Six, cards.Five,
		cards.Four, cards.Three, cards.Two}
	for _, rank := range ranks {
		for suit := cards.Spades; suit <= cards.Clubs; suit++ {
			card := cards.Card{Rank: rank, Suit: suit}
			if !usedCards[card] {
				possibleCards = append(possibleCards, card)
			}
		}
	}

	if len(possibleCards) < 2 {
		// Shouldn't happen
		return [2]float64{node.Pot / 2, node.Pot / 2}
	}

	// Build final board based on street
	var finalBoard []cards.Card
	if len(board) == 3 {
		// Flop: sample turn card, then river card
		turnCard := possibleCards[m.rng.Intn(len(possibleCards))]

		// Remove turn card from available cards
		possibleRivers := make([]cards.Card, 0, len(possibleCards)-1)
		for _, card := range possibleCards {
			if card != turnCard {
				possibleRivers = append(possibleRivers, card)
			}
		}

		riverCard := possibleRivers[m.rng.Intn(len(possibleRivers))]

		// Build final board: flop + turn + river
		finalBoard = append([]cards.Card{}, board...)
		finalBoard = append(finalBoard, turnCard, riverCard)
	} else {
		// Turn: sample river card only
		riverCard := possibleCards[m.rng.Intn(len(possibleCards))]

		// Build final board: turn + river
		finalBoard = append([]cards.Card{}, board...)
		finalBoard = append(finalBoard, riverCard)
	}

	// Evaluate hands with the final board (5 cards)
	hand0 := append([]cards.Card{combo0.Card1, combo0.Card2}, finalBoard...)
	hand1 := append([]cards.Card{combo1.Card1, combo1.Card2}, finalBoard...)

	rank0 := cards.Evaluate(hand0)
	rank1 := cards.Evaluate(hand1)

	cmp := rank0.Compare(rank1)

	if cmp > 0 {
		// Player 0 wins
		return [2]float64{node.Pot, 0}
	} else if cmp < 0 {
		// Player 1 wins
		return [2]float64{0, node.Pot}
	} else {
		// Tie (split pot)
		return [2]float64{node.Pot / 2, node.Pot / 2}
	}
}

// sampleAction samples an action index according to the given strategy
func (m *MCCFR) sampleAction(strategy []float64) int {
	if len(strategy) == 0 {
		return 0
	}

	// Sample according to cumulative probabilities
	r := m.rng.Float64()
	cumulative := 0.0
	for i, prob := range strategy {
		cumulative += prob
		if r <= cumulative {
			return i
		}
	}

	// Shouldn't happen unless there's floating point error
	return len(strategy) - 1
}

// GetProfile returns the current strategy profile
func (m *MCCFR) GetProfile() *StrategyProfile {
	return m.profile
}
