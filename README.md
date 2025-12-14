# Poker Solver

A dependency-free Go poker solver implementing Counterfactual Regret Minimization (CFR) for computing Game Theory Optimal (GTO) strategies in Heads-Up No-Limit Hold'em.

## Philosophy

**Good enough beats perfect.** This solver prioritizes:
- ✅ **Practical exploitability** < 1% pot in seconds (not theoretical perfection in hours)
- ✅ **WASM-ready performance** for browser-based solving
- ✅ **Zero dependencies** for maximum portability and easy compilation
- ✅ **Clear, maintainable code** over micro-optimizations

## What This Is

A CLI tool (and future WASM library) that:
- Solves river poker situations using vanilla CFR (v0.1)
- Encodes positions in a chess FEN-like compact notation
- Finds Nash equilibrium strategies in milliseconds
- Compiles to native binary with zero dependencies
- Future: WASM support, MCCFR, multi-street solving

## Design Approach

**Incremental complexity:** We build the solver street-by-street (river → turn → flop → multi-street), validating correctness at each phase before adding complexity.

**Real-time performance:** Target sub-10s solves for single-street decisions, making GTO analysis practical during actual play preparation and study.

**Full-spectrum solving:** Provide reasonable probability calculations at any game phase—river, turn, flop, or full game trees—with exploitability bounds appropriate to each scenario.

## Performance

### Actual Measurements (Native Go Binary)

**River Solver (v0.1):**
- **Solve time:** 34ms for 10k CFR iterations (combo vs combo)
- **Throughput:** 293k iterations/sec
- **Strategies:** Converge to Nash equilibrium

**Turn Solver (v0.3):**
- **Solve time:** 200ms for 5k MCCFR iterations (with river rollout)
- **Rollout:** Samples random river cards efficiently
- **Exploitability:** <1% pot after convergence

**Flop Solver (v0.5):**
- **Solve time:** 5.2s for 1k MCCFR iterations (AA,KK vs QQ,JJ with bucketing)
- **Tree reduction:** 76% with bucketing (15 vs 61 info sets)
- **Bucket cache:** 447ns per lookup (58,000× faster than uncached)
- **Equity calc:** Flop: 21ms, Turn: 460μs, River: 9.4μs per combo

### Test Coverage (192 tests passing)
- **pkg/cards:** 86.5% coverage
- **pkg/notation:** 90.2% coverage
- **pkg/solver:** 83.1% coverage
- **pkg/tree:** 83.3% coverage
- **pkg/equity:** 95.5% coverage
- **pkg/abstraction:** 93.1% coverage
- **Overall:** 70.9% total coverage (71% when excluding CLI glue code)

**Dependencies:** Zero (pure stdlib)

## Quick Start

```bash
# Build the solver
make build

# Solve a river position (uses vanilla CFR)
./bin/poker-solver --iterations 10000 "BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d2s|>BTN"

# Solve a turn position (automatically uses MCCFR with river rollout)
./bin/poker-solver --iterations 5000 "BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d|>BTN"

# Solve range-vs-range
./bin/poker-solver --iterations 5000 "BTN:AA,KK:S100/BB:QQ,JJ:S100|P10|Th9h2c5d8s|>BTN"

# Verbose mode shows game state and regrets
./bin/poker-solver --verbose --iterations 5000 "BTN:AA,KK:S100/BB:QQ,JJ:S100|P10|Th9h2c5d8s|>BTN"

# Save strategy to JSON file
./bin/poker-solver --save=strategy.json --iterations 10000 "BTN:AA:S100/BB:QQ:S100|P10|Kh9s4c7d2s|>BTN"

# Load and display saved strategy
./bin/poker-solver --load=strategy.json

# Build WebAssembly binary
make wasm

# Run WASM demo (requires web server)
cd web && python3 -m http.server 8080
# Then open http://localhost:8080

# Run tests
make test

# Run benchmarks
make bench
```

**Example Output (Range-vs-Range):**
```
=== RANGE-VS-RANGE STRATEGIES ===

BTN:
  AA (acts first):
    x: 100.0%
    (averaged over 6 combos)
  KK (acts first):
    x: 100.0%
    (averaged over 6 combos)

BB:
  QQ (facing x):
    x: 20.0%
    b5.0: 20.0%
    b7.5: 20.0%
    ...
```

## Position Notation (Poker FEN)

Compact format for encoding game state:
```
BTN:AsKd:S98.5/BB:??:S97|P3|Th9h2c|r2.5c|>BB
│   │    │     │   │  │  │  │     │    └─ Action indicator
│   │    │     │   │  │  │  │     └────── Action history
│   │    │     │   │  │  │  └──────────── Board cards
│   │    │     │   │  │  └─────────────── Pot size
│   │    │     │   │  └────────────────── Position/range/stack
│   │    │     │   └───────────────────── Stack size
│   │    │     └───────────────────────── Hole cards (or ?? for range)
│   │    └─────────────────────────────── Position label
│   └──────────────────────────────────── Hole cards
└──────────────────────────────────────── Position label
```

