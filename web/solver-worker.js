// Poker Solver Web Worker
// Runs the WASM solver in a background thread to avoid freezing the UI

importScripts('wasm_exec.js');

let solverReady = false;
let go = null;

// Initialize WASM
async function initWasm() {
    try {
        go = new Go();
        const result = await WebAssembly.instantiateStreaming(
            fetch('poker-solver.wasm'),
            go.importObject
        );

        // Run the Go program
        go.run(result.instance);

        // Wait a bit for the Go program to register its functions
        await new Promise(resolve => setTimeout(resolve, 100));

        solverReady = true;
        postMessage({ type: 'ready' });
    } catch (error) {
        postMessage({
            type: 'error',
            error: `Failed to initialize WASM: ${error.message}`
        });
    }
}

// Handle messages from main thread
self.onmessage = async function(e) {
    const { type, id, data } = e.data;

    if (type === 'init') {
        await initWasm();
        return;
    }

    if (!solverReady) {
        postMessage({
            type: 'error',
            id,
            error: 'Solver not ready. Call init first.'
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

            case 'cancel':
                handleCancel(id);
                break;

            default:
                postMessage({
                    type: 'error',
                    id,
                    error: `Unknown message type: ${type}`
                });
        }
    } catch (error) {
        postMessage({
            type: 'error',
            id,
            error: error.message || String(error)
        });
    }
};

// Handle solve request
async function handleSolve(id, data) {
    const { position, iterations } = data;

    // Progress callback
    const onProgress = (progress) => {
        postMessage({
            type: 'progress',
            id,
            progress
        });
    };

    try {
        // Call WASM solver
        const result = await pokerSolver.solve(position, iterations, onProgress);

        postMessage({
            type: 'result',
            id,
            result
        });
    } catch (error) {
        postMessage({
            type: 'error',
            id,
            error: error.message || String(error)
        });
    }
}

// Handle parse position request
function handleParsePosition(id, data) {
    const { position } = data;

    try {
        const result = pokerSolver.parsePosition(position);

        if (result.error) {
            throw new Error(result.error);
        }

        postMessage({
            type: 'result',
            id,
            result
        });
    } catch (error) {
        postMessage({
            type: 'error',
            id,
            error: error.message || String(error)
        });
    }
}

// Handle cancellation request
function handleCancel(id) {
    try {
        if (typeof pokerSolver.cancel === 'function') {
            pokerSolver.cancel();
        }
        postMessage({
            type: 'canceled',
            id
        });
    } catch (error) {
        postMessage({
            type: 'error',
            id,
            error: error.message || String(error)
        });
    }
}
