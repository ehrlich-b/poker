package cards

import (
	"sort"
)

// HandRank represents the category of a poker hand
type HandRank uint8

const (
	HighCard HandRank = iota
	OnePair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
)

// HandValue represents the complete value of a 5-card poker hand
// Higher values beat lower values
type HandValue struct {
	Rank   HandRank
	Values [5]Rank // Tiebreaker values (e.g., trip rank, kicker ranks)
}

// Compare returns -1 if h < other, 0 if equal, 1 if h > other
func (h HandValue) Compare(other HandValue) int {
	if h.Rank != other.Rank {
		if h.Rank < other.Rank {
			return -1
		}
		return 1
	}

	// Same rank, compare tiebreakers
	for i := 0; i < 5; i++ {
		if h.Values[i] != other.Values[i] {
			if h.Values[i] < other.Values[i] {
				return -1
			}
			return 1
		}
	}

	return 0
}

// Evaluate returns the best possible 5-card hand from 7 cards
func Evaluate(cards []Card) HandValue {
	if len(cards) != 7 {
		panic("Evaluate requires exactly 7 cards")
	}

	// Check all 21 possible 5-card combinations and return the best
	best := HandValue{Rank: HighCard}

	// Generate all combinations of 5 cards from 7
	for i := 0; i < 7; i++ {
		for j := i + 1; j < 7; j++ {
			for k := j + 1; k < 7; k++ {
				for l := k + 1; l < 7; l++ {
					for m := l + 1; m < 7; m++ {
						hand := []Card{cards[i], cards[j], cards[k], cards[l], cards[m]}
						value := evaluate5Cards(hand)
						if value.Compare(best) > 0 {
							best = value
						}
					}
				}
			}
		}
	}

	return best
}

// evaluate5Cards evaluates exactly 5 cards
func evaluate5Cards(cards []Card) HandValue {
	// Count ranks and suits
	rankCounts := make([]int, 13)
	suitCounts := make([]int, 4)

	for _, card := range cards {
		rankCounts[card.Rank]++
		suitCounts[card.Suit]++
	}

	// Check for flush
	isFlush := false
	for _, count := range suitCounts {
		if count == 5 {
			isFlush = true
			break
		}
	}

	// Check for straight
	isStraight, straightHigh := checkStraight(rankCounts)

	// Straight flush
	if isFlush && isStraight {
		return HandValue{
			Rank:   StraightFlush,
			Values: [5]Rank{straightHigh, 0, 0, 0, 0},
		}
	}

	// Count rank groups (for pairs, trips, quads)
	groups := getRankGroups(rankCounts)

	// Four of a kind
	if len(groups) >= 1 && groups[0].count == 4 {
		return HandValue{
			Rank:   FourOfAKind,
			Values: [5]Rank{groups[0].rank, groups[1].rank, 0, 0, 0},
		}
	}

	// Full house
	if len(groups) >= 2 && groups[0].count == 3 && groups[1].count == 2 {
		return HandValue{
			Rank:   FullHouse,
			Values: [5]Rank{groups[0].rank, groups[1].rank, 0, 0, 0},
		}
	}

	// Flush
	if isFlush {
		// Get all 5 ranks sorted descending
		ranks := make([]Rank, 0, 5)
		for r := int(Ace); r >= int(Two); r-- {
			rank := Rank(r)
			if rankCounts[rank] > 0 {
				ranks = append(ranks, rank)
			}
		}
		return HandValue{
			Rank:   Flush,
			Values: [5]Rank{ranks[0], ranks[1], ranks[2], ranks[3], ranks[4]},
		}
	}

	// Straight
	if isStraight {
		return HandValue{
			Rank:   Straight,
			Values: [5]Rank{straightHigh, 0, 0, 0, 0},
		}
	}

	// Three of a kind
	if len(groups) >= 1 && groups[0].count == 3 {
		return HandValue{
			Rank:   ThreeOfAKind,
			Values: [5]Rank{groups[0].rank, groups[1].rank, groups[2].rank, 0, 0},
		}
	}

	// Two pair
	if len(groups) >= 2 && groups[0].count == 2 && groups[1].count == 2 {
		return HandValue{
			Rank:   TwoPair,
			Values: [5]Rank{groups[0].rank, groups[1].rank, groups[2].rank, 0, 0},
		}
	}

	// One pair
	if len(groups) >= 1 && groups[0].count == 2 {
		return HandValue{
			Rank:   OnePair,
			Values: [5]Rank{groups[0].rank, groups[1].rank, groups[2].rank, groups[3].rank, 0},
		}
	}

	// High card
	return HandValue{
		Rank:   HighCard,
		Values: [5]Rank{groups[0].rank, groups[1].rank, groups[2].rank, groups[3].rank, groups[4].rank},
	}
}

type rankGroup struct {
	rank  Rank
	count int
}

// getRankGroups returns ranks grouped by count, sorted by count descending, then rank descending
func getRankGroups(rankCounts []int) []rankGroup {
	groups := make([]rankGroup, 0, 5)

	// Iterate from Ace down to Two using int to avoid underflow
	for r := int(Ace); r >= int(Two); r-- {
		rank := Rank(r)
		if rankCounts[rank] > 0 {
			groups = append(groups, rankGroup{rank: rank, count: rankCounts[rank]})
		}
	}

	// Sort by count descending, then by rank descending
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].count != groups[j].count {
			return groups[i].count > groups[j].count
		}
		return groups[i].rank > groups[j].rank
	})

	return groups
}

// checkStraight checks if the ranks form a straight
// Returns (isStraight, highCard)
func checkStraight(rankCounts []int) (bool, Rank) {
	// Check for regular straights (A-high down to 6-high)
	// Note: 5-high (wheel) is a special case handled below
	for h := int(Ace); h >= int(Six); h-- {
		high := Rank(h)
		hasStraight := true
		for i := 0; i < 5; i++ {
			rank := Rank(int(high) - i)
			if rankCounts[rank] == 0 {
				hasStraight = false
				break
			}
		}
		if hasStraight {
			return true, high
		}
	}

	// Check for wheel (A-2-3-4-5)
	// This is a special case because Ace acts as a low card
	if rankCounts[Ace] > 0 && rankCounts[Two] > 0 && rankCounts[Three] > 0 &&
		rankCounts[Four] > 0 && rankCounts[Five] > 0 {
		return true, Five // In a wheel, the high card is 5
	}

	return false, 0
}

// String returns a human-readable representation of the hand rank
func (r HandRank) String() string {
	switch r {
	case HighCard:
		return "High Card"
	case OnePair:
		return "One Pair"
	case TwoPair:
		return "Two Pair"
	case ThreeOfAKind:
		return "Three of a Kind"
	case Straight:
		return "Straight"
	case Flush:
		return "Flush"
	case FullHouse:
		return "Full House"
	case FourOfAKind:
		return "Four of a Kind"
	case StraightFlush:
		return "Straight Flush"
	default:
		return "Unknown"
	}
}
