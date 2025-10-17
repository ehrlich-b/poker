# Poker Solver

A dependency-free Go poker solver implementing Monte Carlo Counterfactual Regret Minimization (MCCFR) for computing Game Theory Optimal (GTO) strategies in Heads-Up No-Limit Hold'em.

## Philosophy

**Good enough beats perfect.** This solver prioritizes:
- ✅ **Practical exploitability** < 1% pot in seconds (not theoretical perfection in hours)
- ✅ **WASM-ready performance** for browser-based solving
- ✅ **Zero dependencies** for maximum portability and easy compilation
- ✅ **Clear, maintainable code** over micro-optimizations

## What This Is

A CLI tool (and future WASM library) that:
- Solves postflop poker situations using MCCFR
- Encodes positions in a chess FEN-like compact notation
- Outputs exploitability-bounded GTO strategies in seconds
- Compiles to native binary or WebAssembly with `go build`

## Design Approach

**Incremental complexity:** We build the solver street-by-street (river → turn → flop → multi-street), validating correctness at each phase before adding complexity.

**Real-time performance:** Target sub-10s solves for single-street decisions, making GTO analysis practical during actual play preparation and study.

**Full-spectrum solving:** Provide reasonable probability calculations at any game phase—river, turn, flop, or full game trees—with exploitability bounds appropriate to each scenario.

## Performance Targets

### Grandmaster-Level Solving
With **100k MCCFR iterations** on a single postflop street:
- **Exploitability:** <0.5% of pot (undetectable by humans)
- **Native binary:** ~5-10 seconds on modern CPU
- **WASM in browser:** ~30-60 seconds (acceptable for learning tools)

### Complexity Tradeoffs
| Scenario | Iterations Needed | Native Time | WASM Time |
|----------|------------------|-------------|-----------|
| Single street (flop), 2 bet sizes | 10k | <1s | ~5s |
| Single street (turn), 3 bet sizes | 50k | ~5s | ~30s |
| Single street (river), simplified | 100k | ~10s | ~60s |
| Multi-street (flop→turn→river) | 1M+ | minutes | ⚠️ impractical |

**Insight:** Grandmaster-level play doesn't require perfection—just strategies unexploitable in practice.

## Quick Start

```bash
# Solve a position using compact notation
poker-solver solve "BTN:AsKd:S100/BB:??:S100|P3|Th9h2c|>BTN" --iterations 50000

# Output strategy as JSON
poker-solver strategy output.json

# Analyze specific action frequencies
poker-solver range --player BTN --action bet_66
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

### v0.1 - Single Street Solver (Current)
- [x] Position notation parser
- [ ] Card evaluation (hand strength)
- [ ] Game tree builder (single street)
- [ ] Vanilla CFR implementation
- [ ] CLI with solve/output commands

### v0.2 - Monte Carlo Optimization
- [ ] MCCFR with outcome sampling
- [ ] Exploitability calculation
- [ ] Strategy serialization (JSON)
- [ ] Performance benchmarks

### v0.3 - WASM Export
- [ ] JavaScript bindings
- [ ] Browser-compatible build
- [ ] Progress streaming
- [ ] Web worker support

### v0.4 - Production Ready
- [ ] Card abstraction (bucketing)
- [ ] Bet abstraction (geometric sizing)
- [ ] Multi-street solving (optional)
- [ ] Range vs range analysis

## Technical Stack

- **Language:** Go 1.21+ (pure stdlib, zero dependencies)
- **Algorithm:** Monte Carlo CFR (MCCFR) with outcome sampling
- **Target:** Native binary + WASM (`GOOS=js GOARCH=wasm`)
- **Input/Output:** JSON + compact notation parsing

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
