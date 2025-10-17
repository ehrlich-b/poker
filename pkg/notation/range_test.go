package notation

import (
	"testing"

	"github.com/behrlich/poker-solver/pkg/cards"
)

func TestParseRange_SinglePairs(t *testing.T) {
	tests := []struct {
		input     string
		wantCount int
	}{
		{"AA", 6},
		{"KK", 6},
		{"22", 6},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			combos, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange(%q) error = %v", tt.input, err)
			}
			if len(combos) != tt.wantCount {
				t.Errorf("ParseRange(%q) returned %d combos, want %d", tt.input, len(combos), tt.wantCount)
			}
			// Verify all combos are valid pairs
			for _, combo := range combos {
				if combo.Card1.Rank != combo.Card2.Rank {
					t.Errorf("Expected pair but got %v", combo)
				}
				if combo.Card1.Suit == combo.Card2.Suit {
					t.Errorf("Expected different suits but got %v", combo)
				}
			}
		})
	}
}

func TestParseRange_PairRanges(t *testing.T) {
	tests := []struct {
		input     string
		wantCount int
	}{
		{"KK-JJ", 18}, // KK(6) + QQ(6) + JJ(6)
		{"AA-KK", 12}, // AA(6) + KK(6)
		{"QQ-99", 24}, // QQ(6) + JJ(6) + TT(6) + 99(6)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			combos, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange(%q) error = %v", tt.input, err)
			}
			if len(combos) != tt.wantCount {
				t.Errorf("ParseRange(%q) returned %d combos, want %d", tt.input, len(combos), tt.wantCount)
			}
		})
	}
}

func TestParseRange_SuitedHands(t *testing.T) {
	tests := []struct {
		input     string
		wantCount int
	}{
		{"AKs", 4}, // As-Ks, Ah-Kh, Ad-Kd, Ac-Kc
		{"AQs", 4},
		{"T9s", 4},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			combos, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange(%q) error = %v", tt.input, err)
			}
			if len(combos) != tt.wantCount {
				t.Errorf("ParseRange(%q) returned %d combos, want %d", tt.input, len(combos), tt.wantCount)
			}
			// Verify all combos are suited
			for _, combo := range combos {
				if combo.Card1.Suit != combo.Card2.Suit {
					t.Errorf("Expected suited but got %v", combo)
				}
			}
		})
	}
}

func TestParseRange_SuitedRanges(t *testing.T) {
	tests := []struct {
		input     string
		wantCount int
	}{
		{"AKs-ATs", 16}, // AKs(4) + AQs(4) + AJs(4) + ATs(4)
		{"KQs-KJs", 8},  // KQs(4) + KJs(4)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			combos, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange(%q) error = %v", tt.input, err)
			}
			if len(combos) != tt.wantCount {
				t.Errorf("ParseRange(%q) returned %d combos, want %d", tt.input, len(combos), tt.wantCount)
			}
			// Verify all combos are suited
			for _, combo := range combos {
				if combo.Card1.Suit != combo.Card2.Suit {
					t.Errorf("Expected suited but got %v", combo)
				}
			}
		})
	}
}

func TestParseRange_OffsuitHands(t *testing.T) {
	tests := []struct {
		input     string
		wantCount int
	}{
		{"AKo", 12}, // All offsuit combinations
		{"AQo", 12},
		{"KQo", 12},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			combos, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange(%q) error = %v", tt.input, err)
			}
			if len(combos) != tt.wantCount {
				t.Errorf("ParseRange(%q) returned %d combos, want %d", tt.input, len(combos), tt.wantCount)
			}
			// Verify all combos are offsuit
			for _, combo := range combos {
				if combo.Card1.Suit == combo.Card2.Suit {
					t.Errorf("Expected offsuit but got %v", combo)
				}
			}
		})
	}
}

