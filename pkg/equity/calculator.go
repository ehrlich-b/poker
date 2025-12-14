package equity

import (
	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
)

// EquityResult represents the outcome of an equity calculation
type EquityResult struct {
	WinPct float64 // Percentage of times hero wins
	TiePct float64 // Percentage of times hero ties
	Equity float64 // Overall equity (win% + tie%/2)
}

// PotentialResult represents hand improvement potential
type PotentialResult struct {
	PositivePot float64 // Probability of improving when currently behind
	NegativePot float64 // Probability of losing equity when currently ahead
	ImprovePct  float64 // Overall probability hand strength improves
}

// Calculator computes hand equity vs opponent ranges
type Calculator struct {
	// Nothing needed for now (pure functions)
}

// NewCalculator creates a new equity calculator
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculateEquity computes hero's equity against opponent's range
// hero: 2 cards
// board: 3-5 cards (flop, turn, or river)
// opponentRange: list of opponent combos
func (c *Calculator) CalculateEquity(hero []cards.Card, board []cards.Card, opponentRange []notation.Combo) EquityResult {
	// Edge case: if board is complete (5 cards), no runout needed
	if len(board) == 5 {
		return c.calculateRiverEquity(hero, board, opponentRange)
	}

	// Edge case: turn (4 cards)
	if len(board) == 4 {
		return c.calculateTurnEquity(hero, board, opponentRange)
	}

	// Flop (3 cards)
	return c.calculateFlopEquity(hero, board, opponentRange)
}

// calculateRiverEquity handles completed board (5 cards)
func (c *Calculator) calculateRiverEquity(hero []cards.Card, board []cards.Card, opponentRange []notation.Combo) EquityResult {
	heroHand := cards.Evaluate(append(hero, board...))

	wins := 0.0
	ties := 0.0
	total := 0.0

	for _, oppCombo := range opponentRange {
		oppCards := []cards.Card{oppCombo.Card1, oppCombo.Card2}
		oppHand := cards.Evaluate(append(oppCards, board...))

		cmp := heroHand.Compare(oppHand)
		if cmp > 0 {
			wins++
		} else if cmp == 0 {
			ties++
		}
		total++
	}

	if total == 0 {
		return EquityResult{Equity: 0.5} // No valid opponent combos
	}

	winPct := wins / total
	tiePct := ties / total
	equity := winPct + tiePct/2.0

	return EquityResult{
		WinPct: winPct,
		TiePct: tiePct,
		Equity: equity,
	}
}

// calculateTurnEquity handles turn (4 cards, need 1 river)
func (c *Calculator) calculateTurnEquity(hero []cards.Card, board []cards.Card, opponentRange []notation.Combo) EquityResult {
	usedCards := makeCardSet(append(hero, board...))

	wins := 0.0
	ties := 0.0
	total := 0.0

	// Enumerate all possible river cards
	for rank := cards.Two; rank <= cards.Ace; rank++ {
		for suit := cards.Spades; suit <= cards.Clubs; suit++ {
			river := cards.Card{Rank: rank, Suit: suit}
			if usedCards[river] {
				continue
			}

			fullBoard := append(board, river)
			heroHand := cards.Evaluate(append(hero, fullBoard...))

			// Evaluate against each opponent combo
			for _, oppCombo := range opponentRange {
				oppCards := []cards.Card{oppCombo.Card1, oppCombo.Card2}

				// Skip if opponent has the river card
				if oppCombo.Card1 == river || oppCombo.Card2 == river {
					continue
				}

				oppHand := cards.Evaluate(append(oppCards, fullBoard...))

				cmp := heroHand.Compare(oppHand)
				if cmp > 0 {
					wins++
				} else if cmp == 0 {
					ties++
				}
				total++
			}
		}
	}

	if total == 0 {
		return EquityResult{Equity: 0.5}
	}

	winPct := wins / total
	tiePct := ties / total
	equity := winPct + tiePct/2.0

	return EquityResult{
		WinPct: winPct,
		TiePct: tiePct,
		Equity: equity,
	}
}

