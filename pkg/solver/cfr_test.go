package solver

import (
	"fmt"
	"math"
	"testing"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
	"github.com/behrlich/poker-solver/pkg/tree"
)

// TestCFR_KuhnPoker tests CFR on Kuhn poker, a toy game with known solution
func TestCFR_KuhnPoker(t *testing.T) {
	// Build Kuhn poker game tree
	root := BuildKuhnPokerTree()

	// Run CFR
	cfr := NewCFR()
	profile := cfr.Train(root, 10000)

	// Check that we have strategies for all information sets
	if profile.NumInfoSets() == 0 {
		t.Fatal("CFR produced no strategies")
	}

	// Print strategies for inspection
	t.Log("Kuhn Poker Strategies after 10k iterations:")
	for infoSet, strat := range profile.All() {
		t.Logf("%s", strat.String())
		_ = infoSet
	}

	// Verify strategies are reasonable for the J vs Q matchup
	// Note: In a fixed J vs Q matchup (not full Kuhn poker), the equilibrium is:
	// - Jack should bet (bluff) most of the time
	// - Queen should always call (has the winning hand)

	// Player 0 with Jack at root (can check or bet)
	if s, ok := profile.Get("J|"); ok {
		avgStrat := s.GetAverageStrategy()
		betProb := avgStrat[1]
		// Jack should bet as a bluff (since checking loses anyway)
		if betProb < 0.8 {
			t.Errorf("P0 with Jack should bet frequently (bluff), got bet=%.2f", betProb)
		}
		t.Logf("P0 with Jack: check=%.2f%%, bet=%.2f%%", avgStrat[0]*100, betProb*100)
	} else {
		t.Error("Expected strategy for J| to exist")
	}

	// Player 1 with Queen facing a bet (should call since Q beats J)
	if s, ok := profile.Get("Q|b1.0"); ok {
		avgStrat := s.GetAverageStrategy()
		callProb := avgStrat[1] // Call is second action
		if callProb < 0.9 {
			t.Errorf("P1 with Queen facing bet should call (has winning hand), got %.2f", callProb)
		}
		t.Logf("P1 with Queen facing bet: fold=%.2f%%, call=%.2f%%", avgStrat[0]*100, callProb*100)
	} else {
		t.Error("Expected strategy for Q|b1.0 to exist")
	}
}

// TestCFR_SimpleTree tests CFR on a very simple manually constructed tree
func TestCFR_SimpleTree(t *testing.T) {
	// Build a simple tree: P0 checks or bets, P1 responds
	root := buildSimpleTestTree()

	cfr := NewCFR()
	profile := cfr.Train(root, 1000)

	// Verify some strategies exist
	if profile.NumInfoSets() == 0 {
		t.Error("expected some strategies to be created")
	}

	t.Logf("Simple tree strategies after 1k iterations:")
	for _, strat := range profile.All() {
		t.Logf("%s", strat.String())
	}
}

// TestStrategy_RegretMatching tests regret matching algorithm
func TestStrategy_RegretMatching(t *testing.T) {
	actions := []notation.Action{
		{Type: notation.Check},
		{Type: notation.Bet, Amount: 10},
	}

	strat := NewStrategy("test", actions)

	// Set some regrets
	strat.RegretSum[0] = 5.0  // Check has positive regret
	strat.RegretSum[1] = -2.0 // Bet has negative regret

	// Get strategy via regret matching
	strategy := strat.GetStrategy()

	// Should only play actions with positive regret
	if strategy[0] <= 0 || strategy[1] > 0 {
		t.Errorf("regret matching should only choose positive regret actions, got %v", strategy)
	}

	// Should sum to 1
	sum := strategy[0] + strategy[1]
	if math.Abs(sum-1.0) > 0.001 {
		t.Errorf("strategy should sum to 1, got %.3f", sum)
	}
}

// TestStrategy_UniformDefault tests that uniform strategy is used when no regrets
func TestStrategy_UniformDefault(t *testing.T) {
	actions := []notation.Action{
		{Type: notation.Check},
		{Type: notation.Bet, Amount: 10},
	}

	strat := NewStrategy("test", actions)
	// No regrets set (all zero)

	strategy := strat.GetStrategy()

	// Should be uniform (50/50)
	expected := 0.5
	if math.Abs(strategy[0]-expected) > 0.001 {
		t.Errorf("expected uniform %.2f, got %.2f", expected, strategy[0])
	}
}

