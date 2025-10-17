package cards

import (
	"testing"
)

func TestParseCard(t *testing.T) {
	tests := []struct {
		input    string
		wantRank Rank
		wantSuit Suit
		wantErr  bool
	}{
		{"As", Ace, Spades, false},
		{"Kh", King, Hearts, false},
		{"Qd", Queen, Diamonds, false},
		{"Jc", Jack, Clubs, false},
		{"Ts", Ten, Spades, false},
		{"9h", Nine, Hearts, false},
		{"2c", Two, Clubs, false},
		{"as", Ace, Spades, false},   // lowercase should work
		{"TD", Ten, Diamonds, false}, // mixed case
		{"", 0, 0, true},             // empty
		{"A", 0, 0, true},            // too short
		{"Asx", 0, 0, true},          // too long
		{"Xx", 0, 0, true},           // invalid rank
		{"Ax", 0, 0, true},           // invalid suit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCard(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCard(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Rank != tt.wantRank || got.Suit != tt.wantSuit {
					t.Errorf("ParseCard(%q) = %v, want Rank=%v Suit=%v", tt.input, got, tt.wantRank, tt.wantSuit)
				}
			}
		})
	}
}

func TestCardString(t *testing.T) {
	tests := []struct {
		card Card
		want string
	}{
		{Card{Ace, Spades}, "As"},
		{Card{King, Hearts}, "Kh"},
		{Card{Ten, Diamonds}, "Td"},
		{Card{Two, Clubs}, "2c"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.card.String(); got != tt.want {
				t.Errorf("Card.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCards(t *testing.T) {
	tests := []struct {
		input   string
		want    []Card
		wantErr bool
	}{
		{
			"AsKh",
			[]Card{{Ace, Spades}, {King, Hearts}},
			false,
		},
		{
			"As Kh Qd",
			[]Card{{Ace, Spades}, {King, Hearts}, {Queen, Diamonds}},
			false,
		},
		{
			"2s3h4d5c6s",
			[]Card{{Two, Spades}, {Three, Hearts}, {Four, Diamonds}, {Five, Clubs}, {Six, Spades}},
			false,
		},
		{
			"A", // odd length
			nil,
			true,
		},
		{
			"AsXx", // invalid card
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCards(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCards(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseCards(%q) returned %d cards, want %d", tt.input, len(got), len(tt.want))
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("ParseCards(%q)[%d] = %v, want %v", tt.input, i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that parsing a card and converting back to string gives the same result
	inputs := []string{"As", "Kh", "Qd", "Jc", "Ts", "9h", "2c"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			card, err := ParseCard(input)
			if err != nil {
				t.Fatalf("ParseCard(%q) error = %v", input, err)
			}
			got := card.String()
			if got != input {
				t.Errorf("Round trip failed: %q -> %v -> %q", input, card, got)
			}
		})
	}
}
