# Poker Solver - Claude Code Guidance

Project-specific instructions for working on the poker solver.

## Core Philosophy

**"Good enough beats perfect"** - Build a grandmaster-level solver in seconds, not a research tool in hours.

## Critical Design Decisions (From Review)

### 1. START WITH RIVER, NOT FLOP

**Why River First:**
- ✅ Simplest street: no draws, no potential, just fixed hand strength
- ✅ Trivial card abstraction: bucket by hand strength percentile
- ✅ Easy validation: small enough to verify correctness by inspection
- ✅ Builds confidence in CFR implementation before complexity

**Progression:**
1. v0.1: River-only solver (single decision point)
2. v0.2: Turn solver (turn decision → river solver at terminals)
3. v0.3: Flop solver (most complex, do last)

This is the **correct architectural path** - not a nice-to-have.

### 2. Range-vs-Range is CORE, Not Future

**Problem:** GTO is a range-vs-range equilibrium, not hand-vs-range.

**Solution for v0.1:**
- Implement simple range parser: `AA,KK,AKs,QQ-TT,AQs-AJs`
- Expand to specific combos (e.g., `AA` → 6 combos)
- Tree builder solves for every combo in both ranges
- Output shows frequencies per action per combo

**Example:**
```
Input:  BTN:AA,KK,AKs/BB:QQ-77,AJs-ATs|P20|Kh9s4c/7d/2s|>BTN
Solve:  BTN range (18 combos) vs BB range (36 combos)
Output: BTN AA: bet 100%, check 0%
        BTN KK: bet 85%, check 15%
        BTN AKs: bet 60%, check 40%
```

### 3. Pot-Relative Bet Sizing is Mandatory

**Problem:** Users shouldn't manually calculate `b3.5` for a 7bb pot.

**Solution:**
- Config takes pot fractions: `[0.33, 0.66, 1.5]` (33%, 66%, 150% pot)
- Tree builder applies these at each decision node
- Always include "all-in" as an action if stack < max bet size

**Example:**
```
Pot: 10bb, Stack: 30bb
Actions: fold, call, bet_3.3 (33%), bet_6.6 (66%), bet_15 (150%), bet_30 (all-in)
```

### 4. Performance Critical Path

**The Bottleneck Hierarchy:**
1. **`pkg/cards/hand.go:Evaluate()`** - Called millions of times per solve
2. **`pkg/solver/mccfr.go`** - Tree traversal loop
3. **Everything else** - Not the bottleneck

**Implications:**
- Write `Evaluate()` first, benchmark obsessively
- Use lookup tables if needed (e.g., 7-card eval can use 12-bit hash tables)
- Accept that pure Go will be 5-10x slower than C/Rust (still fast enough)
- If stuck: profile first, optimize second

**Benchmark Targets:**
- `Evaluate()`: <100ns per call (achievable with lookup tables)
- `MCCFRIteration()`: <1ms per iteration (for ~1k infosets)
- Full solve (10k iters): <5s native, <30s WASM

### 5. Information Set Keys (Technical Correctness)

**InfoSet = What the player knows:**
- Public history: board cards, action sequence
- Private cards: player's own hole cards
- **NOT** opponent's range (that's what we're solving for)

**Example Keys:**
```
River, BTN acts first with AhKh:
  "Kh9s4c7d2s||>BTN|AhKh"

River, BB faces bet with Qd Jd:
  "Kh9s4c7d2s|b10|>BB|QdJd"
```

With bucketing:
```
  "Kh9s4c7d2s||>BTN|BUCKET_8"  (top pair)
```

### 6. WASM Web Workers are NON-NEGOTIABLE

**Problem:** Any computation >1s will freeze the browser tab.

**Solution (v0.3):**
- Solve runs in Web Worker (background thread)
- Progress callback every N iterations: `{iter: 5000, exploit: 0.8%}`
- Main thread updates UI (progress bar, current strategy)
- User can cancel mid-solve

**Implementation:**
```javascript
// main.js
const worker = new Worker('solver-worker.js');
worker.postMessage({ position: "...", iterations: 50000 });
worker.onmessage = (e) => {
  if (e.data.progress) updateProgressBar(e.data.iter);
  if (e.data.result) displayStrategy(e.data.result);
};
```