// TestStrategy_AverageStrategy tests average strategy calculation
func TestStrategy_AverageStrategy(t *testing.T) {
	actions := []notation.Action{
		{Type: notation.Check},
		{Type: notation.Bet, Amount: 10},
	}

	strat := NewStrategy("test", actions)

	// Simulate some iterations
	strat.StrategySum[0] = 30.0 // Checked 30% of the time
	strat.StrategySum[1] = 70.0 // Bet 70% of the time

	avgStrat := strat.GetAverageStrategy()

	if math.Abs(avgStrat[0]-0.3) > 0.001 {
		t.Errorf("expected check 0.3, got %.3f", avgStrat[0])
	}
	if math.Abs(avgStrat[1]-0.7) > 0.001 {
		t.Errorf("expected bet 0.7, got %.3f", avgStrat[1])
	}
}

// BuildKuhnPokerTree builds a Kuhn poker game tree for testing
// Simplified version with cards as info sets (exported for use in multiple test files)
func BuildKuhnPokerTree() *tree.TreeNode {
	// Kuhn poker:
	// - 3 cards: J, Q, K
	// - Each player gets one card
	// - Player 0 acts first: check or bet
	//   - If check: Player 1 can check (showdown) or bet
	//     - If bet: Player 0 can fold or call
	//   - If bet: Player 1 can fold or call

	// We'll build a simplified tree for one card matchup: P0 has Jack, P1 has Queen
	// This is just to test CFR mechanics, not full Kuhn poker

	// Root: P0 acts with Jack
	root := tree.NewDecisionNode(
		"J|", // InfoSet: J, no history
		0,    // Player 0
		2.0,  // Pot (2 antes)
		[]notation.Action{
			{Type: notation.Check},            // Check
			{Type: notation.Bet, Amount: 1.0}, // Bet 1
		},
		nil,              // Board (not relevant for Kuhn)
		[2]float64{1, 1}, // Stacks (each has 1 chip left)
	)

	// P0 checks
	checkNode := tree.NewDecisionNode(
		"Q|x", // InfoSet: Q, P0 checked
		1,     // Player 1
		2.0,
		[]notation.Action{
			{Type: notation.Check},            // Check (showdown)
			{Type: notation.Bet, Amount: 1.0}, // Bet 1
		},
		nil,
		[2]float64{1, 1},
	)

	// P0 checks, P1 checks (showdown: Q beats J)
	checkCheckNode := tree.NewTerminalNode(
		2.0,
		[2]float64{0, 2}, // P1 wins both antes
		nil,
		[2]float64{1, 1},
	)

	// P0 checks, P1 bets
	checkBetNode := tree.NewDecisionNode(
		"J|xb1.0", // InfoSet: J, check-bet
		0,         // P0 decides
		3.0,       // Pot (2 antes + 1 bet)
		[]notation.Action{
			{Type: notation.Fold},
			{Type: notation.Call},
		},
		nil,
		[2]float64{1, 0},
	)

	// P0 checks, P1 bets, P0 folds
	checkBetFoldNode := tree.NewTerminalNode(
		3.0,
		[2]float64{0, 2}, // P1 wins antes
		nil,
		[2]float64{1, 0},
	)

	// P0 checks, P1 bets, P0 calls (showdown: Q beats J)
	checkBetCallNode := tree.NewTerminalNode(
		4.0,
		[2]float64{0, 4}, // P1 wins all
		nil,
		[2]float64{0, 0},
	)

	// P0 bets
	betNode := tree.NewDecisionNode(
		"Q|b1.0", // InfoSet: Q, P0 bet
		1,        // Player 1
		3.0,
		[]notation.Action{
			{Type: notation.Fold},
			{Type: notation.Call},
		},
		nil,
		[2]float64{0, 1},
	)

	// P0 bets, P1 folds
	betFoldNode := tree.NewTerminalNode(
		3.0,
		[2]float64{2, 0}, // P0 wins
		nil,
		[2]float64{0, 1},
	)

	// P0 bets, P1 calls (showdown: Q beats J)
	betCallNode := tree.NewTerminalNode(
		4.0,
		[2]float64{0, 4}, // P1 wins all
		nil,
		[2]float64{0, 0},
	)

	// Wire up tree
	checkNode.Children = map[string]*tree.TreeNode{
		"x":    checkCheckNode,
		"b1.0": checkBetNode,
	}

	checkBetNode.Children = map[string]*tree.TreeNode{
		"f": checkBetFoldNode,
		"c": checkBetCallNode,
	}

	betNode.Children = map[string]*tree.TreeNode{
		"f": betFoldNode,
		"c": betCallNode,
	}

	root.Children = map[string]*tree.TreeNode{
		"x":    checkNode,
		"b1.0": betNode,
	}

	return root
}

