//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"
	"time"

	"github.com/behrlich/poker-solver/pkg/notation"
	"github.com/behrlich/poker-solver/pkg/solver"
	"github.com/behrlich/poker-solver/pkg/tree"
)

// Global state for cancellation
var cancelSolve chan bool

func main() {
	// Register JavaScript functions
	js.Global().Set("pokerSolver", makePokerSolverAPI())

	// Prevent the Go program from exiting
	select {}
}

// makePokerSolverAPI creates the JavaScript API object
func makePokerSolverAPI() js.Value {
	api := make(map[string]interface{})

	api["solve"] = js.FuncOf(solveWrapper)
	api["parsePosition"] = js.FuncOf(parsePositionWrapper)
	api["cancel"] = js.FuncOf(cancelWrapper)
	api["version"] = "0.3.0"

	return js.ValueOf(api)
}

// solveWrapper wraps the solve function for JavaScript
// Arguments: positionStr (string), iterations (number), progressCallback (function)
// Returns: Promise that resolves to strategy JSON
func solveWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return js.ValueOf(map[string]interface{}{
			"error": "Usage: solve(positionStr, iterations, progressCallback?)",
		})
	}

	positionStr := args[0].String()
	iterations := args[1].Int()

	var progressCallback js.Value
	if len(args) >= 3 && !args[2].IsNull() && !args[2].IsUndefined() {
		progressCallback = args[2]
	}

	// Create a promise
	promiseConstructor := js.Global().Get("Promise")
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		// Run solver in goroutine
		go func() {
			defer func() {
				if r := recover(); r != nil {
					reject.Invoke(js.ValueOf(fmt.Sprintf("Solver panicked: %v", r)))
				}
			}()

			result, err := runSolver(positionStr, iterations, progressCallback)
			if err != nil {
				reject.Invoke(js.ValueOf(err.Error()))
				return
			}

			resolve.Invoke(js.ValueOf(result))
		}()

		return nil
	})

	return promiseConstructor.New(handler)
}

// runSolver performs the actual solving
func runSolver(positionStr string, iterations int, progressCallback js.Value) (map[string]interface{}, error) {
	// Parse position
	gs, err := notation.ParsePosition(positionStr)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Determine range vs range
	isRangeVsRange := len(gs.Players[0].Range) > 1 || len(gs.Players[1].Range) > 1

	// Build tree using lightweight action config for web
	config := tree.ActionConfig{
		BetSizes:   []float64{0.75}, // single mid-size bet for low branching
		AllowCheck: true,
		AllowCall:  true,
		AllowFold:  true,
	}
	builder := tree.NewBuilder(config)

	var root *tree.TreeNode
	if isRangeVsRange {
		root, err = builder.BuildRange(gs, gs.Players[0].Range, gs.Players[1].Range)
	} else {
		combo0 := gs.Players[0].Range[0]
		combo1 := gs.Players[1].Range[0]
		root, err = builder.Build(gs, combo0, combo1)
	}

	if err != nil {
		return nil, fmt.Errorf("tree build error: %w", err)
	}

	// Create cancel channel
	cancelSolve = make(chan bool, 1)

	// Determine solver type
	numBoardCards := len(gs.Board)
	isFlop := numBoardCards == 3
	isTurn := numBoardCards == 4
	var profile *solver.StrategyProfile

	switch {
	case isFlop || isTurn || isRangeVsRange:
		// MCCFR for flop/turn and range river (chance sampling)
		mccfr := solver.NewMCCFR(42)
		profile = trainWithProgress(mccfr, root, iterations, progressCallback, false)
	default:
		// Vanilla CFR only for single-combo river
		cfr := solver.NewCFR()
		profile = trainWithProgress(cfr, root, iterations, progressCallback, true)
	}

	// Convert to JSON
	strategyJSON, err := profile.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("JSON conversion error: %w", err)
	}

	return map[string]interface{}{
		"strategyJSON": string(strategyJSON),
		"infoSets":     profile.NumInfoSets(),
		"position":     positionStr,
	}, nil
}

// trainWithProgress runs CFR/MCCFR with progress callbacks
func trainWithProgress(trainer interface{}, root *tree.TreeNode, iterations int, progressCallback js.Value, isCFR bool) *solver.StrategyProfile {
	// Safety limit
	const maxIterations = 100000
	const maxSolveDuration = 5 * time.Second
	if iterations > maxIterations {
		iterations = maxIterations
	}

	// Report progress every N iterations
	reportInterval := iterations / 20 // Report 20 times during solve
	if reportInterval < 100 {
		reportInterval = 100
	}

	start := time.Now()

	if isCFR {
		cfr := trainer.(*solver.CFR)

		for i := 0; i < iterations; i++ {
			// Check for cancellation
			select {
			case <-cancelSolve:
				return cfr.GetProfile()
			default:
			}

			// Run iteration
			cfr.Iterate(root)

			// Report progress
			if !progressCallback.IsUndefined() && !progressCallback.IsNull() && (i%reportInterval == 0 || i == iterations-1) {
				progress := map[string]interface{}{
					"iteration": i + 1,
					"total":     iterations,
					"percent":   float64(i+1) / float64(iterations) * 100,
				}
				progressCallback.Invoke(js.ValueOf(progress))
			}

			if time.Since(start) >= maxSolveDuration {
				return cfr.GetProfile()
			}
		}

		return cfr.GetProfile()
	} else {
		mccfr := trainer.(*solver.MCCFR)

		for i := 0; i < iterations; i++ {
			// Check for cancellation
			select {
			case <-cancelSolve:
				return mccfr.GetProfile()
			default:
			}

			// Run iteration
			mccfr.Iterate(root)

			// Report progress
			if !progressCallback.IsUndefined() && !progressCallback.IsNull() && (i%reportInterval == 0 || i == iterations-1) {
				progress := map[string]interface{}{
					"iteration": i + 1,
					"total":     iterations,
					"percent":   float64(i+1) / float64(iterations) * 100,
				}
				progressCallback.Invoke(js.ValueOf(progress))
			}

			if time.Since(start) >= maxSolveDuration {
				return mccfr.GetProfile()
			}
		}

		return mccfr.GetProfile()
	}
}

// parsePositionWrapper wraps the position parser for JavaScript
func parsePositionWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.ValueOf(map[string]interface{}{
			"error": "Usage: parsePosition(positionStr)",
		})
	}

	positionStr := args[0].String()

	gs, err := notation.ParsePosition(positionStr)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Convert to JavaScript-friendly format
	result := map[string]interface{}{
		"pot":    gs.Pot,
		"street": gs.Street.String(),
		"toAct":  gs.ToAct,
		"players": []map[string]interface{}{
			{
				"position": string(gs.Players[0].Position),
				"stack":    gs.Players[0].Stack,
				"combos":   len(gs.Players[0].Range),
			},
			{
				"position": string(gs.Players[1].Position),
				"stack":    gs.Players[1].Stack,
				"combos":   len(gs.Players[1].Range),
			},
		},
	}

	return js.ValueOf(result)
}

// cancelWrapper lets JS request solver cancellation
func cancelWrapper(this js.Value, args []js.Value) interface{} {
	if cancelSolve != nil {
		select {
		case cancelSolve <- true:
		default:
		}
	}
	return js.ValueOf(map[string]interface{}{
		"status": "cancelled",
	})
}
