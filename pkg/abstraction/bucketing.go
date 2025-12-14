package abstraction

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/behrlich/poker-solver/pkg/cards"
	"github.com/behrlich/poker-solver/pkg/equity"
	"github.com/behrlich/poker-solver/pkg/notation"
)

// Bucketer assigns hands to buckets based on equity and potential
type Bucketer struct {
	board         []cards.Card
	opponentRange []notation.Combo
	numBuckets    int
	calculator    *equity.Calculator

	// Grid dimensions for 2D histogram
	equityBins    int // Number of bins along equity axis
	potentialBins int // Number of bins along potential axis

	// Cache for performance
	cache       map[string]int
	oppHash     string
	useSampling bool
	samples     int
	eqCache     map[string]eqPot
}

type eqPot struct {
	equity    float64
	potential float64
}

// NewBucketer creates a bucketer for a given board and opponent range
// numBuckets: total number of buckets (typically 50-200)
func NewBucketer(board []cards.Card, opponentRange []notation.Combo, numBuckets int) *Bucketer {
	// For 2D histogram, we want roughly equal bins in each dimension
	// If numBuckets = 100, use 10x10 grid
	// If numBuckets = 200, use 14x14 grid (196 actual buckets)
	gridSize := int(math.Sqrt(float64(numBuckets)))

	return &Bucketer{
		board:         board,
		opponentRange: opponentRange,
		numBuckets:    numBuckets,
		calculator:    equity.NewCalculator(),
		equityBins:    gridSize,
		potentialBins: gridSize,
		cache:         make(map[string]int),
		oppHash:       hashRange(opponentRange),
		useSampling:   false,
		samples:       0,
		eqCache:       make(map[string]eqPot),
	}
}

// NewBucketerSampled creates a bucketer that uses Monte Carlo sampling for equity/potential.
// Sampling is deterministic (seeded from board, hero, and opponent range hash) to keep buckets stable.
func NewBucketerSampled(board []cards.Card, opponentRange []notation.Combo, numBuckets int, samples int) *Bucketer {
	if samples <= 0 {
		samples = 200 // reasonable default for WASM/mobile
	}

	b := NewBucketer(board, opponentRange, numBuckets)
	b.useSampling = true
	b.samples = samples
	return b
}

// BucketHand assigns a hand to a bucket ID (0 to numBuckets-1)
func (b *Bucketer) BucketHand(hero []cards.Card) int {
	// Check cache
	cacheKey := fmt.Sprintf("%s%s", hero[0].String(), hero[1].String())
	if bucket, exists := b.cache[cacheKey]; exists {
		return bucket
	}

	// Calculate equity and potential
	var equityVal float64
	var potentialVal float64

	if b.useSampling {
		eq, pot := b.sampleEquityPotential(hero)
		equityVal = eq
		potentialVal = pot
	} else {
		equityResult := b.calculator.CalculateEquity(hero, b.board, b.opponentRange)
		potentialResult := b.calculator.CalculatePotential(hero, b.board, b.opponentRange)
		equityVal = equityResult.Equity
		potentialVal = potentialResult.ImprovePct
	}

	// Use equity + potential for bucketing
	// On flop: both matter (draws have high potential, made hands have high equity)
	// On turn: mostly equity (less potential remaining)
	// On river: only equity (no potential)

	// Assign to 2D histogram bin
	equityBin := int(equityVal * float64(b.equityBins))
	potentialBin := int(potentialVal * float64(b.potentialBins))

	// Clamp to valid range
	if equityBin >= b.equityBins {
		equityBin = b.equityBins - 1
	}
	if potentialBin >= b.potentialBins {
		potentialBin = b.potentialBins - 1
	}

	// Convert 2D bin to 1D bucket ID
	bucketID := equityBin*b.potentialBins + potentialBin

	// Cache result
	b.cache[cacheKey] = bucketID

	return bucketID
}

// BucketCombo is a convenience wrapper for notation.Combo
func (b *Bucketer) BucketCombo(combo notation.Combo) int {
	hero := []cards.Card{combo.Card1, combo.Card2}
	return b.BucketHand(hero)
}