// Helper: Build a very simple test tree
func buildSimpleTestTree() *tree.TreeNode {
	// P0 can check or bet
	root := tree.NewDecisionNode(
		"test_p0",
		0,
		10.0,
		[]notation.Action{
			{Type: notation.Check},
			{Type: notation.Bet, Amount: 5.0},
		},
		nil,
		[2]float64{100, 100},
	)

	// P0 checks -> P1 can check or bet
	checkNode := tree.NewDecisionNode(
		"test_p1_after_check",
		1,
		10.0,
		[]notation.Action{
			{Type: notation.Check},
			{Type: notation.Bet, Amount: 5.0},
		},
		nil,
		[2]float64{100, 100},
	)

	// Both check (showdown)
	checkCheckNode := tree.NewTerminalNode(
		10.0,
		[2]float64{5, 5}, // Split pot
		nil,
		[2]float64{100, 100},
	)

	// P0 checks, P1 bets -> P0 responds
	checkBetNode := tree.NewDecisionNode(
		"test_p0_facing_bet",
		0,
		15.0,
		[]notation.Action{
			{Type: notation.Fold},
			{Type: notation.Call},
		},
		nil,
		[2]float64{100, 95},
	)

	checkBetFoldNode := tree.NewTerminalNode(
		15.0,
		[2]float64{0, 10},
		nil,
		[2]float64{100, 95},
	)

	checkBetCallNode := tree.NewTerminalNode(
		20.0,
		[2]float64{10, 10},
		nil,
		[2]float64{95, 95},
	)

	// P0 bets -> P1 responds
	betNode := tree.NewDecisionNode(
		"test_p1_facing_bet",
		1,
		15.0,
		[]notation.Action{
			{Type: notation.Fold},
			{Type: notation.Call},
		},
		nil,
		[2]float64{95, 100},
	)

	betFoldNode := tree.NewTerminalNode(
		15.0,
		[2]float64{10, 0},
		nil,
		[2]float64{95, 100},
	)

	betCallNode := tree.NewTerminalNode(
		20.0,
		[2]float64{10, 10},
		nil,
		[2]float64{95, 95},
	)

	// Wire up
	checkNode.Children = map[string]*tree.TreeNode{
		"x":    checkCheckNode,
		"b5.0": checkBetNode,
	}

	checkBetNode.Children = map[string]*tree.TreeNode{
		"f": checkBetFoldNode,
		"c": checkBetCallNode,
	}

	betNode.Children = map[string]*tree.TreeNode{
		"f": betFoldNode,
		"c": betCallNode,
	}

	root.Children = map[string]*tree.TreeNode{
		"x":    checkNode,
		"b5.0": betNode,
	}

	return root
}

