package notation

import (
	"fmt"
	"strings"

	"github.com/behrlich/poker-solver/pkg/cards"
)

// Combo represents a specific 2-card combination (hole cards)
type Combo struct {
	Card1 cards.Card
	Card2 cards.Card
}

// String returns the combo in standard notation (e.g., "AsKh")
func (c Combo) String() string {
	return fmt.Sprintf("%s%s", c.Card1, c.Card2)
}

// ParseRange parses a range string and returns all possible combos
// Examples:
//   - "AA" → 6 combos (AsAh, AsAd, AsAc, AhAd, AhAc, AdAc)
//   - "AKs" → 4 combos (AsKs, AhKh, AdKd, AcKc)
//   - "AKo" → 12 combos (all offsuit combinations)
//   - "KK-JJ" → 18 combos (KK, QQ, JJ)
//   - "AA,KK,AKs" → 6+6+4 = 16 combos
func ParseRange(rangeStr string) ([]Combo, error) {
	rangeStr = strings.TrimSpace(rangeStr)
	if rangeStr == "" {
		return nil, fmt.Errorf("empty range string")
	}

	// Split by comma to get individual range components
	parts := strings.Split(rangeStr, ",")

	var allCombos []Combo
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if this is a range (contains dash)
		if strings.Contains(part, "-") {
			combos, err := parseRangeWithDash(part)
			if err != nil {
				return nil, fmt.Errorf("error parsing range %q: %w", part, err)
			}
			allCombos = append(allCombos, combos...)
		} else {
			// Single hand notation
			combos, err := parseSingleHand(part)
			if err != nil {
				return nil, fmt.Errorf("error parsing hand %q: %w", part, err)
			}
			allCombos = append(allCombos, combos...)
		}
	}

	return allCombos, nil
}

// parseSingleHand parses a single hand notation (e.g., "AA", "AKs", "AKo")
func parseSingleHand(hand string) ([]Combo, error) {
	hand = strings.TrimSpace(hand)

	// Must be 2 or 3 characters (e.g., "AA" or "AKs")
	if len(hand) < 2 || len(hand) > 3 {
		return nil, fmt.Errorf("invalid hand notation: %q", hand)
	}

	rank1, err := parseRankChar(hand[0])
	if err != nil {
		return nil, err
	}

	rank2, err := parseRankChar(hand[1])
	if err != nil {
		return nil, err
	}

	// Determine if suited or offsuit
	var suited bool
	if len(hand) == 3 {
		switch hand[2] {
		case 's', 'S':
			suited = true
		case 'o', 'O':
			suited = false
		default:
			return nil, fmt.Errorf("invalid suited/offsuit indicator: %c (expected 's' or 'o')", hand[2])
		}
	} else {
		// No indicator means it's a pair
		if rank1 != rank2 {
			return nil, fmt.Errorf("ambiguous hand %q (use 's' for suited or 'o' for offsuit)", hand)
		}
	}

	return generateCombos(rank1, rank2, suited), nil
}

// parseRangeWithDash parses a range with a dash (e.g., "KK-JJ", "AKs-ATs")
func parseRangeWithDash(rangeStr string) ([]Combo, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format: %q (expected format: AA-KK)", rangeStr)
	}

	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])

	// Parse start and end hands
	startRank1, startRank2, startSuited, err := parseHandComponents(start)
	if err != nil {
		return nil, fmt.Errorf("invalid start hand %q: %w", start, err)
	}

	endRank1, endRank2, endSuited, err := parseHandComponents(end)
	if err != nil {
		return nil, fmt.Errorf("invalid end hand %q: %w", end, err)
	}

	// Validate that suited/offsuit matches
	if startSuited != endSuited {
		return nil, fmt.Errorf("mismatched suited/offsuit in range %q", rangeStr)
	}

	var allCombos []Combo

	// Handle pair ranges (e.g., "KK-JJ")
	if startRank1 == startRank2 && endRank1 == endRank2 {
		// Iterate from start rank down to end rank
		for r := int(startRank1); r >= int(endRank1); r-- {
			rank := cards.Rank(r)
			allCombos = append(allCombos, generateCombos(rank, rank, startSuited)...)
		}
		return allCombos, nil
	}

	// Handle non-pair ranges (e.g., "AKs-ATs", "KQo-KJo")
	// First rank must be the same for both
	if startRank1 != endRank1 {
		return nil, fmt.Errorf("invalid range %q (first rank must match)", rangeStr)
	}

	// Iterate from start second rank down to end second rank
	for r := int(startRank2); r >= int(endRank2); r-- {
		rank2 := cards.Rank(r)
		allCombos = append(allCombos, generateCombos(startRank1, rank2, startSuited)...)
	}

	return allCombos, nil
}

