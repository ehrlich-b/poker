# Poker Solver Design Document

## Core Philosophy: Practical GTO in Seconds

**Goal:** Compute unexploitable strategies fast enough for interactive learning, not research-grade perfection.

**Key Insight:** Professional poker players can't detect exploitability <0.5% pot. We target **"grandmaster-level good enough"** in seconds, not **"theoretically optimal"** in hours.

---

## 1. Position Notation Format (Poker FEN)

### Design Goals
- **Compact:** Single-line string encoding full game state
- **Human-readable:** Can parse by eye for debugging
- **URL-safe:** Can embed in query params for web sharing
- **Unambiguous:** One-to-one mapping to game tree root

### Format Specification

```
<players>|<pot>|<board>|<history>|<action>
```

#### Components

**Players:** `POS:CARDS:STACK[/POS:CARDS:STACK]`
- `POS`: Position label (BTN, SB, BB, UTG, etc.)
- `CARDS`: Hole cards (`AsKd`) or `??` for unknown/range
- `STACK`: Stack size in BB (e.g., `S100` = 100bb)

**Pot:** `P{amount}`
- Amount in BB (e.g., `P3` = 3bb pot)

**Board:** `{cards}`
- Flop: `Th9h2c`
- Turn: `Th9h2c/Js` (slash separator)
- River: `Th9h2c/Js/3d`
- Preflop: empty or `-`

**History:** `{action}{action}...`
- `c` = call
- `x` = check
- `f` = fold
- `b{size}` = bet (e.g., `b3.5` = bet 3.5bb)
- `r{size}` = raise (e.g., `r9` = raise to 9bb)

**Action:** `>{position}`
- Who acts next (e.g., `>BTN`)

### Examples

**Example 1: Flop continuation bet spot**
```
BTN:AsKd:S98/BB:??:S97|P3|Th9h2c|>BTN
```
- BTN has AK with 98bb, BB unknown range with 97bb
- Pot is 3bb (BTN raised 2.5bb pre, BB called 2.5bb, blinds 0.5bb+1bb)
- Flop: T♥9♥2♣
- BTN acts first

**Example 2: Turn check-raise spot**
```
BTN:??:S85/BB:Qh Jd:S90|P12|Ad7h3c/5s|bh6xr18|>BTN
```
- BTN range vs BB specific hand (QJ)
- Pot is 12bb
- Board: A♦7♥3♣5♠
- History: BTN bet half-pot (6bb), BB checked, then raised to 18bb
- BTN to act facing raise

**Example 3: River all-in decision**
```
BTN:9s9h:S100/BB:??:S100|P45|Kc9d2h/7c/As|b15cb30c|>BTN
```
- BTN has 99, BB has range
- Pot is 45bb
- Board: K♣9♦2♥7♣A♠
- History: bet 15, call, bet 30, call
- BTN to act

### Card Notation
- Ranks: `A`, `K`, `Q`, `J`, `T`, `9`, ..., `2`
- Suits: `s` (spades), `h` (hearts), `d` (diamonds), `c` (clubs)
- Specific hand: `AsKd`
- Range notation: `AA,KK-JJ,AKs,AQs-AJs,AKo` (expanded to all combos)

**Range Syntax:**
- Pairs: `AA` (6 combos), `KK-JJ` (18 combos)
- Suited: `AKs` (4 combos), `AQs-ATs` (16 combos)
- Offsuit: `AKo` (12 combos), `AQo-AJo` (24 combos)

**v0.1 Implementation:** Range parser is **core**, not optional. GTO requires range-vs-range solving.

### Action Size Notation

**Absolute:** `b12` = bet 12bb (less common, used for specific history parsing)

**Pot-Relative (Primary):** `b0.5p` = bet 50% pot
- `b0.33p` = 33% pot (small bet)
- `b0.66p` = 66% pot (medium bet)
- `b1.5p` = 150% pot (overbet)
- Always includes `all-in` if stack < max bet

**v0.1 Implementation:** Pot-relative is **mandatory**. Tree builder takes `[0.33, 0.66, 1.5]` config and applies at each node.

---

## 2. System Architecture