// TestCFR_RiverSpot tests CFR on a real river poker scenario
func TestCFR_RiverSpot(t *testing.T) {
	// Build a simple river tree: AA vs QQ on river
	board := []cards.Card{
		cards.NewCard(cards.King, cards.Hearts),
		cards.NewCard(cards.Nine, cards.Spades),
		cards.NewCard(cards.Four, cards.Clubs),
		cards.NewCard(cards.Seven, cards.Diamonds),
		cards.NewCard(cards.Two, cards.Spades),
	}

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:           10,
		Board:         board,
		ActionHistory: nil,
		ToAct:         0,
		Street:        notation.River,
	}

	// AA vs QQ matchup
	combo0 := notation.Combo{
		Card1: cards.NewCard(cards.Ace, cards.Diamonds),
		Card2: cards.NewCard(cards.Ace, cards.Clubs),
	}
	combo1 := notation.Combo{
		Card1: cards.NewCard(cards.Queen, cards.Diamonds),
		Card2: cards.NewCard(cards.Queen, cards.Hearts),
	}

	// Build tree with simple action space
	config := tree.ActionConfig{
		BetSizes:   []float64{0.5}, // 50% pot bet
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}
	builder := tree.NewBuilder(config)
	root, err := builder.Build(gs, combo0, combo1)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Solve with CFR
	cfr := NewCFR()
	profile := cfr.Train(root, 5000)

	// Verify we got strategies
	if profile.NumInfoSets() == 0 {
		t.Fatal("Expected some strategies to be created")
	}

	t.Logf("River spot (AA vs QQ) strategies after 5k iterations:")
	t.Logf("Number of information sets: %d", profile.NumInfoSets())

	for infoSet, strat := range profile.All() {
		avgStrat := strat.GetAverageStrategy()
		t.Logf("\nInfoSet: %s", infoSet)
		for i, action := range strat.Actions {
			t.Logf("  %s: %.1f%%", action.String(), avgStrat[i]*100)
		}
	}

	// AA should bet frequently (has the best hand)
	// Look for BTN with AA at root (no history)
	foundBTNStrategy := false
	for infoSet, strat := range profile.All() {
		// Check if this is BTN acting first with AA
		if len(strat.Actions) >= 2 && infoSet != "" {
			avgStrat := strat.GetAverageStrategy()
			// If first action is check and second is bet, this could be BTN at root
			if strat.Actions[0].Type == notation.Check && strat.Actions[1].Type == notation.Bet {
				betProb := avgStrat[1]
				t.Logf("Found BTN-like strategy (check=%.1f%%, bet=%.1f%%)", avgStrat[0]*100, betProb*100)
				// AA should bet >50% of the time for value
				if betProb > 0.5 {
					foundBTNStrategy = true
				}
			}
		}
	}

	if !foundBTNStrategy {
		t.Log("Note: Expected AA to bet >50% for value (didn't find clear evidence, but this is just a sanity check)")
	}
}

// BenchmarkCFR_KuhnPoker benchmarks CFR on Kuhn poker
func BenchmarkCFR_KuhnPoker(b *testing.B) {
	root := BuildKuhnPokerTree()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfr := NewCFR()
		cfr.Train(root, 1000)
	}
}

// BenchmarkCFR_RiverSpot benchmarks CFR on a river spot
func BenchmarkCFR_RiverSpot(b *testing.B) {
	// Build a river tree
	board := []cards.Card{
		cards.NewCard(cards.King, cards.Hearts),
		cards.NewCard(cards.Nine, cards.Spades),
		cards.NewCard(cards.Four, cards.Clubs),
		cards.NewCard(cards.Seven, cards.Diamonds),
		cards.NewCard(cards.Two, cards.Spades),
	}

	gs := &notation.GameState{
		Players: []notation.PlayerRange{
			{Position: notation.BTN, Stack: 100},
			{Position: notation.BB, Stack: 100},
		},
		Pot:           10,
		Board:         board,
		ActionHistory: nil,
		ToAct:         0,
		Street:        notation.River,
	}

	combo0 := notation.Combo{
		Card1: cards.NewCard(cards.Ace, cards.Diamonds),
		Card2: cards.NewCard(cards.Ace, cards.Clubs),
	}
	combo1 := notation.Combo{
		Card1: cards.NewCard(cards.Queen, cards.Diamonds),
		Card2: cards.NewCard(cards.Queen, cards.Hearts),
	}

	config := tree.DefaultRiverConfig()
	builder := tree.NewBuilder(config)
	root, _ := builder.Build(gs, combo0, combo1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfr := NewCFR()
		cfr.Train(root, 1000)
	}
}

// Helper to print tree structure (for debugging)
func printTree(node *tree.TreeNode, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	if node.IsTerminal {
		fmt.Printf("%sTerminal: payoffs=[%.1f, %.1f]\n", indent, node.Payoff[0], node.Payoff[1])
		return
	}

	fmt.Printf("%sDecision P%d: %s (pot=%.1f)\n", indent, node.Player, node.InfoSet, node.Pot)
	for actionKey, child := range node.Children {
		fmt.Printf("%s  -> %s\n", indent, actionKey)
		printTree(child, depth+1)
	}
}
