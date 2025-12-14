package abstraction

import (
	"testing"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/notation"
)

func TestNewBucketer(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	if bucketer == nil {
		t.Fatal("Expected bucketer to be created")
	}

	// For 100 buckets, we expect 10x10 grid
	if bucketer.equityBins != 10 {
		t.Errorf("Expected 10 equity bins for 100 buckets, got %d", bucketer.equityBins)
	}

	if bucketer.potentialBins != 10 {
		t.Errorf("Expected 10 potential bins for 100 buckets, got %d", bucketer.potentialBins)
	}

	actualBuckets := bucketer.NumBuckets()
	if actualBuckets != 100 {
		t.Errorf("Expected 100 buckets, got %d", actualBuckets)
	}
}

func TestBucketHand_Deterministic(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	hero, _ := cards.ParseCards("KdKc")

	// Bucket same hand multiple times
	bucket1 := bucketer.BucketHand(hero)
	bucket2 := bucketer.BucketHand(hero)
	bucket3 := bucketer.BucketHand(hero)

	if bucket1 != bucket2 || bucket2 != bucket3 {
		t.Errorf("Bucketing is not deterministic: %d, %d, %d", bucket1, bucket2, bucket3)
	}
}

func TestBucketHand_SimilarHands(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	// Use a wider opponent range (QQ, JJ) so AA and KK have similar equity
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Queen, Suit: cards.Diamonds}, Card2: cards.Card{Rank: cards.Queen, Suit: cards.Hearts}},
		{Card1: cards.Card{Rank: cards.Jack, Suit: cards.Diamonds}, Card2: cards.Card{Rank: cards.Jack, Suit: cards.Hearts}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	// AA and KK should be in same or adjacent buckets (both strong overpairs vs QQ,JJ)
	aa, _ := cards.ParseCards("AdAc")
	kk, _ := cards.ParseCards("KdKc")

	bucketAA := bucketer.BucketHand(aa)
	bucketKK := bucketer.BucketHand(kk)

	// Allow up to 11 bucket difference (adjacent in 2D grid including diagonal)
	diff := bucketAA - bucketKK
	if diff < 0 {
		diff = -diff
	}

	if diff > 11 { // Allow diagonal adjacency in 10x10 grid
		t.Errorf("AA (bucket %d) and KK (bucket %d) are too far apart (diff %d)", bucketAA, bucketKK, diff)
	}

	t.Logf("AA: bucket %d, KK: bucket %d (diff %d) vs QQ,JJ", bucketAA, bucketKK, diff)
}

func TestBucketHand_DifferentHandTypes(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	// Strong made hand vs weak air
	aa, _ := cards.ParseCards("AdAc")      // Strong overpair
	sevenTwo, _ := cards.ParseCards("7d2c") // Complete air

	bucketAA := bucketer.BucketHand(aa)
	bucketAir := bucketer.BucketHand(sevenTwo)

	// These should be in very different buckets
	diff := bucketAA - bucketAir
	if diff < 0 {
		diff = -diff
	}

	if diff < 20 {
		t.Errorf("AA (bucket %d) and 72o (bucket %d) should be far apart, but diff is only %d", bucketAA, bucketAir, diff)
	}

	t.Logf("AA: bucket %d, 72o: bucket %d (diff %d)", bucketAA, bucketAir, diff)
}

func TestBucketHand_DrawVsMadeHand(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	// Flush draw vs overpair
	flushDraw, _ := cards.ParseCards("AhKh") // Flush draw with overcards
	overpair, _ := cards.ParseCards("QdQc")  // Overpair

	bucketDraw := bucketer.BucketHand(flushDraw)
	bucketPair := bucketer.BucketHand(overpair)

	// These have different equity/potential profiles
	// Flush draw: lower equity, higher potential
	// Overpair: higher equity, lower potential
	// Should be in different buckets
	if bucketDraw == bucketPair {
		t.Errorf("Flush draw and overpair should be in different buckets")
	}

	t.Logf("Flush draw: bucket %d, Overpair: bucket %d", bucketDraw, bucketPair)
	t.Logf("  %s", bucketer.GetBucketInfo(bucketDraw))
	t.Logf("  %s", bucketer.GetBucketInfo(bucketPair))
}