### Package Structure

```
poker-solver/
├── cmd/
│   └── poker-solver/          # CLI entry point
│       └── main.go
├── pkg/
│   ├── cards/                 # Card primitives and evaluation
│   │   ├── card.go           # Card type, parsing
│   │   ├── deck.go           # Deck shuffling, dealing
│   │   ├── hand.go           # Hand strength evaluation
│   │   └── equity.go         # Equity calculations
│   ├── notation/              # Position notation parser
│   │   ├── parser.go         # FEN-like format → GameState
│   │   └── types.go          # GameState, Action types
│   ├── tree/                  # Game tree construction
│   │   ├── builder.go        # Builds decision tree from GameState
│   │   ├── node.go           # TreeNode, InfoSet types
│   │   └── actions.go        # Legal action generation
│   ├── solver/                # CFR implementation
│   │   ├── cfr.go            # Vanilla CFR
│   │   ├── mccfr.go          # Monte Carlo CFR (outcome sampling)
│   │   ├── strategy.go       # Strategy storage and retrieval
│   │   └── exploitability.go # Best response calculation
│   └── output/                # Strategy formatting
│       ├── json.go           # JSON serialization
│       └── pretty.go         # Human-readable output
└── wasm/                      # WebAssembly bindings (future)
    ├── bindings.go
    └── index.html
```

### Core Data Structures

#### GameState
```go
type GameState struct {
    Players      []Player       // Positions, stacks, hole cards
    Pot          float64        // Current pot in BB
    Board        []Card         // Community cards
    ActionHistory []Action      // Past actions
    ToAct        int            // Player index to act
}
```

#### TreeNode
```go
type TreeNode struct {
    InfoSet      string         // Information set key: f(public_history, private_cards)
    Player       int            // Acting player (0 or 1)
    Pot          float64
    Actions      []Action       // Legal actions from this node
    Children     map[Action]*TreeNode
    IsTerminal   bool           // Showdown or fold
    Payoff       [2]float64     // Terminal payoffs for each player
}
```

**InfoSet Key Format:**
- Encodes what the player knows: public board/history + their own cards
- Example: `"Kh9s4c7d2s|b10|>BB|QdJd"` (river, facing bet, holding QJ)
- With bucketing: `"Kh9s4c7d2s|b10|>BB|BUCKET_5"` (river, facing bet, mid-strength)

#### Strategy
```go
type Strategy struct {
    InfoSet   string
    Actions   []Action
    Probs     []float64        // Probability distribution over actions
    RegretSum []float64        // Cumulative regret for each action
    StratSum  []float64        // Cumulative strategy for averaging
}
```

---

## 3. Monte Carlo CFR Algorithm

### Why MCCFR?

**Vanilla CFR:** Traverses entire game tree every iteration
- Time per iteration: O(tree size)
- Memory: O(information sets)
- For single-street poker: manageable but slow

**Monte Carlo CFR (Outcome Sampling):** Samples single trajectory per iteration
- Time per iteration: O(tree depth)
- Memory: same as vanilla
- Convergence: Slightly slower in iterations, but **much faster wall-clock time**
- **Best for WASM:** Avoids browser freezing with incremental progress

### Convergence Analysis

#### Single-Street Postflop (Realistic Scenario)
- **Tree size:** ~1,000-10,000 terminal nodes (2 players, 3-4 bet sizes, 1 street)
- **Information sets:** ~500-2,000 (depends on board texture)
- **Iterations needed:**
  - 10k iterations → exploitability ~1-2% pot
  - 50k iterations → exploitability ~0.5% pot ✅ **Grandmaster level**
  - 100k iterations → exploitability ~0.2% pot (overkill for humans)

#### Performance Benchmarks (Estimated)
| Scenario | Info Sets | Iterations | Native (Go) | WASM (Browser) |
|----------|-----------|------------|-------------|----------------|
| River (2 sizes) | 100-300 | 10k | <1s | ~5s |
| River (3 sizes) | 300-500 | 10k | ~2s | ~10s |
| Turn→river (2 sizes) | 500-1,000 | 50k | ~10s | ~60s |
| Flop→turn→river (2 sizes) | 2,000-5,000 | 100k | ~60s | ~300s |