// sampleEquityPotential computes equity and potential using Monte Carlo sampling with deterministic seeding.
func (b *Bucketer) sampleEquityPotential(hero []cards.Card) (float64, float64) {
	cacheKey := hero[0].String() + hero[1].String()
	if val, ok := b.eqCache[cacheKey]; ok {
		return val.equity, val.potential
	}

	seed := deterministicSeed(hero, b.board, b.oppHash)
	rng := rand.New(rand.NewSource(seed))

	// Build deck of remaining cards
	remaining := remainingDeck(hero, b.board)
	if len(remaining) < 2 {
		// Nothing to sample, fall back to deterministic evaluation
		e := b.calculator.CalculateEquity(hero, b.board, b.opponentRange)
		p := b.calculator.CalculatePotential(hero, b.board, b.opponentRange)
		b.eqCache[cacheKey] = eqPot{equity: e.Equity, potential: p.ImprovePct}
		return e.Equity, p.ImprovePct
	}

	samples := b.samples
	if samples <= 0 {
		samples = 200
	}

	var eqSamples []float64

	for s := 0; s < samples; s++ {
		var boardRunout []cards.Card
		switch len(b.board) {
		case 5:
			boardRunout = b.board
		case 4:
			// sample river
			river := remaining[rng.Intn(len(remaining))]
			boardRunout = append([]cards.Card{}, b.board...)
			boardRunout = append(boardRunout, river)
		case 3:
			if len(remaining) < 2 {
				continue
			}
			turnIdx := rng.Intn(len(remaining))
			turn := remaining[turnIdx]

			// pick river from remaining without turn
			riverPool := append([]cards.Card{}, remaining[:turnIdx]...)
			riverPool = append(riverPool, remaining[turnIdx+1:]...)
			if len(riverPool) == 0 {
				continue
			}
			river := riverPool[rng.Intn(len(riverPool))]

			boardRunout = append([]cards.Card{}, b.board...)
			boardRunout = append(boardRunout, turn, river)
		default:
			// unsupported board size
			continue
		}

		heroHand := cards.Evaluate(append(hero, boardRunout...))

		wins := 0.0
		ties := 0.0
		total := 0.0

		for _, oppCombo := range b.opponentRange {
			// skip conflicts with runout
			conflict := false
			for _, card := range []cards.Card{oppCombo.Card1, oppCombo.Card2} {
				if card == hero[0] || card == hero[1] {
					conflict = true
					break
				}
				for _, bcard := range boardRunout {
					if card == bcard {
						conflict = true
						break
					}
				}
				if conflict {
					break
				}
			}
			if conflict {
				continue
			}

			oppHand := cards.Evaluate([]cards.Card{oppCombo.Card1, oppCombo.Card2, boardRunout[0], boardRunout[1], boardRunout[2], boardRunout[3], boardRunout[4]})
			cmp := heroHand.Compare(oppHand)
			if cmp > 0 {
				wins++
			} else if cmp == 0 {
				ties++
			}
			total++
		}

		if total == 0 {
			continue
		}

		eq := (wins / total) + (ties / (2.0 * total))
		eqSamples = append(eqSamples, eq)
	}

	if len(eqSamples) == 0 {
		e := b.calculator.CalculateEquity(hero, b.board, b.opponentRange)
		p := b.calculator.CalculatePotential(hero, b.board, b.opponentRange)
		b.eqCache[cacheKey] = eqPot{equity: e.Equity, potential: p.ImprovePct}
		return e.Equity, p.ImprovePct
	}

	// mean
	mean := 0.0
	for _, v := range eqSamples {
		mean += v
	}
	mean /= float64(len(eqSamples))

	// variance
	var variance float64
	for _, v := range eqSamples {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(eqSamples))

	normalizedVar := variance / 0.25
	if normalizedVar > 1.0 {
		normalizedVar = 1.0
	}
	if normalizedVar < 0 {
		normalizedVar = 0
	}

	b.eqCache[cacheKey] = eqPot{equity: mean, potential: normalizedVar}
	return mean, normalizedVar
}

// remainingDeck returns all cards excluding hero and board.
func remainingDeck(hero []cards.Card, board []cards.Card) []cards.Card {
	used := make(map[cards.Card]bool)
	for _, c := range board {
		used[c] = true
	}
	for _, c := range hero {
		used[c] = true
	}

	remaining := make([]cards.Card, 0, 52-len(used))
	for rank := cards.Two; rank <= cards.Ace; rank++ {
		for suit := cards.Spades; suit <= cards.Clubs; suit++ {
			card := cards.Card{Rank: rank, Suit: suit}
			if !used[card] {
				remaining = append(remaining, card)
			}
		}
	}
	return remaining
}

// deterministicSeed builds a repeatable seed from hero, board, and opponent range hash.
func deterministicSeed(hero []cards.Card, board []cards.Card, oppHash string) int64 {
	builder := make([]byte, 0, 64)
	for _, c := range hero {
		builder = append(builder, c.String()...)
	}
	for _, c := range board {
		builder = append(builder, c.String()...)
	}
	builder = append(builder, oppHash...)
	var hash int64
	for _, b := range builder {
		hash = hash*31 + int64(b)
	}
	return hash
}

func hashRange(r []notation.Combo) string {
	parts := make([]string, 0, len(r))
	for _, c := range r {
		parts = append(parts, c.String())
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// GetBucketInfo returns human-readable info about a bucket
func (b *Bucketer) GetBucketInfo(bucketID int) string {
	// Convert bucket ID back to 2D coordinates
	equityBin := bucketID / b.potentialBins
	potentialBin := bucketID % b.potentialBins

	// Calculate ranges
	equityMin := float64(equityBin) / float64(b.equityBins)
	equityMax := float64(equityBin+1) / float64(b.equityBins)
	potentialMin := float64(potentialBin) / float64(b.potentialBins)
	potentialMax := float64(potentialBin+1) / float64(b.potentialBins)

	return fmt.Sprintf("Bucket %d: Equity [%.2f-%.2f], Potential [%.2f-%.2f]",
		bucketID, equityMin, equityMax, potentialMin, potentialMax)
}

// NumBuckets returns the total number of buckets
func (b *Bucketer) NumBuckets() int {
	return b.equityBins * b.potentialBins
}

// ClearCache clears the bucket cache (useful if board or opponent range changes)
func (b *Bucketer) ClearCache() {
	b.cache = make(map[string]int)
	b.eqCache = make(map[string]eqPot)
}
