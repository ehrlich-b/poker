package cards

import (
	"testing"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name     string
		cards    string
		wantRank HandRank
	}{
		// Straight flushes
		{
			"Royal flush",
			"AhKhQhJhTh2d3c",
			StraightFlush,
		},
		{
			"Straight flush",
			"9s8s7s6s5s2h3d",
			StraightFlush,
		},
		{
			"Wheel straight flush",
			"5d4d3d2dAd7h8c",
			StraightFlush,
		},

		// Four of a kind
		{
			"Quad aces",
			"AsAhAdAcKs2d3c",
			FourOfAKind,
		},
		{
			"Quad twos",
			"2s2h2d2cAhKsQd",
			FourOfAKind,
		},

		// Full house
		{
			"Aces full of kings",
			"AsAhAdKsKh2d3c",
			FullHouse,
		},
		{
			"Threes full of twos",
			"3s3h3d2s2hAcKd",
			FullHouse,
		},

		// Flush
		{
			"Ace-high flush",
			"AhKh9h5h2h3dQc",
			Flush,
		},
		{
			"King-high flush",
			"KsQs9s7s2s3h4d",
			Flush,
		},

		// Straight
		{
			"Broadway straight",
			"AhKdQcJs Ts 2h 3c",
			Straight,
		},
		{
			"Wheel straight",
			"Ah2s3d4c5h7s9d",
			Straight,
		},
		{
			"Seven-high straight",
			"7h6d5s4c3h2sAd",
			Straight,
		},

		// Three of a kind
		{
			"Trip aces",
			"AsAhAdKsQh2d3c",
			ThreeOfAKind,
		},

		// Two pair
		{
			"Aces and kings",
			"AsAhKdKs Qh 2d 3c",
			TwoPair,
		},
		{
			"Threes and twos",
			"3s3h2d2sAhKdQc",
			TwoPair,
		},

		// One pair
		{
			"Pair of aces",
			"AsAhKdQs Jh 9d 7c",
			OnePair,
		},
		{
			"Pair of twos",
			"2s2hAhKd9cJs7d", // Changed to avoid accidental straight
			OnePair,
		},

		// High card
		{
			"Ace high",
			"AhKd9s7c5h3d2s",
			HighCard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards, err := ParseCards(tt.cards)
			if err != nil {
				t.Fatalf("Failed to parse cards: %v", err)
			}

			got := Evaluate(cards)
			if got.Rank != tt.wantRank {
				t.Errorf("Evaluate(%v) = %v, want %v", tt.name, got.Rank, tt.wantRank)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name   string
		hand1  string
		hand2  string
		want   int // -1 if hand1 < hand2, 0 if equal, 1 if hand1 > hand2
	}{
		{
			"Straight flush beats quads",
			"9s8s7s6s5s2h3d", // Straight flush
			"AsAhAdAcKs2d3c", // Quad aces
			1,
		},
		{
			"Quads beat full house",
			"2s2h2d2cAhKsQd", // Quad twos
			"AsAhAdKsKh2d3c", // Aces full
			1,
		},
		{
			"Full house beats flush",
			"3s3h3d2s2hAcKd", // Threes full
			"AhKh9h5h2h3dQc", // Ace-high flush
			1,
		},
		{
			"Flush beats straight",
			"AhKh9h5h2h3dQc", // Ace-high flush
			"AhKdQcJsTs2h3c", // Broadway
			1,
		},
		{
			"Higher pair wins",
			"AsAhKdQsJh9d7c", // Pair of aces
			"KsKhAdQsJh9d7c", // Pair of kings
			1,
		},
		{
			"Same pair, higher kicker wins",
			"AsAhKdQsJh9d7c", // Pair of aces, K kicker
			"AdAcQh9s7d5c3h", // Pair of aces, Q kicker (avoid straight)
			1,
		},
		{
			"Identical hands tie",
			"AsAhKdQsJh9d7c",
			"AdAcKhQcJs9h7s",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards1, err := ParseCards(tt.hand1)
			if err != nil {
				t.Fatalf("Failed to parse hand1: %v", err)
			}

			cards2, err := ParseCards(tt.hand2)
			if err != nil {
				t.Fatalf("Failed to parse hand2: %v", err)
			}

			val1 := Evaluate(cards1)
			val2 := Evaluate(cards2)
			got := val1.Compare(val2)

			if got != tt.want {
				t.Errorf("Compare(%v vs %v) = %v, want %v\n  hand1: %v %v\n  hand2: %v %v",
					tt.name, tt.name, got, tt.want,
					val1.Rank, val1.Values,
					val2.Rank, val2.Values)
			}
		})
	}
}

func TestCheckStraight(t *testing.T) {
	tests := []struct {
		name     string
		ranks    []Rank
		wantIs   bool
		wantHigh Rank
	}{
		{
			"Broadway",
			[]Rank{Ace, King, Queen, Jack, Ten},
			true,
			Ace,
		},
		{
			"Wheel (A-2-3-4-5)",
			[]Rank{Ace, Two, Three, Four, Five},
			true,
			Five, // High card in wheel is 5
		},
		{
			"Seven high straight",
			[]Rank{Seven, Six, Five, Four, Three},
			true,
			Seven,
		},
		{
			"Not a straight (gap)",
			[]Rank{Ace, King, Queen, Jack, Nine},
			false,
			0,
		},
		{
			"Not a straight (pair)",
			[]Rank{Ace, Ace, King, Queen, Jack},
			false,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rankCounts := make([]int, 13)
			for _, rank := range tt.ranks {
				rankCounts[rank]++
			}

			gotIs, gotHigh := checkStraight(rankCounts)
			if gotIs != tt.wantIs || gotHigh != tt.wantHigh {
				t.Errorf("checkStraight(%v) = (%v, %v), want (%v, %v)",
					tt.name, gotIs, gotHigh, tt.wantIs, tt.wantHigh)
			}
		})
	}
}
