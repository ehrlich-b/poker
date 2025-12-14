# Poker Solver – Lightweight WASM Roadmap

## Red Alerts (fix before shipping)
- WASM flop routing is wrong: flop terminals are rollout nodes but WASM uses vanilla CFR (zero-valued payoffs). Force MCCFR for flop/turn and for river with ranges.
- River range-vs-range in WASM runs full CFR over all combo pairs; this blows up in browser. Use chance-sampling MCCFR on river when ranges exist.
- No cancel path exposed to JS; long solves cannot be aborted -> risk of jank/thermal runaway on phones. Add cancel + wall-time/iteration clamps.
- Equity/potential bucketing enumerates every runout; on WASM this will beachball. Replace with Monte Carlo sampling + memo cache.
- Action abstraction for web still uses 0.5/0.75/1.5 pot (+ all-in). Too many actions for mobile; default to 1 size per street (geometric) and no raises facing bets.
- Strategy JSON is encode→decode in WASM path; drop decode and return raw JSON string (parse in UI if needed).

## Lightweight “Fuzzy” Solver Plan (mobile-first)
- Algorithm: MCCFR outcome sampling everywhere. Deterministic seed. Early-stop on diminishing regret (or fixed small iteration caps: river 2–5k, turn 10–20k, flop 30–50k).
- Actions: 1 geometric bet size per street (+check/call/fold, +all-in optional). Facing bets: call/fold only by default for web; allow opt-in raises.
- Abstraction: Bucketing on flop/turn (e.g., 50 flop buckets, 20 turn buckets; river by equity percentile if needed). Reuse bucket cache.
- Equity/Potential: Monte Carlo sampler with memoization keyed by (board, hero hand/bucket, opp range hash). Fixed sample count for predictability.
- Safety: Expose cancelSolve to JS worker; enforce hard iteration cap and wall-time budget per solve. Worker-only execution.
- Output: Return raw JSON string; offer compact summary shape for UI (counts, infoset total).

## Implementation Tasks
1) WASM Entry
   - Route flop/turn and river-with-ranges to MCCFR; keep single-combo river on CFR.
   - Add cancel API (worker message -> close channel) and wall-time guard.
   - Swap action config to minimal geometric sizes for web defaults.
   - Return raw JSON string; keep infoSets count separately.
2) Solver Perf
   - MCCFR: avoid per-visit allocations (reuse action/regret buffers; store child key slice on nodes to sample without building slices; avoid map iteration churn).
   - CFR/MCCFR: preallocate scratch arrays if needed; consider pooled RNG.
3) Equity/Bucketing
   - Implement MC equity/potential sampler with memoization.
   - Plug sampler into Bucketer; invalidate cache when board/range changes.
   - Add “fast/slow” sampling presets.
4) Web Defaults
   - Add quality presets (mobile-safe/desktop). Tie to iterations, bet sizes, buckets, and sample counts.
   - Document expected solve times and exploitability bands for presets.
5) Packaging
   - Build stripped WASM (`-ldflags "-s -w"`) and TinyGo variant; pick best size/speed tradeoff.
   - Keep worker-only demo; ensure progress/cancel UI hooks.

## Validation
- Add WASM-side smoke tests: flop/turn route to MCCFR; cancel stops early; iteration cap respected.
- Bench WASM in headless browser: measure solve time for small river and flop presets; ensure no main-thread work.
- Add regression ensuring river range uses sampling, not full-tree CFR.
