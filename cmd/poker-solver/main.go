package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/behrlich/poker-solver/pkg/abstraction"
	"github.com/behrlich/poker-solver/pkg/notation"
	"github.com/behrlich/poker-solver/pkg/solver"
	"github.com/behrlich/poker-solver/pkg/tree"
)

func main() {
	// Define flags
	iterations := flag.Int("iterations", 10000, "Number of CFR iterations to run")
	verbose := flag.Bool("verbose", false, "Show detailed output")
	saveFile := flag.String("save", "", "Save strategy profile to JSON file")
	loadFile := flag.String("load", "", "Load strategy profile from JSON file (skips solving)")

	// Geometric bet sizing flags
	useGeometric := flag.Bool("geometric", false, "Use geometric bet sizing")
	targetPot := flag.Float64("target-pot", 30.0, "Target pot size in BB for geometric sizing")
	numSizes := flag.Int("num-sizes", 1, "Number of geometric bet sizes to generate (1-3)")

	// Card abstraction flags
	numBuckets := flag.Int("buckets", 0, "Number of buckets for card abstraction (0 = disabled)")

	flag.Parse()

	// Handle load mode
	if *loadFile != "" {
		profile, err := solver.LoadFromFile(*loadFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading strategy: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded strategy profile with %d information sets\n\n", profile.NumInfoSets())

		// For display, we need to parse the position string if provided
		args := flag.Args()
		if len(args) >= 1 {
			gs, err := notation.ParsePosition(args[0])
			if err == nil {
				isRangeVsRange := len(gs.Players[0].Range) > 1 || len(gs.Players[1].Range) > 1
				printStrategies(profile, gs, isRangeVsRange, *verbose)
			} else {
				// No position or invalid position - just show all strategies
				printAllStrategies(profile, *verbose)
			}
		} else {
			printAllStrategies(profile, *verbose)
		}
		return
	}

	// Get position string from arguments
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: poker-solver [flags] <position>\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # River (uses vanilla CFR)\n")
		fmt.Fprintf(os.Stderr, "  poker-solver \"BTN:AsKd:S100/BB:QhQd:S100|P10|Kh9s4c7d2s|>BTN\"\n\n")
		fmt.Fprintf(os.Stderr, "  # Turn (uses MCCFR with rollout)\n")
		fmt.Fprintf(os.Stderr, "  poker-solver \"BTN:AA:S100/BB:QQ:S100|P10|Kh9s4c7d|>BTN\"\n\n")
		fmt.Fprintf(os.Stderr, "  # Flop with geometric sizing and bucketing\n")
		fmt.Fprintf(os.Stderr, "  poker-solver --geometric --target-pot=30 --buckets=100 \\\n")
		fmt.Fprintf(os.Stderr, "    \"BTN:AA,KK:S97.5/BB:QQ,JJ:S97.5|P5.5|Th9h2c|>BTN\"\n\n")
		fmt.Fprintf(os.Stderr, "  # Save/load strategies\n")
		fmt.Fprintf(os.Stderr, "  poker-solver --save=strategy.json \"BTN:AA:S100/BB:QQ:S100|P10|Kh9s4c7d2s|>BTN\"\n")
		fmt.Fprintf(os.Stderr, "  poker-solver --load=strategy.json\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	positionStr := args[0]

	// Parse position
	if *verbose {
		fmt.Printf("Parsing position: %s\n", positionStr)
	}

	gs, err := notation.ParsePosition(positionStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing position: %v\n", err)
		os.Exit(1)
	}

	// Verify we have specific cards (not ranges) for both players
	if len(gs.Players) != 2 {
		fmt.Fprintf(os.Stderr, "Error: Only 2-player games supported\n")
		os.Exit(1)
	}

	// Determine if we have specific cards or ranges
	isRangeVsRange := len(gs.Players[0].Range) > 1 || len(gs.Players[1].Range) > 1

	if *verbose {
		fmt.Printf("\nGame State:\n")
		if isRangeVsRange {
			fmt.Printf("  %s: %d combos (%.1fbb)\n", gs.Players[0].Position, len(gs.Players[0].Range), gs.Players[0].Stack)
			fmt.Printf("  %s: %d combos (%.1fbb)\n", gs.Players[1].Position, len(gs.Players[1].Range), gs.Players[1].Stack)
		} else {
			fmt.Printf("  %s: %s (%.1fbb)\n", gs.Players[0].Position, gs.Players[0].Range[0].String(), gs.Players[0].Stack)
			fmt.Printf("  %s: %s (%.1fbb)\n", gs.Players[1].Position, gs.Players[1].Range[0].String(), gs.Players[1].Stack)
		}
		fmt.Printf("  Pot: %.1fbb\n", gs.Pot)
		fmt.Printf("  Board: ")
		for _, card := range gs.Board {
			fmt.Printf("%s", card.String())
		}
		fmt.Printf(" (%s)\n", gs.Street.String())
		fmt.Printf("  To act: %s\n\n", gs.Players[gs.ToAct].Position)
	}

	// Determine street and configuration
	numBoardCards := len(gs.Board)
	isRiver := numBoardCards == 5
	isTurn := numBoardCards == 4
	isFlop := numBoardCards == 3

	// Build action config
	var config tree.ActionConfig
	if *useGeometric {
		// Use geometric bet sizing
		effectiveStack := gs.Players[0].Stack
		if gs.Players[1].Stack < effectiveStack {
			effectiveStack = gs.Players[1].Stack
		}

		// Calculate number of streets remaining
		numStreets := 0
		if isFlop {
			numStreets = 3 // flop, turn, river
		} else if isTurn {
			numStreets = 2 // turn, river
		} else if isRiver {
			numStreets = 1 // river only
		} else {
			fmt.Fprintf(os.Stderr, "Error: Unsupported board size %d (must be 3, 4, or 5 cards)\n", numBoardCards)
			os.Exit(1)
		}

		geoSizing := tree.NewGeometricSizing(*targetPot, numStreets, effectiveStack)
		config = tree.ActionConfig{
			GeometricSizing:   geoSizing,
			NumGeometricSizes: *numSizes,
			AllowCheck:        true,
			AllowCall:         true,
			AllowFold:         true,
		}

		if *verbose {
			fmt.Printf("Using geometric sizing: target %.1fbb pot, %d streets, %d sizes\n",
				*targetPot, numStreets, *numSizes)
		}
	} else {
		// Use default pot-relative bet sizes
		config = tree.DefaultRiverConfig()
	}

	// Build game tree
	if *verbose {
		if isRangeVsRange {
			fmt.Printf("Building range-vs-range tree (%d × %d combos)...\n", len(gs.Players[0].Range), len(gs.Players[1].Range))
		} else {
			fmt.Printf("Building game tree...\n")
		}
	}

	builder := tree.NewBuilder(config)

	// Add bucketing if requested
	if *numBuckets > 0 {
		// Create bucketer from acting player's perspective
		// Note: This creates a single bucketer with opponent's range.
		// For full range-vs-range, ideally we'd have two bucketers (one per player).
		// This simplified approach works well for small ranges and combo-vs-range scenarios.
		oppIdx := 1 - gs.ToAct
		oppRange := gs.Players[oppIdx].Range

		bucketer := abstraction.NewBucketer(gs.Board, oppRange, *numBuckets)
		builder.SetBucketer(bucketer)

		if *verbose {
			fmt.Printf("Using card abstraction: %d buckets, opponent range = %d combos\n",
				*numBuckets, len(oppRange))
		}
	}

	var root *tree.TreeNode
	if isRangeVsRange {
		root, err = builder.BuildRange(gs, gs.Players[0].Range, gs.Players[1].Range)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building range tree: %v\n", err)
			os.Exit(1)
		}
	} else {
		combo0 := gs.Players[0].Range[0]
		combo1 := gs.Players[1].Range[0]
		root, err = builder.Build(gs, combo0, combo1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building tree: %v\n", err)
			os.Exit(1)
		}
	}

	if *verbose {
		fmt.Printf("Tree built successfully\n\n")
	}

	// Determine which solver to use based on street
	// MCCFR: Required for turn/flop (needs rollout for future cards)
	// Vanilla CFR: Efficient for river (no future cards to sample)
	var profile *solver.StrategyProfile

	if isFlop || isTurn {
		// Use MCCFR for multi-street positions (flop or turn)
		streetName := "turn"
		if isFlop {
			streetName = "flop"
		}
		fmt.Printf("Solving %s position with MCCFR (%d iterations)...\n", streetName, *iterations)
		mccfr := solver.NewMCCFR(42) // Fixed seed for reproducibility
		profile = mccfr.Train(root, *iterations)
	} else if isRiver {
		// Use vanilla CFR for river positions (more efficient, no rollout needed)
		fmt.Printf("Solving river position with CFR (%d iterations)...\n", *iterations)
		cfr := solver.NewCFR()
		profile = cfr.Train(root, *iterations)
	} else {
		fmt.Fprintf(os.Stderr, "Error: Unsupported street (board has %d cards)\n", numBoardCards)
		os.Exit(1)
	}

	fmt.Printf("Solved! Found %d information sets\n\n", profile.NumInfoSets())

	// Save strategy if requested
	if *saveFile != "" {
		err := profile.SaveToFile(*saveFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving strategy: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Strategy saved to %s\n\n", *saveFile)
	}

	// Output strategies
	printStrategies(profile, gs, isRangeVsRange, *verbose)
}

func printStrategies(profile *solver.StrategyProfile, gs *notation.GameState, isRangeVsRange bool, verbose bool) {
	if isRangeVsRange {
		printRangeStrategies(profile, gs, verbose)
	} else {
		printComboStrategies(profile, gs, verbose)
	}
}

// printComboStrategies prints strategies for specific combo-vs-combo scenarios
func printComboStrategies(profile *solver.StrategyProfile, gs *notation.GameState, verbose bool) {
	fmt.Printf("=== STRATEGIES ===\n\n")

	// Get all infosets and sort them for consistent output
	allStrats := profile.All()
	infoSets := make([]string, 0, len(allStrats))
	for infoSet := range allStrats {
		infoSets = append(infoSets, infoSet)
	}
	sort.Strings(infoSets)

	// Print each strategy
	for _, infoSet := range infoSets {
		strat := allStrats[infoSet]
		avgStrat := strat.GetAverageStrategy()

		// Parse infoset to show player position
		// InfoSet format: "board|history|>player|cards"
		fmt.Printf("InfoSet: %s\n", infoSet)

		// Print action probabilities
		for i, action := range strat.Actions {
			prob := avgStrat[i]
			if prob > 0.001 { // Only show actions with >0.1% probability
				fmt.Printf("  %s: %.1f%%\n", action.String(), prob*100)
			}
		}

		if verbose {
			// Show regrets in verbose mode
			fmt.Printf("  Regrets: ")
			for i, regret := range strat.RegretSum {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%.2f", regret)
			}
			fmt.Printf("\n")
		}

		fmt.Printf("\n")
	}
}

// printRangeStrategies prints aggregated strategies for range-vs-range scenarios
func printRangeStrategies(profile *solver.StrategyProfile, gs *notation.GameState, verbose bool) {
	fmt.Printf("=== RANGE-VS-RANGE STRATEGIES ===\n\n")

	// Aggregate strategies by hand type and game situation
	// Map: "position|history|handtype" -> aggregated strategy
	aggregated := make(map[string]*AggregatedStrategy)

	allStrats := profile.All()
	for infoSet, strat := range allStrats {
		// Parse infoset: "board|history|>player|cards"
		parts := parseInfoSet(infoSet)
		if parts == nil {
			continue
		}

		// Extract hand type from specific cards (e.g., "AsAh" -> "AA")
		handType := getHandType(parts.cards)

		// Create aggregation key
		aggKey := fmt.Sprintf("%s|%s|%s", parts.player, parts.history, handType)

		if _, exists := aggregated[aggKey]; !exists {
			aggregated[aggKey] = &AggregatedStrategy{
				Player:   parts.player,
				History:  parts.history,
				HandType: handType,
				Actions:  strat.Actions,
				Probs:    make([]float64, len(strat.Actions)),
				Count:    0,
			}
		}

		// Add this combo's strategy to the aggregate
		avgStrat := strat.GetAverageStrategy()
		for i := range avgStrat {
			aggregated[aggKey].Probs[i] += avgStrat[i]
		}
		aggregated[aggKey].Count++
	}

	// Average the probabilities
	for _, agg := range aggregated {
		for i := range agg.Probs {
			agg.Probs[i] /= float64(agg.Count)
		}
	}

	// Group by player and sort
	playerStrats := make(map[string][]*AggregatedStrategy)
	for _, agg := range aggregated {
		playerStrats[agg.Player] = append(playerStrats[agg.Player], agg)
	}

	// Print strategies grouped by player
	players := []string{}
	for player := range playerStrats {
		players = append(players, player)
	}
	sort.Strings(players)

	for _, player := range players {
		fmt.Printf("%s:\n", player)

		// Sort strategies by history length (simpler situations first)
		strats := playerStrats[player]
		sort.Slice(strats, func(i, j int) bool {
			if len(strats[i].History) != len(strats[j].History) {
				return len(strats[i].History) < len(strats[j].History)
			}
			if strats[i].HandType != strats[j].HandType {
				return strats[i].HandType < strats[j].HandType
			}
			return strats[i].History < strats[j].History
		})

		for _, agg := range strats {
			// Print situation
			situation := "acts first"
			if len(agg.History) > 0 {
				situation = fmt.Sprintf("facing %s", agg.History)
			}

			fmt.Printf("  %s (%s):\n", agg.HandType, situation)

			// Print action probabilities
			for i, action := range agg.Actions {
				prob := agg.Probs[i]
				if prob > 0.01 { // Show actions with >1% probability
					fmt.Printf("    %s: %.1f%%\n", action.String(), prob*100)
				}
			}

			if verbose {
				fmt.Printf("    (averaged over %d combos)\n", agg.Count)
			}
		}
		fmt.Printf("\n")
	}
}

// InfoSetParts holds parsed components of an information set key
type InfoSetParts struct {
	board   string
	history string
	player  string
	cards   string
}

// parseInfoSet parses an information set key into its components
// Format: "board|history|>player|cards"
func parseInfoSet(infoSet string) *InfoSetParts {
	// Split by |
	parts := make([]string, 0, 4)
	lastIdx := 0
	for i := 0; i < len(infoSet); i++ {
		if infoSet[i] == '|' {
			parts = append(parts, infoSet[lastIdx:i])
			lastIdx = i + 1
		}
	}
	parts = append(parts, infoSet[lastIdx:])

	if len(parts) != 4 {
		return nil
	}

	player := parts[2]
	if len(player) > 0 && player[0] == '>' {
		player = player[1:]
	}

	return &InfoSetParts{
		board:   parts[0],
		history: parts[1],
		player:  player,
		cards:   parts[3],
	}
}

// getHandType extracts hand type from specific cards
// e.g., "AsAh" -> "AA", "KsKd" -> "KK"
// For bucketed hands, returns the bucket ID as-is
func getHandType(cards string) string {
	// Handle bucketed hands - return bucket ID as-is
	if strings.HasPrefix(cards, "BUCKET_") {
		return cards
	}

	if len(cards) < 4 {
		return cards
	}

	// Extract ranks (first and third characters)
	rank1 := cards[0]
	rank2 := cards[2]

	if rank1 == rank2 {
		// Pair
		return string([]byte{rank1, rank2})
	}

	// Non-pair - return in canonical order (higher rank first)
	suit1 := cards[1]
	suit2 := cards[3]

	suited := "o"
	if suit1 == suit2 {
		suited = "s"
	}

	// Determine canonical order
	ranks := "AKQJT98765432"
	idx1 := -1
	idx2 := -1
	for i, r := range ranks {
		if byte(r) == rank1 {
			idx1 = i
		}
		if byte(r) == rank2 {
			idx2 = i
		}
	}

	if idx1 < idx2 {
		return string([]byte{rank1, rank2}) + suited
	}
	return string([]byte{rank2, rank1}) + suited
}

// AggregatedStrategy holds averaged strategy for a hand type in a situation
type AggregatedStrategy struct {
	Player   string
	History  string
	HandType string
	Actions  []notation.Action
	Probs    []float64
	Count    int
}

// printAllStrategies prints all strategies in the profile (for load mode without position)
func printAllStrategies(profile *solver.StrategyProfile, verbose bool) {
	fmt.Printf("=== ALL STRATEGIES ===\n\n")

	allStrats := profile.All()
	infoSets := make([]string, 0, len(allStrats))
	for infoSet := range allStrats {
		infoSets = append(infoSets, infoSet)
	}
	sort.Strings(infoSets)

	for _, infoSet := range infoSets {
		strat := allStrats[infoSet]
		avgStrat := strat.GetAverageStrategy()

		fmt.Printf("InfoSet: %s\n", infoSet)
		for i, action := range strat.Actions {
			prob := avgStrat[i]
			if prob > 0.001 {
				fmt.Printf("  %s: %.1f%%\n", action.String(), prob*100)
			}
		}

		if verbose {
			fmt.Printf("  Regrets: ")
			for i, regret := range strat.RegretSum {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%.2f", regret)
			}
			fmt.Printf("\n")
		}

		fmt.Printf("\n")
	}
}