See [DESIGN.md](DESIGN.md) for full specification.

## Development Roadmap

### ✅ v0.1 - River Solver (COMPLETE)
- [x] Position notation parser (full FEN support)
- [x] Card evaluation (7-card hand strength)
- [x] Game tree builder (single street, pot-relative bet sizing)
- [x] Vanilla CFR implementation (tested on Kuhn poker + real spots)
- [x] CLI with --iterations and --verbose flags
- [x] Integration tests (end-to-end, symmetric scenarios, performance)

**Status:** Solves river combo-vs-combo in 34ms (10k iterations). All tests passing.

### ✅ v0.2 - Range-vs-Range Solver (COMPLETE)
- [x] Range-vs-range solving with chance nodes
- [x] Aggregated strategy output by hand type
- [x] CLI support for range notation
- [x] Integration tests for range scenarios

**Status:** Solves AA,KK vs QQ,JJ (144 combo pairs) in 3.5s (5k iterations). True GTO equilibrium!

### ✅ v0.3 - Turn Solver (COMPLETE)
- [x] Turn notation parser (already supported in v0.1)
- [x] Turn→river tree builder with rollout nodes
- [x] MCCFR with outcome sampling (efficient for large trees)
- [x] Automatic solver selection (MCCFR for turn, CFR for river)
- [x] Integration tests for turn solving
- [x] Strategy serialization (save/load JSON)
- [x] Exploitability calculation (best response)

**Status:** Turn solver complete! MCCFR samples river cards efficiently. Save/load strategies to JSON. Calculate exploitability with best response algorithm.

### ✅ v0.4 - WASM Export (COMPLETE)
- [x] JavaScript bindings
- [x] Browser-compatible build (`GOOS=js GOARCH=wasm`)
- [x] Progress streaming via callbacks
- [x] Web worker support (non-blocking UI)

**Status:** WASM solver working in browser! 3.2MB binary, Web Worker for background solving, progress callbacks, interactive demo at `web/index.html`.

### ✅ v0.5 - Production Ready (COMPLETE)
- [x] v0.5.1: Hand strength + potential calculation (equity, variance-based potential)
- [x] v0.5.2: Card abstraction (histogram bucketing by equity × potential)
- [x] v0.5.3: Multi-street tree builder (flop→turn→river with bucketing)
- [x] v0.5.4: Geometric bet sizing (target pot-based bet calculation)
- [x] v0.5.5: CLI integration (all features accessible via command-line)

**Status:** Core solver complete! River/turn/flop solving, card abstraction, geometric sizing, range-vs-range—all working. 192 tests passing, 70.9% coverage. Ready for web UI.

### 🚧 v0.6 - Web UI & Learning Platform (IN PROGRESS)
- [ ] Vite + TypeScript project setup
- [ ] Visual range builder (13x13 clickable grid)
- [ ] Position builder (drag-drop cards, visual board)
- [ ] Strategy display (grouped hands, bar charts)
- [ ] Position library (save/load, localStorage)
- [ ] Responsive design (mobile-friendly)
- [ ] Deploy to static host (Vercel/Netlify)

**Goal:** Interactive poker learning website with visual tools. No more manual FEN notation—build positions visually, solve with one click, view strategies in readable format.

## Technical Stack

**Solver (Backend):**
- **Language:** Go 1.21+ (pure stdlib, zero dependencies)
- **Algorithm:** Monte Carlo CFR (MCCFR) with outcome sampling
- **Target:** Native binary + WASM (`GOOS=js GOARCH=wasm`)
- **Input/Output:** JSON + compact notation parsing

**Web UI (Frontend - v0.6+):**
- **Build Tool:** Vite (instant hot reload, zero-config bundling)
- **Language:** TypeScript (type-safe WASM bindings)
- **Framework:** Vanilla JS/TS (no framework lock-in, can add React/Svelte later)
- **Deployment:** Static files (Vercel, Netlify, GitHub Pages)
- **Storage:** localStorage (positions, strategies, preferences)

## Why Go?

1. **WASM-first:** Excellent WebAssembly support out of the box
2. **Performance:** Compiled, garbage-collected, but fast enough for CFR
3. **Simplicity:** Stdlib has everything needed (no `npm install` hell)
4. **Cross-platform:** Single `go build` for any target

## References

- [Regret Minimization in Games with Incomplete Information (Zinkevich et al.)](http://modelai.gettysburg.edu/2013/cfr/cfr.pdf)
- [Monte Carlo Sampling for Regret Minimization (Lanctot et al.)](https://papers.nips.cc/paper/2009/file/00411460f7c92d2124a67ea0f4cb5f85-Paper.pdf)
- [An Introduction to Counterfactual Regret Minimization](http://modelai.gettysburg.edu/2013/cfr/)

## License

MIT - Build cool stuff, share learnings, don't be evil.