## Code Quality Standards

### Testing Requirements

**Every package must have:**
- Unit tests for all exported functions
- Benchmark tests for performance-critical paths
- Example tests for documentation

**Critical test cases:**
- Toy games (Kuhn poker) must solve to known equilibrium
- Symmetric scenarios must produce symmetric strategies
- Exploitability must decrease monotonically with iterations

### Benchmarking Protocol

**Before optimizing:**
```bash
go test -bench=. -benchmem ./pkg/cards > before.txt
# make changes
go test -bench=. -benchmem ./pkg/cards > after.txt
benchstat before.txt after.txt
```

**Target benchmarks:**
- `BenchmarkEvaluate`: <100ns/op, 0 allocs
- `BenchmarkMCCFRIteration`: <1ms/op (for 1k infosets)
- `BenchmarkFullSolve10k`: <5s total

## Development Workflow

### v0.1 River Solver Checklist

- [ ] `pkg/cards`: Card types, deck, hand evaluation (with benchmarks!)
- [ ] `pkg/notation`: Parse river position notation
- [ ] `pkg/notation`: Range parser (`AA,KK-QQ` → combos)
- [ ] `pkg/tree`: Build single-decision river tree
- [ ] `pkg/solver`: Vanilla CFR (get it working)
- [ ] `cmd/poker-solver`: CLI with `solve` command
- [ ] Test: Solve simple river spot, verify exploitability <1%

### When Stuck

1. **Re-read anchor docs:** README.md, DESIGN.md, this file
2. **Check assumptions:** Is the tree building correctly? Are regrets accumulating?
3. **Test on toy game:** Does Kuhn poker solve correctly?
4. **Profile:** Don't guess where the slowness is
5. **Ask for validation:** Show intermediate results

## Common Pitfalls

### Don't Do This

❌ Implement flop solver first (too complex to validate)
❌ Skip benchmarking hand evaluation (will regret later)
❌ Hard-code bet sizes as absolute BB amounts (unusable)
❌ Solve hand-vs-range (not GTO, misleading results)
❌ Block browser main thread with WASM (instant user frustration)

### Do This

✅ Start with river (simplest, validate CFR works)
✅ Write benchmarks for `Evaluate()` before writing the function
✅ Use pot-relative bet sizes from day one
✅ Parse ranges, solve range-vs-range
✅ Use Web Workers for any WASM computation >1s

## File Organization Discipline

**Keep packages small and focused:**
- `pkg/cards`: Pure card logic, zero game-tree knowledge
- `pkg/notation`: Pure parsing, zero solver knowledge
- `pkg/tree`: Pure tree building, zero CFR knowledge
- `pkg/solver`: Pure CFR, zero I/O knowledge
- `cmd/poker-solver`: Glue code, orchestration only

**Test files co-located:**
```
pkg/cards/
  card.go
  card_test.go      # unit tests
  deck.go
  deck_test.go
  hand.go
  hand_test.go
  hand_bench_test.go # benchmarks for Evaluate()
```

## Success Criteria Reminder

**v0.1 is successful when:**
- Single river decision solves in <5s (native)
- Exploitability <1% pot (10k iterations)
- 100% test coverage on `pkg/cards` and `pkg/solver`
- Symmetric spots produce symmetric strategies (validation)
- Range parser works: `AA,KK-JJ` → correct 18 combos

**v0.1 is NOT about:**
- WASM (that's v0.3)
- Multi-street (that's v0.4)
- Card abstraction (river doesn't need it)
- Perfect exploitability (0.5% is plenty)

## References Quick Links

**Academic:**
- CFR paper: http://modelai.gettysburg.edu/2013/cfr/cfr.pdf
- MCCFR paper: https://papers.nips.cc/paper/2009/file/00411460f7c92d2124a67ea0f4cb5f85-Paper.pdf

**Practical:**
- Go WASM: https://github.com/golang/go/wiki/WebAssembly
- Hand evaluator (reference): https://github.com/chehsunliu/poker

## Update This File

When you discover critical insights or make architectural decisions, **immediately update this file**. Future Claude (or future you) will thank you.