**v0.1 Target:** River with 2 sizes, <5s native (achievable)

**Confidence:** These targets are **achievable** based on:
1. Go's performance (~10-20x slower than C++, but solver is compute-bound)
2. WASM overhead (~5-10x slowdown for CPU-intensive tasks)
3. Simple tree traversal (no complex data structures)

### Error Bounds

**Exploitability** = max EV an omniscient opponent could gain playing best response

**Rule of Thumb:**
- Exploitability < 1% pot → Indistinguishable from GTO for humans
- Exploitability < 0.1% pot → "Solved" for all practical purposes
- Exploitability < 0.01% pot → Research-grade (unnecessary overkill)

**Human Context:**
- World-class players make ~2-5% EV mistakes per hand (in complex spots)
- Solver with 0.5% exploitability is **better than any human**
- We're building grandmaster+ level tools, not AlphaZero

---

## 4. Abstraction Strategies

### Card Abstraction (Future Optimization)

**Problem:** 1,326 starting hand combos → 2,809,475,760 river runouts
**Solution:** Bucket similar hands by equity + potential

**Example: Flop Bucketing**
```
Bucket 1: Overpairs (AA-QQ on T92 board)
Bucket 2: Top pair strong kicker (ATs, KTs)
Bucket 3: Top pair weak kicker (T9s, T8s)
Bucket 4: Flush draws (AhXh, KhXh)
Bucket 5: Gutshots (QJ, 87)
...
Bucket N: Air (low unpaired hands)
```

**Buckets needed:**
- Flop: 50-200 buckets (balance accuracy vs speed)
- Turn: 50-200 buckets (re-bucket based on turn card)
- River: 50-200 buckets (final hand strength)

**Impact:**
- Reduces info sets by ~10-100x
- Slight accuracy loss (~1-3% exploitability increase)
- **Worth it for multi-street solving**

### Bet Abstraction

**Problem:** Continuous bet sizing = infinite tree size
**Solution:** Discretize to 2-4 geometrically-spaced sizes

**Common Abstractions:**
```
Aggressive:
- Small: 33% pot (value + protection)
- Medium: 75% pot (balanced bluffs/value)
- Large: 150% pot (polarized)

Conservative:
- Half-pot: 50%
- Pot: 100%

GTO-Inspired:
- Geometric: 66-75% (optimal tree growth for multi-street)
```

**v0.1 Target:** 2-3 fixed sizes per node (user-configurable)

---

## 5. WASM Compilation Strategy

### Build Commands
```bash
# Native binary
go build -o poker-solver ./cmd/poker-solver

# WASM
GOOS=js GOARCH=wasm go build -o poker-solver.wasm ./wasm

# Size optimization
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o poker-solver.wasm ./wasm
tinygo build -o poker-solver.wasm -target wasm ./wasm  # Alternative: 10x smaller
```

### JavaScript Interop

```javascript
// Load WASM module
const go = new Go();
const result = await WebAssembly.instantiateStreaming(
  fetch("poker-solver.wasm"),
  go.importObject
);
go.run(result.instance);

// Solve position with progress callback
await solvePosition(
  "BTN:AsKd:S100/BB:??:S100|P3|Th9h2c|>BTN",
  {
    iterations: 50000,
    onProgress: (iter, exploitability) => {
      console.log(`Iteration ${iter}: ${exploitability}% exploitable`);
    }
  }
);
```

### Performance Considerations

**Challenges:**
- No goroutines → single-threaded (WASM limitation)
- Slower memory access (~2-3x overhead)
- GC pauses visible in browser

