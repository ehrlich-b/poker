# Poker Solver - TODO

## Current Status: v0.1 River Solver (In Progress)

**Last Updated:** 2025-10-17 - Session 2 (In Progress)

### **Session 1 Summary (2025-10-16):**
**Completed:**
- ✅ Project scaffolding (README, DESIGN, CLAUDE, Makefile, Go module)
- ✅ pkg/cards - Full hand evaluation with tests & benchmarks (~5μs per eval)
- ✅ pkg/notation - Range parser + core types (100% tested)
  - Range parser: `AA,KK-JJ,AKs,AQo` → all combos
  - Action types, GameState, Position/Street enums
  - 25 test functions, 0 failures

**Status:** ~40% of v0.1 complete (2 of 5 components done)

### **Session 2 Summary (2025-10-17):**
**Completed:**
- ✅ Fixed README.md to reflect full project ambition (removed artificial limitations)
- ✅ pkg/notation - Position FEN parser (100% tested)
  - Full FEN parsing: `BTN:range:stack/BB:range:stack|pot|board|history|action`
  - Supports specific cards, ranges, and unknown ranges (??)
  - Action history parsing (bet, raise, call, check, fold with amounts)
  - All streets: preflop, flop, turn, river
  - 45+ comprehensive tests, all passing
- ✅ pkg/tree - Game Tree Builder (100% tested)
  - TreeNode struct with info sets, children, terminal nodes
  - InfoSet key generation: `"board|history|>player|cards"`
  - Action generator with pot-relative bet sizing
  - Recursive tree builder for combo vs combo matchups
  - Showdown and fold payoff calculation
  - 20+ comprehensive tests, all passing

**Status:** ~80% of v0.1 complete (4 of 5 components done: cards, notation, tree. Remaining: solver, CLI)

**Next:** CFR solver → CLI → v0.1 complete!

---

### pkg/tree - Game Tree Builder
- [x] Define `TreeNode` struct (decision and terminal nodes)
- [x] Define `InfoSet` key format (`"board|history|>player|cards"`)
- [x] Implement `ActionConfig` for configurable action spaces
- [x] Implement action generator (GenerateActions)
  - Check/bet for no action, fold/call for facing bet
  - Pot-relative bet sizes (e.g., 0.5p, 0.75p, 1.5p)
  - Automatic all-in when bet >= stack
- [x] Implement recursive tree builder (`Builder.Build`)
  - Builds tree for specific combo vs combo
  - Handles showdown terminals (both check or bet→call)
  - Handles fold terminals
- [x] Calculate showdown payoffs using hand evaluation
- [x] Write comprehensive unit tests (100% coverage)

**Test Results:**
- All 20+ tree tests passing ✓
- InfoSet generation correct ✓
- Action generation with various configs ✓
- Tree structure validated (check→check→showdown, bet→fold, etc.) ✓
- Showdown uses hand evaluator correctly ✓

## ✅ Completed

### Project Setup
- [x] Create README.md with project overview
- [x] Create DESIGN.md with river-first approach
- [x] Create CLAUDE.md for project-specific guidance
- [x] Initialize Go module
- [x] Create directory structure (`pkg/cards`, `pkg/notation`, etc.)
- [x] Create Makefile with build/test/bench targets

### pkg/cards - Card Evaluation
- [x] Implement Card type with Rank and Suit
- [x] Implement card parsing (`ParseCard`, `ParseCards`)
- [x] Implement 7-card hand evaluator (`Evaluate`)
- [x] Handle all hand rankings (straight flush, quads, full house, etc.)
- [x] Handle wheel straight (A-2-3-4-5) correctly
- [x] Fix unsigned integer underflow bugs in rank iteration
- [x] Implement hand comparison (`Compare`)
- [x] Write comprehensive unit tests (100% coverage on core logic)
- [x] Write benchmark tests and establish baseline

**Baseline Performance (bench.txt):**
- `Evaluate`: ~5,000 ns/op (5 μs)
- `Compare`: ~1 ns/op (essentially free)
- `ParseCard`: ~8 ns/op (very fast)

**Known Optimization Opportunities (defer to later):**
- 84 allocations per Evaluate call (should be 0)
- ~3.5 KB memory per evaluation
- Could use lookup tables instead of computing each time

