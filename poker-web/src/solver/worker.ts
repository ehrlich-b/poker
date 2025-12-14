/**
 * Poker Solver Web Worker
 * Runs the WASM solver in a background thread to avoid freezing the UI
 */

// Import wasm_exec.js for Go WASM runtime
// This must be loaded before the WASM module
self.importScripts('/wasm_exec.js');

// TypeScript types (these will be stripped in the build)
type Progress = {
  iteration: number;
  total: number;
  percent: number;
};

type SolverResult = {
  infoSets: number;
  strategies: Record<string, any>;
  position?: string;
  iterations?: number;
};

type PositionInfo = {
  board: string[];
  pot: number;
  players: { position: string; stack: number; range?: string }[];
  error?: string;
};

type PokerSolver = {
  solve(
    position: string,
    iterations: number,
    onProgress?: (progress: Progress) => void
  ): Promise<SolverResult>;
  parsePosition(position: string): PositionInfo;
};

// Global variables for WASM
let solverReady = false;
let go: any = null;

/**
 * Access to the WASM-exported solver
 * This will be set by the Go WASM module after initialization
 */
declare const pokerSolver: PokerSolver;

/**
 * Go WASM runtime class
 * Provided by wasm_exec.js
 */
declare class Go {
  importObject: any;
  run(instance: WebAssembly.Instance): void;
}

/**
 * Initialize the WASM module
 */
async function initWasm(): Promise<void> {
  try {
    go = new Go();

    const result = await WebAssembly.instantiateStreaming(
      fetch('/poker-solver.wasm'),
      go.importObject
    );

    // Run the Go program
    // This will register the pokerSolver object
    go.run(result.instance);

    // Wait a bit for the Go program to register its functions
    await new Promise((resolve) => setTimeout(resolve, 100));

    solverReady = true;
    postMessage({ type: 'ready' });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    postMessage({
      type: 'error',
      error: `Failed to initialize WASM: ${errorMessage}`,
    });
  }
}

/**
 * Handle solve request
 */
async function handleSolve(id: number, data: { position: string; iterations: number }): Promise<void> {
  const { position, iterations } = data;

  // Progress callback
  const onProgress = (progress: Progress) => {
    postMessage({
      type: 'progress',
      id,
      progress,
    });
  };

  try {
    // Call WASM solver
    const result = await pokerSolver.solve(position, iterations, onProgress);

    postMessage({
      type: 'result',
      id,
      result,
    });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    postMessage({
      type: 'error',
      id,
      error: errorMessage,
    });
  }
}

/**
 * Handle parse position request
 */
function handleParsePosition(id: number, data: { position: string }): void {
  const { position } = data;

  try {
    const result = pokerSolver.parsePosition(position);

    if (result.error) {
      throw new Error(result.error);
    }

    postMessage({
      type: 'result',
      id,
      result,
    });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    postMessage({
      type: 'error',
      id,
      error: errorMessage,
    });
  }
}

/**
 * Handle messages from the main thread
 */
self.onmessage = async function (e: MessageEvent) {
  const { type, id, data } = e.data;

  if (type === 'init') {
    await initWasm();
    return;
  }

  if (!solverReady) {
    postMessage({
      type: 'error',
      id,
      error: 'Solver not ready. Call init first.',
    });
    return;
  }

  try {
    switch (type) {
      case 'solve':
        await handleSolve(id, data);
        break;

      case 'parsePosition':
        handleParsePosition(id, data);
        break;

      default:
        postMessage({
          type: 'error',
          id,
          error: `Unknown message type: ${type}`,
        });
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    postMessage({
      type: 'error',
      id,
      error: errorMessage,
    });
  }
};
