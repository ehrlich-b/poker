# Poker Solver - Web Demo

Browser-based poker solver powered by WebAssembly.

## Running the Demo

The WASM demo requires a web server (browsers block WASM loading from `file://` URLs due to CORS).

### Quick Start

```bash
# From the project root:
make wasm

# Then serve the web directory:
cd web
python3 -m http.server 8080
# Or: npx serve
# Or: php -S localhost:8080
```

Open http://localhost:8080 in your browser.

## Features

- **WebAssembly Performance**: Near-native speed poker solving in the browser
- **Web Workers**: Non-blocking UI - solve runs in background thread
- **Progress Tracking**: Real-time progress updates during solve
- **Multiple Examples**: River, turn, and range-vs-range scenarios

## Usage

1. Enter a position in Poker FEN notation:
   - `BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d2s|>BTN` (river)
   - `BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d|>BTN` (turn)
   - `BTN:AA,KK:S100/BB:QQ,JJ:S100|P10|Th9h2c5d8s|>BTN` (range vs range)

2. Set iteration count (more iterations = better convergence)
   - River: 5,000-10,000 iterations
   - Turn: 2,000-5,000 iterations (MCCFR is more efficient)
   - Range vs range: 3,000-7,000 iterations

3. Click "Solve" and watch progress

4. View equilibrium strategies in JSON format

## API

The WASM module exposes a JavaScript API:

```javascript
// Solve a position
const result = await pokerSolver.solve(
    position,    // Position string
    iterations,  // Number of CFR/MCCFR iterations
    onProgress   // Optional progress callback
);

// Parse a position (validate notation)
const parsed = pokerSolver.parsePosition(position);
```

## Performance

Typical solve times (on modern hardware):

- **River** (combo vs combo): ~100-200ms for 5,000 iterations
- **Turn** (with rollout): ~500ms-1s for 3,000 iterations
- **Range vs range**: ~2-4s for 5,000 iterations (144 combo pairs)

WASM performance is typically 2-3x slower than native, but still very usable for interactive solving.

## Browser Support

Requires a modern browser with WebAssembly support:
- Chrome/Edge 57+
- Firefox 52+
- Safari 11+

## Building

```bash
# Build WASM binary
make wasm

# Files are output to web/:
# - poker-solver.wasm (3.2MB)
# - wasm_exec.js (Go WASM runtime)
# - index.html (demo page)
# - solver-client.js (client API)
# - solver-worker.js (Web Worker)
```

## Architecture

```
Browser Main Thread          Web Worker Thread
─────────────────────        ─────────────────
   index.html
       │
       ├─ solver-client.js ──→ solver-worker.js
       │                            │
       │                            ├─ wasm_exec.js
       │                            └─ poker-solver.wasm
       │                                    │
       │                                    └─ Go solver code
       │
       └─ Progress updates ←────────────────┘
          Strategy results
```

## Limitations

- Maximum 100,000 iterations (safety limit)
- Only 2-player games supported
- Flop solving requires card abstraction (use --buckets flag in native CLI, WASM support coming soon)

## Future Enhancements

- Strategy visualization (charts/tables)
- Exploitability metrics in UI
- Flop solving support in WASM (currently available in native CLI only)
