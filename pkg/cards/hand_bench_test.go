package cards

import (
	"testing"
)

// BenchmarkEvaluate benchmarks the critical Evaluate function
func BenchmarkEvaluate(b *testing.B) {
	// Parse test hands once
	hands := []struct {
		name  string
		cards []Card
	}{
		{
			"Royal flush",
			mustParseCards("AhKhQhJhTh2d3c"),
		},
		{
			"Quad aces",
			mustParseCards("AsAhAdAcKs2d3c"),
		},
		{
			"Full house",
			mustParseCards("AsAhAdKsKh2d3c"),
		},
		{
			"Flush",
			mustParseCards("AhKh9h5h2h3dQc"),
		},
		{
			"Straight",
			mustParseCards("AhKdQcJsTs2h3c"),
		},
		{
			"Two pair",
			mustParseCards("AsAhKdKsQh2d3c"),
		},
		{
			"One pair",
			mustParseCards("AsAhKdQsJh9d7c"),
		},
		{
			"High card",
			mustParseCards("AhKd9s7c5h3d2s"),
		},
	}

	b.Run("AllHandTypes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, hand := range hands {
				_ = Evaluate(hand.cards)
			}
		}
	})

	// Benchmark individual hand types
	for _, hand := range hands {
		b.Run(hand.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = Evaluate(hand.cards)
			}
		})
	}
}

// BenchmarkCompare benchmarks hand comparison
func BenchmarkCompare(b *testing.B) {
	cards1 := mustParseCards("9s8s7s6s5s2h3d")
	cards2 := mustParseCards("AsAhAdAcKs2d3c")

	val1 := Evaluate(cards1)
	val2 := Evaluate(cards2)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = val1.Compare(val2)
	}
}

// BenchmarkParseCard benchmarks card parsing
func BenchmarkParseCard(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseCard("As")
	}
}

// BenchmarkParseCards benchmarks parsing multiple cards
func BenchmarkParseCards(b *testing.B) {
	input := "AhKhQhJhTh2d3c"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ParseCards(input)
	}
}

// Helper function for benchmarks
func mustParseCards(s string) []Card {
	cards, err := ParseCards(s)
	if err != nil {
		panic(err)
	}
	return cards
}