func TestBucketCombo(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	combo := notation.Combo{
		Card1: cards.Card{Rank: cards.King, Suit: cards.Diamonds},
		Card2: cards.Card{Rank: cards.King, Suit: cards.Clubs},
	}

	bucket := bucketer.BucketCombo(combo)

	if bucket < 0 || bucket >= 100 {
		t.Errorf("Expected bucket in range [0, 100), got %d", bucket)
	}

	t.Logf("KK combo bucketed to %d", bucket)
}

func TestGetBucketInfo(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	// Test a few bucket IDs
	info0 := bucketer.GetBucketInfo(0)
	info50 := bucketer.GetBucketInfo(50)
	info99 := bucketer.GetBucketInfo(99)

	t.Logf("Bucket 0: %s", info0)
	t.Logf("Bucket 50: %s", info50)
	t.Logf("Bucket 99: %s", info99)

	// Basic validation - info strings should be non-empty
	if info0 == "" || info50 == "" || info99 == "" {
		t.Error("Bucket info should not be empty")
	}
}

func TestBucketHand_Cache(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	hero, _ := cards.ParseCards("KdKc")

	// First call - not cached
	bucket1 := bucketer.BucketHand(hero)

	// Second call - should hit cache
	bucket2 := bucketer.BucketHand(hero)

	if bucket1 != bucket2 {
		t.Errorf("Cached result differs: %d vs %d", bucket1, bucket2)
	}

	// Clear cache and verify it still works
	bucketer.ClearCache()
	bucket3 := bucketer.BucketHand(hero)

	if bucket1 != bucket3 {
		t.Errorf("After cache clear, result differs: %d vs %d", bucket1, bucket3)
	}
}

func TestBucketHand_BucketDistribution(t *testing.T) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)

	// Sample a variety of hands and check bucket distribution
	testHands := []string{
		"AdAc", "KdKc", "QdQc", "JdJc", "TdTc", "9d9c", "8d8c", "7d7c", "6d6c", "5d5c",
		"AhKh", "AhQh", "AhJh", "KhQh", "QhJh", // Suited connectors/aces
		"Ah2h", "Kh3h", "Qh4h",                  // Suited but weaker
		"7d2c", "8d3c", "9d4c",                  // Air
	}

	bucketCounts := make(map[int]int)

	for _, handStr := range testHands {
		hand, _ := cards.ParseCards(handStr)
		bucket := bucketer.BucketHand(hand)
		bucketCounts[bucket]++

		t.Logf("%s -> bucket %d (%s)", handStr, bucket, bucketer.GetBucketInfo(bucket))
	}

	// Should use multiple buckets
	uniqueBuckets := len(bucketCounts)
	if uniqueBuckets < 5 {
		t.Errorf("Expected at least 5 different buckets, got %d", uniqueBuckets)
	}

	t.Logf("Sampled %d hands, used %d unique buckets", len(testHands), uniqueBuckets)
}

// Benchmark bucketing performance
func BenchmarkBucketHand(b *testing.B) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)
	hero, _ := cards.ParseCards("KdKc")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bucketer.BucketHand(hero)
	}
}

// Benchmark bucketing without cache
func BenchmarkBucketHand_NoCache(b *testing.B) {
	board, _ := cards.ParseCards("Th9h2c")
	oppRange := []notation.Combo{
		{Card1: cards.Card{Rank: cards.Ace, Suit: cards.Spades}, Card2: cards.Card{Rank: cards.Ace, Suit: cards.Diamonds}},
	}

	bucketer := NewBucketer(board, oppRange, 100)
	hero, _ := cards.ParseCards("KdKc")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bucketer.ClearCache()
		bucketer.BucketHand(hero)
	}
}