### pkg/notation - Range Parser & Types
- [x] Implement `Combo` type for hole card combinations
- [x] Implement range parser: `AA,KK-JJ,AKs,AQo-AJo` → combos
- [x] Handle pair ranges (AA-KK), suited ranges (AKs-ATs), offsuit ranges (AQo-AJo)
- [x] Validate range syntax (error on `AA-KKo`, ambiguous `AK`)
- [x] Define Action types (Check, Call, Bet, Raise, Fold)
- [x] Define GameState struct (players, pot, board, action history)
- [x] Define Position, Street enums
- [x] Write comprehensive unit tests (100% coverage)

**Test Results:**
- `ParseRange("AA,KK,AKs")` → 16 combos ✓
- `ParseRange("QQ-JJ,AJs-ATs")` → 20 combos ✓
- Error handling validated for invalid syntax

### pkg/notation - Position FEN Parser
- [x] Implement Position FEN parser (`ParsePosition`)
- [x] Parse players with ranges: `BTN:AA,KK:S100/BB:QQ-JJ:S100`
- [x] Parse pot: `P20` → 20bb
- [x] Parse board: `Kh9s4c7d2s` (with slash support for streets)
- [x] Parse action history: `b10c` → [bet 10, call]
- [x] Parse action indicator: `>BTN` → BTN to act
- [x] Support specific cards (`AsKd`), ranges (`AA,KK`), and unknown (`??`)
- [x] Write comprehensive unit tests (all streets, error cases)

**Test Results:**
- All 45+ parser tests passing ✓
- Handles flop, turn, river, and preflop positions ✓
- Error handling for malformed FENs ✓
- Action history parsing with decimals: `b3.5c` ✓

---

## 🚧 In Progress

### v0.1 River Solver - Next Steps

#### 1. pkg/notation - Position Notation Parser ✅ COMPLETE
- [x] Define `GameState` struct ✓
- [x] Define `Action` types (check, call, bet, raise, fold) ✓
- [x] Implement range parser: `AA,KK-JJ,AKs` → combos ✓
- [x] Implement river position FEN parser ✓
- [x] Write comprehensive tests for range expansion ✓
- [x] Write comprehensive tests for FEN parser ✓

**Progress:** Position FEN parser complete with 100% test coverage! Can now parse full position strings like `"BTN:AA,KK:S100/BB:QQ-JJ:S100|P20|Kh9s4c7d2s|>BTN"` into GameState.

**Example Target:**
```
Input:  BTN:AA,KK,AKs/BB:QQ-JJ,AJs-ATs|P20|Kh9s4c7d2s|>BTN
Output: GameState{
  BTN range: 18 combos (AA=6, KK=6, AKs=4)
  BB range: 18 combos (QQ=6, JJ=6, AJs=4, ATs=4)
  Pot: 20bb
  Board: K♥9♠4♣7♦2♠ (river)
  Action: BTN to act
}
```

#### 2. pkg/tree - Game Tree Builder ✅ COMPLETE
- [x] Define `TreeNode` struct ✓
- [x] Define `InfoSet` key format ✓
- [x] Implement action generator (check, bet X%, bet Y%, all-in) ✓
- [x] Build single-decision river tree ✓
- [x] Calculate pot odds and payoffs at terminals ✓
- [x] Write tree traversal tests ✓

**Progress:** Complete tree builder implementation! Can build full game trees for specific combo matchups with:
- InfoSet generation: `"Kh9s4c7d2s|b10|>BB|QdJd"`
- Configurable action spaces (pot-relative bet sizes)
- Recursive tree building with showdown/fold terminals
- Full test coverage (20+ tests)

**Test Results:**
- TreeNode creation and info sets ✓
- Action generation (check, bet, call, fold) ✓
- Tree building for AA vs QQ scenarios ✓
- Showdown payoff calculation ✓
- Fold payoff calculation ✓

#### 3. pkg/solver - Vanilla CFR
- [ ] Define `Strategy` struct (regret sums, strategy sums)
- [ ] Implement CFR iteration
- [ ] Implement regret matching
- [ ] Implement average strategy calculation
- [ ] Calculate exploitability (best response)
- [ ] Write CFR tests on toy game (Kuhn poker)
- [ ] Test on simple river spot

**Toy Game Test:**
- Solve Kuhn poker (known solution)
- Verify strategies converge to Nash equilibrium
- Verify exploitability decreases monotonically

#### 4. cmd/poker-solver - CLI
- [ ] Implement `solve` command
- [ ] Implement strategy output (JSON format)
- [ ] Add progress reporting (iterations, exploitability)
- [ ] Add CLI flags (iterations, output file, etc.)