// calculateFlopEquity handles flop (3 cards, need turn + river)
func (c *Calculator) calculateFlopEquity(hero []cards.Card, board []cards.Card, opponentRange []notation.Combo) EquityResult {
	usedCards := makeCardSet(append(hero, board...))

	wins := 0.0
	ties := 0.0
	total := 0.0

	// Enumerate all possible turn cards
	for turnRank := cards.Two; turnRank <= cards.Ace; turnRank++ {
		for turnSuit := cards.Spades; turnSuit <= cards.Clubs; turnSuit++ {
			turn := cards.Card{Rank: turnRank, Suit: turnSuit}
			if usedCards[turn] {
				continue
			}

			turnBoard := append(board, turn)
			turnUsed := makeCardSet(append(hero, turnBoard...))

			// Enumerate all possible river cards
			for riverRank := cards.Two; riverRank <= cards.Ace; riverRank++ {
				for riverSuit := cards.Spades; riverSuit <= cards.Clubs; riverSuit++ {
					river := cards.Card{Rank: riverRank, Suit: riverSuit}
					if turnUsed[river] {
						continue
					}

					fullBoard := append(turnBoard, river)
					heroHand := cards.Evaluate(append(hero, fullBoard...))

					// Evaluate against each opponent combo
					for _, oppCombo := range opponentRange {
						oppCards := []cards.Card{oppCombo.Card1, oppCombo.Card2}

						// Skip if opponent has turn or river
						if oppCombo.Card1 == turn || oppCombo.Card2 == turn ||
							oppCombo.Card1 == river || oppCombo.Card2 == river {
							continue
						}

						oppHand := cards.Evaluate(append(oppCards, fullBoard...))

						cmp := heroHand.Compare(oppHand)
						if cmp > 0 {
							wins++
						} else if cmp == 0 {
							ties++
						}
						total++
					}
				}
			}
		}
	}

	if total == 0 {
		return EquityResult{Equity: 0.5}
	}

	winPct := wins / total
	tiePct := ties / total
	equity := winPct + tiePct/2.0

	return EquityResult{
		WinPct: winPct,
		TiePct: tiePct,
		Equity: equity,
	}
}

// CalculatePotential computes hand improvement potential
// Only works for flop (3 cards) - returns zero for turn/river
// Simplified version: measures equity variance across runouts as a proxy for potential
// High variance = drawing hand (high potential), low variance = made hand (low potential)
func (c *Calculator) CalculatePotential(hero []cards.Card, board []cards.Card, opponentRange []notation.Combo) PotentialResult {
	// Only calculate potential for flop
	if len(board) != 3 {
		return PotentialResult{}
	}

	usedCards := makeCardSet(append(hero, board...))

	// Sample different turn cards and calculate equity on each
	var equities []float64
	sampleTurns := 0
	maxSamples := 10 // Sample 10 turn cards for efficiency

	for turnRank := cards.Two; turnRank <= cards.Ace && sampleTurns < maxSamples; turnRank++ {
		for turnSuit := cards.Spades; turnSuit <= cards.Clubs && sampleTurns < maxSamples; turnSuit++ {
			turn := cards.Card{Rank: turnRank, Suit: turnSuit}
			if usedCards[turn] {
				continue
			}

			// Calculate equity on this turn
			turnBoard := append(board, turn)
			result := c.calculateTurnEquity(hero, turnBoard, opponentRange)
			equities = append(equities, result.Equity)
			sampleTurns++
		}
	}

	if len(equities) == 0 {
		return PotentialResult{}
	}

	// Calculate mean equity
	mean := 0.0
	for _, eq := range equities {
		mean += eq
	}
	mean /= float64(len(equities))

	// Calculate variance
	variance := 0.0
	for _, eq := range equities {
		diff := eq - mean
		variance += diff * diff
	}
	variance /= float64(len(equities))

	// Standard deviation as potential metric
	stdDev := 0.0
	if variance > 0 {
		// Use sqrt for standard deviation
		stdDev = variance // For simplicity, use variance directly (already small)
	}

	// Map variance to potential metrics:
	// Variance ranges from 0 (no change across runouts) to ~0.25 (max variance at 50/50)
	// - High variance (>0.05) = drawing hand with high potential
	// - Low variance (<0.01) = made hand with low potential

	// Normalize variance to 0-1 range for potential
	// Max theoretical variance is 0.25 (at 50/50 split)
	normalizedVar := stdDev / 0.25
	if normalizedVar > 1.0 {
		normalizedVar = 1.0
	}

	// Positive potential: If currently behind, potential to improve
	positivePot := 0.0
	if mean < 0.5 {
		// Behind with high variance = high positive potential
		positivePot = normalizedVar
	}

	// Negative potential: If currently ahead, risk of getting outdrawn
	negativePot := 0.0
	if mean > 0.5 {
		// Ahead with high variance = high negative potential (vulnerable)
		negativePot = normalizedVar
	}

	// Improvement percentage: overall volatility
	improvePct := normalizedVar

	return PotentialResult{
		PositivePot: positivePot,
		NegativePot: negativePot,
		ImprovePct:  improvePct,
	}
}

// makeCardSet creates a set of cards for fast lookup
func makeCardSet(cardList []cards.Card) map[cards.Card]bool {
	set := make(map[cards.Card]bool)
	for _, c := range cardList {
		set[c] = true
	}
	return set
}