func TestParseRange_OffsuitRanges(t *testing.T) {
	tests := []struct {
		input     string
		wantCount int
	}{
		{"AQo-AJo", 24}, // AQo(12) + AJo(12)
		{"KQo-KJo", 24}, // KQo(12) + KJo(12)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			combos, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange(%q) error = %v", tt.input, err)
			}
			if len(combos) != tt.wantCount {
				t.Errorf("ParseRange(%q) returned %d combos, want %d", tt.input, len(combos), tt.wantCount)
			}
			// Verify all combos are offsuit
			for _, combo := range combos {
				if combo.Card1.Suit == combo.Card2.Suit {
					t.Errorf("Expected offsuit but got %v", combo)
				}
			}
		})
	}
}

func TestParseRange_MultipleHands(t *testing.T) {
	tests := []struct {
		input     string
		wantCount int
	}{
		{"AA,KK", 12},         // 6 + 6
		{"AA,KK,AKs", 16},     // 6 + 6 + 4
		{"AA,AKs,AKo", 22},    // 6 + 4 + 12
		{"KK-JJ,AKs,AQo", 34}, // KK-JJ(18) + AKs(4) + AQo(12) = 34
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			combos, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange(%q) error = %v", tt.input, err)
			}
			if len(combos) != tt.wantCount {
				t.Errorf("ParseRange(%q) returned %d combos, want %d", tt.input, len(combos), tt.wantCount)
			}
		})
	}
}

func TestParseRange_SpecificExample(t *testing.T) {
	// This is the example from DESIGN.md
	// BTN: AA,KK,AKs = 6 + 6 + 4 = 16 combos
	// BB: QQ-JJ,AJs-ATs = (6+6) + (4+4) = 20 combos

	btnRange, err := ParseRange("AA,KK,AKs")
	if err != nil {
		t.Fatalf("ParseRange(BTN) error = %v", err)
	}
	if len(btnRange) != 16 {
		t.Errorf("BTN range: got %d combos, want 16", len(btnRange))
	}

	bbRange, err := ParseRange("QQ-JJ,AJs-ATs")
	if err != nil {
		t.Fatalf("ParseRange(BB) error = %v", err)
	}
	if len(bbRange) != 20 {
		t.Errorf("BB range: got %d combos, want 20", len(bbRange))
	}
}

func TestParseRange_Errors(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", true},        // empty
		{"A", true},       // too short
		{"AK", true},      // ambiguous (need s or o)
		{"AKx", true},     // invalid indicator
		{"AA-KKo", true},  // mismatched suited/offsuit
		{"AKs-AQo", true}, // mismatched suited/offsuit
		{"XX", true},      // invalid ranks
		{"AK-KQ", true},   // invalid range (first rank doesn't match)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRange(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestCombo_String(t *testing.T) {
	combo := Combo{
		Card1: cards.NewCard(cards.Ace, cards.Spades),
		Card2: cards.NewCard(cards.King, cards.Hearts),
	}
	want := "AsKh"
	if got := combo.String(); got != want {
		t.Errorf("Combo.String() = %q, want %q", got, want)
	}
}

func TestParseRange_NoDuplicates(t *testing.T) {
	// Ensure we don't generate duplicate combos
	combos, err := ParseRange("AA")
	if err != nil {
		t.Fatalf("ParseRange error = %v", err)
	}

	seen := make(map[string]bool)
	for _, combo := range combos {
		key := combo.String()
		if seen[key] {
			t.Errorf("Duplicate combo: %s", key)
		}
		seen[key] = true
	}
}

func TestParseRange_PairComboOrder(t *testing.T) {
	// For pairs, card1 should always be higher suit (by index)
	combos, err := ParseRange("AA")
	if err != nil {
		t.Fatalf("ParseRange error = %v", err)
	}

	// Should be: AsAh, AsAd, AsAc, AhAd, AhAc, AdAc
	expected := []string{"AsAh", "AsAd", "AsAc", "AhAd", "AhAc", "AdAc"}
	if len(combos) != len(expected) {
		t.Fatalf("Expected %d combos, got %d", len(expected), len(combos))
	}

	for i, combo := range combos {
		got := combo.String()
		if got != expected[i] {
			t.Errorf("Combo %d: got %s, want %s", i, got, expected[i])
		}
	}
}