**Example CLI:**
```bash
./poker-solver solve "BTN:AA,KK/BB:QQ,JJ|P20|Kh9s4c7d2s|>BTN" \
  --iterations 10000 \
  --output strategy.json

Output:
Iteration 10000: Exploitability 0.7% pot
Solved in 2.1s

Strategy (BTN with AA):
  bet_1.5p: 85%
  bet_0.66p: 10%
  check: 5%
```

#### 5. Integration & Testing
- [ ] End-to-end test: parse → build tree → solve → output
- [ ] Symmetric scenario test (both players same range → same strategy)
- [ ] Known solution test (simple spot, verify correctness)
- [ ] Performance test: <5s solve time for simple river spot

---

## 📋 v0.1 Success Criteria

Before marking v0.1 complete, verify:

- ✅ **Correctness:**
  - [ ] Solves simple river spot in <5s
  - [ ] Exploitability <1% pot (10k iterations)
  - [ ] Symmetric spots produce symmetric strategies
  - [ ] Range parser correctly expands `AA,KK-JJ` to 18 combos

- ✅ **Code Quality:**
  - [ ] 100% test coverage on pkg/cards ✓
  - [ ] 100% test coverage on pkg/solver
  - [ ] Zero dependencies (stdlib only) ✓
  - [ ] All benchmarks passing

- ✅ **Documentation:**
  - [ ] README reflects current state
  - [ ] Examples work as documented
  - [ ] DESIGN.md updated with learnings

---

## 🔮 Future Versions (Not Now)

### v0.2 - Turn Solver
- [ ] Turn notation parser
- [ ] Turn→river tree builder
- [ ] MCCFR with outcome sampling
- [ ] Strategy serialization (save/load)
- [ ] 3-4 bet size support

### v0.3 - Flop Solver
- [ ] Flop notation parser
- [ ] Card bucketing for turn/river
- [ ] Multi-street tree (flop→turn→river)
- [ ] Abstraction testing

### v0.4 - WASM Export
- [ ] JavaScript bindings
- [ ] Web Worker integration
- [ ] Progress streaming
- [ ] Browser demo page

### v0.5 - Production Ready
- [ ] Geometric bet sizing
- [ ] Range builder UI
- [ ] Exploitability visualization
- [ ] Multi-street solving

---

## 🔥 Known Issues

### Performance
- **pkg/cards/hand.go:** 84 allocations per Evaluate call
  - Target: 0 allocations
  - Solution: Preallocate buffers, avoid slice creation
  - Priority: LOW (optimize after v0.1 works)

### Technical Debt
- None yet (v0.1 just started)

---

## 📊 Benchmark History

**2025-10-16 - Baseline (Initial Implementation)**
```
BenchmarkEvaluate (single hand):  ~5,000 ns/op   3,528 B/op   84 allocs/op
BenchmarkCompare:                 ~1 ns/op       0 B/op       0 allocs/op
BenchmarkParseCard:               ~8 ns/op       0 B/op       0 allocs/op
```

---

## 🎯 Next Session Goals

### **Session 3 (Next Time):**
1. **pkg/tree** - Game Tree Builder
   - Define `TreeNode` struct
   - Define `InfoSet` key format
   - Implement action generator (check, bet X%, bet Y%, all-in)
   - Build single-decision river tree
   - Calculate pot odds and payoffs at terminals
   - Write tree traversal tests

2. **pkg/solver** - Vanilla CFR (start)
   - Define `Strategy` struct (regret sums, strategy sums)
   - Implement CFR iteration basics
   - Implement regret matching

### **This Week:**
- Complete v0.1 river solver (all 5 components)
- Get first end-to-end solve working

### **This Month:**
- Polish v0.1, optimize, document
- Start v0.2 turn solver

---

## 💡 Design Decisions Log

### 2025-10-16: River-First Approach
**Decision:** Start with river solver, then turn, then flop
**Rationale:** River is simplest (no draws), easiest to validate, builds confidence
**Source:** Design review feedback

### 2025-10-16: Range-vs-Range as Core
**Decision:** Range parser is v0.1, not "future"
**Rationale:** GTO requires range-vs-range solving, not hand-vs-range
**Source:** Design review feedback

### 2025-10-16: Accept 84 Allocations for Now
**Decision:** Don't optimize Evaluate() yet
**Rationale:** Prove correctness first, optimize when profiling shows it's the bottleneck
**Target:** Optimize in v0.2 when MCCFR performance matters more