// parseHandComponents parses hand notation and returns (rank1, rank2, suited, error)
func parseHandComponents(hand string) (cards.Rank, cards.Rank, bool, error) {
	hand = strings.TrimSpace(hand)

	if len(hand) < 2 || len(hand) > 3 {
		return 0, 0, false, fmt.Errorf("invalid hand notation: %q", hand)
	}

	rank1, err := parseRankChar(hand[0])
	if err != nil {
		return 0, 0, false, err
	}

	rank2, err := parseRankChar(hand[1])
	if err != nil {
		return 0, 0, false, err
	}

	var suited bool
	if len(hand) == 3 {
		// Pairs cannot have suited/offsuit indicator
		if rank1 == rank2 {
			return 0, 0, false, fmt.Errorf("pair %q cannot have suited/offsuit indicator", hand)
		}

		switch hand[2] {
		case 's', 'S':
			suited = true
		case 'o', 'O':
			suited = false
		default:
			return 0, 0, false, fmt.Errorf("invalid suited/offsuit indicator: %c", hand[2])
		}
	} else {
		// No indicator - must be a pair
		if rank1 != rank2 {
			return 0, 0, false, fmt.Errorf("ambiguous hand %q", hand)
		}
	}

	return rank1, rank2, suited, nil
}

// parseRankChar converts a character to a Rank
func parseRankChar(b byte) (cards.Rank, error) {
	switch b {
	case 'A', 'a':
		return cards.Ace, nil
	case 'K', 'k':
		return cards.King, nil
	case 'Q', 'q':
		return cards.Queen, nil
	case 'J', 'j':
		return cards.Jack, nil
	case 'T', 't':
		return cards.Ten, nil
	case '9':
		return cards.Nine, nil
	case '8':
		return cards.Eight, nil
	case '7':
		return cards.Seven, nil
	case '6':
		return cards.Six, nil
	case '5':
		return cards.Five, nil
	case '4':
		return cards.Four, nil
	case '3':
		return cards.Three, nil
	case '2':
		return cards.Two, nil
	default:
		return 0, fmt.Errorf("invalid rank: %c", b)
	}
}

// generateCombos generates all possible card combinations for a given hand
func generateCombos(rank1, rank2 cards.Rank, suited bool) []Combo {
	var combos []Combo

	// All suits
	suits := []cards.Suit{cards.Spades, cards.Hearts, cards.Diamonds, cards.Clubs}

	if rank1 == rank2 {
		// Pair: generate all 6 combinations
		for i := 0; i < len(suits); i++ {
			for j := i + 1; j < len(suits); j++ {
				combos = append(combos, Combo{
					Card1: cards.NewCard(rank1, suits[i]),
					Card2: cards.NewCard(rank2, suits[j]),
				})
			}
		}
	} else if suited {
		// Suited: 4 combinations (one per suit)
		for _, suit := range suits {
			combos = append(combos, Combo{
				Card1: cards.NewCard(rank1, suit),
				Card2: cards.NewCard(rank2, suit),
			})
		}
	} else {
		// Offsuit: 12 combinations (all suit combinations except matching)
		for _, suit1 := range suits {
			for _, suit2 := range suits {
				if suit1 != suit2 {
					combos = append(combos, Combo{
						Card1: cards.NewCard(rank1, suit1),
						Card2: cards.NewCard(rank2, suit2),
					})
				}
			}
		}
	}

	return combos
}