**Optimizations:**
- Stream results incrementally (don't block UI)
- Use Web Workers for background solving
- Cache strategies in IndexedDB
- Lazy-load large game trees

---

## 6. Implementation Phases

### **CRITICAL: Start with River, Not Flop**

**Why River First:**
1. **Simplest street:** No draws, no potential, just fixed hand strength
2. **Easy validation:** Small trees, can verify correctness by inspection
3. **Trivial bucketing:** River hand strength = percentile rank (no complex equity calcs)
4. **Builds confidence:** Proves CFR implementation works before adding complexity

**Progression:** River (v0.1) → Turn (v0.2) → Flop (v0.3) → WASM (v0.4)

---

### Phase 1: River Solver (v0.1)
**Goal:** Solve single river decision in <5s with <1% exploitability

**Features:**
- [x] Position notation parser (river boards only)
- [ ] Range parser (`AA,KK-JJ,AKs` → combos)
- [ ] Card evaluation (7-card hand strength)
- [ ] Pot-relative bet sizing (`b0.5p`, `b1.5p`)
- [ ] Tree builder (single decision, 2-3 bet sizes)
- [ ] Vanilla CFR (get it working first)
- [ ] CLI: `solve` command
- [ ] JSON output (strategy per combo)

**Test Case:**
```
Input:  BTN:AA,KK,AKs/BB:QQ-JJ,AJs-ATs|P20|Kh9s4c7d2s|>BTN
Solve:  18 BTN combos vs 18 BB combos (324 matchups)
Output:
  BTN AA: bet_1.5p 85%, bet_0.66p 10%, check 5%
  BTN KK: bet_1.5p 80%, bet_0.66p 15%, check 5%
  BTN AKs: bet_0.66p 60%, check 40%
  Exploitability: 0.7% pot
  Runtime: 2.1s (10k iterations)
```

**Why This Works:**
- River = no card abstraction needed (hand strength is fixed)
- Small tree (~100-500 terminal nodes for 2 ranges)
- Can validate correctness (e.g., AA should always bet for value)

---

### Phase 2: Turn Solver (v0.2)
**Goal:** Solve turn decision with river rollout in <15s

**Features:**
- [ ] Turn notation parser
- [ ] Tree builder: turn decision → river outcomes
- [ ] MCCFR with outcome sampling (needed for turn→river tree)
- [ ] Exploitability calculator (best response)
- [ ] Strategy serialization (save/load)
- [ ] 3-4 bet size support

**Test Case:**
```
Input:  BTN:AA,AKs/BB:KK,AJs|P15|Kh9s4c/7d|>BTN
Solve:  Turn decision, sample river cards
Output: BTN AA: bet_0.66p 70%, check 30%
        Exploitability: 0.5% pot
        Runtime: 12s (50k iterations)
```

**Complexity Jump:**
- Each turn terminal node → sample random river card
- Need MCCFR (vanilla CFR too slow for turn→river tree)
- Card abstraction still not needed (can solve all river runouts)

---

### Phase 3: Flop Solver (v0.3)
**Goal:** Solve flop decision with turn/river rollout

**Features:**
- [ ] Flop notation parser
- [ ] Card bucketing for turn/river (equity + potential)
- [ ] Multi-street tree (flop → turn → river)
- [ ] Abstraction testing (compare bucketed vs unbucketed)

**Test Case:**
```
Input:  BTN:AA,AKs/BB:KK-JJ,AQs-AJs|P3|Th9h2c|>BTN
Solve:  Flop decision with 2 future streets
Output: Exploitability: <1% pot
        Runtime: 30-60s (100k iterations)
```

**Complexity Jump:**
- Flop is hardest street (board texture, draws, 2 future cards)
- Requires card abstraction (too many turn/river combos to solve individually)
- This is the "final boss" of single-street solvers

---

### Phase 4: WASM Export (v0.4)
**Goal:** Browser-based solving with Web Workers

**Features:**
- [ ] JavaScript bindings
- [ ] Progress streaming (callback every 100 iterations)
- [ ] Web Worker integration (mandatory, not optional)
- [ ] Browser demo page

**Performance Target:**
- River solve: <10s in browser
- Turn solve: <30s in browser
- Flop solve: <90s in browser

**Critical:** Web Workers are **non-negotiable**. Any solve >1s must run in background thread.

---

### Phase 5: Production Ready (v0.5+)
**Goal:** Multi-street + full abstraction

**Features:**
- [ ] Geometric bet sizing (auto-calculate optimal sizes)
- [ ] Range builder UI (visual range selection)
- [ ] Exploitability visualization (equity graph)
- [ ] Multi-street solving (preflop → flop → turn → river)
- [ ] Strategy comparison (solver vs real play)

**Scope Warning:** Multi-street is an **exponential complexity jump**. Only pursue after v0.1-0.4 are rock-solid.

---

## 7. Testing Strategy

### Unit Tests
- Card evaluation: All hand rankings, equity calculations
- Parser: Valid/invalid notation strings
- Tree builder: Correct action generation, pot math
- CFR: Toy games (Kuhn poker, Leduc Hold'em)

### Integration Tests
- Known solved games (Kuhn poker: verify strategies match literature)
- Symmetric scenarios (both players should have identical strategies)
- Exploitability bounds (verify convergence)

### Benchmarks
```go
BenchmarkTreeBuild     # Tree construction time
BenchmarkCFRIteration  # Single iteration time
BenchmarkMCCFR10k      # 10k iterations (target: <1s)
BenchmarkExploitability # Best response calculation
```

---

## 8. Future Extensions

### Nice-to-Haves (Post-v1.0)
- **Preflop solver:** Huge trees, needs heavy abstraction
- **Range constructors:** Input "BTN 3bet range" → generate combos
- **Hand history import:** Solve real hands from HH files
- **Multiplayer:** 3+ players (CFR still works, but trees explode)
- **Limit Hold'em:** Simpler tree, faster solving
- **PLO:** 4-card combos = 270,725 starting hands (needs aggressive bucketing)

### Research Ideas
- **Deep CFR:** Neural network value approximation (requires ML libs → breaks "dependency-free")
- **Pruning:** Skip unlikely branches (careful: can introduce bias)
- **Discounted CFR:** Faster convergence (CFR+ variant)

---

## 9. Success Metrics

### v0.1 (River Solver - MVP)
- ✅ Solves river decision in <5s (native)
- ✅ Range parser works (`AA,KK-JJ` → correct combos)
- ✅ Pot-relative bet sizing (`b0.5p` auto-calculated)
- ✅ Exploitability <1% pot (10k iterations)
- ✅ 100% unit test coverage on `pkg/cards` and `pkg/solver`
- ✅ Zero dependencies (stdlib only)
- ✅ Symmetric spots produce symmetric strategies

### v0.2 (Turn Solver)
- ✅ Solves turn→river in <15s (native)
- ✅ MCCFR implemented and tested
- ✅ Exploitability <0.5% pot (50k iterations)

### v0.3 (Flop Solver)
- ✅ Solves flop→turn→river in <60s (native)
- ✅ Card abstraction (bucketing) working
- ✅ Exploitability <1% pot (with abstraction)

### v0.4 (WASM)
- ✅ River solve: <10s in browser
- ✅ <5MB WASM binary size
- ✅ Web Workers preventing UI freeze
- ✅ Progress streaming working

### v1.0 (Production)
- ✅ All streets (river/turn/flop) working in browser
- ✅ Range builder UI
- ✅ Strategy visualization
- ✅ Used in production poker learning site

---

## References

### Academic Papers
- [Regret Minimization in Games with Incomplete Information](http://modelai.gettysburg.edu/2013/cfr/cfr.pdf) (Zinkevich et al., 2007)
- [Monte Carlo Sampling for Regret Minimization in Extensive Games](https://papers.nips.cc/paper/2009/file/00411460f7c92d2124a67ea0f4cb5f85-Paper.pdf) (Lanctot et al., 2009)
- [Solving Imperfect Information Games Using Decomposition](https://arxiv.org/abs/1811.06233) (Brown et al., 2018)

### Practical Resources
- [University of Alberta CPRG](https://poker.cs.ualberta.ca/)
- [CMU Poker Research](http://www.cs.cmu.edu/~sandholm/research.htm)
- [GTO+ Documentation](https://www.gtoplus.com/) (commercial solver)

### Go + WebAssembly
- [Go WASM Wiki](https://github.com/golang/go/wiki/WebAssembly)
- [TinyGo WASM](https://tinygo.org/docs/guides/webassembly/) (smaller binaries)
