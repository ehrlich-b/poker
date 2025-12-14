// Poker Solver Client
// Manages communication with the Web Worker

class PokerSolverClient {
    constructor() {
        this.worker = null;
        this.ready = false;
        this.requestId = 0;
        this.pendingRequests = new Map();
        this.currentSolveId = null;

        this.init();
    }

    init() {
        this.worker = new Worker('solver-worker.js');

        this.worker.onmessage = (e) => {
            this.handleMessage(e.data);
        };

        this.worker.onerror = (error) => {
            this.updateStatus(`Worker error: ${error.message}`, 'error');
        };

        // Initialize WASM
        this.worker.postMessage({ type: 'init' });
    }

    handleMessage(data) {
        const { type, id, result, progress, error } = data;

        switch (type) {
            case 'ready':
                this.ready = true;
                this.updateStatus('Solver ready!', 'success');
                this.enableSolveButton();
                break;

            case 'progress':
                this.handleProgress(progress);
                break;

            case 'result':
                this.handleResult(id, result);
                break;

            case 'canceled':
                this.handleCanceled(id);
                break;

            case 'error':
                this.handleError(id, error);
                break;
        }
    }

    handleProgress(progress) {
        const { iteration, total, percent } = progress;
        this.updateProgress(percent, `${iteration}/${total}`);
    }

    handleResult(id, result) {
        const request = this.pendingRequests.get(id);
        if (request) {
            request.resolve(result);
            this.pendingRequests.delete(id);
        }

        this.currentSolveId = null;
        this.hideProgress();
        this.showResult(result);
    }

    handleCanceled(id) {
        if (id && this.pendingRequests.has(id)) {
            const req = this.pendingRequests.get(id);
            req.reject(new Error('Solve canceled'));
            this.pendingRequests.delete(id);
        }
        this.currentSolveId = null;
        this.hideProgress();
        this.updateStatus('Solve canceled', 'info');
    }

    handleError(id, error) {
        const request = this.pendingRequests.get(id);
        if (request) {
            request.reject(new Error(error));
            this.pendingRequests.delete(id);
        }

        this.updateStatus(error, 'error');
        this.currentSolveId = null;
        this.hideProgress();
    }

    solve(position, iterations) {
        return new Promise((resolve, reject) => {
            const id = ++this.requestId;

            this.pendingRequests.set(id, { resolve, reject });
            this.currentSolveId = id;

            this.worker.postMessage({
                type: 'solve',
                id,
                data: { position, iterations }
            });

            this.showProgress();
            this.updateStatus(`Solving ${position.substring(0, 50)}...`, 'info');
        });
    }

    cancel() {
        if (!this.currentSolveId) {
            return;
        }

        this.worker.postMessage({
            type: 'cancel',
            id: this.currentSolveId
        });
        this.updateStatus('Canceling...', 'info');
    }

    parsePosition(position) {
        return new Promise((resolve, reject) => {
            const id = ++this.requestId;

            this.pendingRequests.set(id, { resolve, reject });

            this.worker.postMessage({
                type: 'parsePosition',
                id,
                data: { position }
            });
        });
    }

    // UI Methods
    updateStatus(message, type = 'info') {
        const statusBar = document.getElementById('statusBar');
        statusBar.textContent = message;
        statusBar.className = `status ${type}`;
    }

    showProgress() {
        document.getElementById('progressContainer').classList.remove('hidden');
        document.getElementById('solveBtn').disabled = true;
        const cancelBtn = document.getElementById('cancelBtn');
        if (cancelBtn) {
            cancelBtn.disabled = false;
        }
    }

    hideProgress() {
        document.getElementById('progressContainer').classList.add('hidden');
        document.getElementById('solveBtn').disabled = false;
        const cancelBtn = document.getElementById('cancelBtn');
        if (cancelBtn) {
            cancelBtn.disabled = true;
        }
        this.updateProgress(0, '');
    }

    updateProgress(percent, label) {
        const progressBar = document.getElementById('progressBar');
        progressBar.style.width = `${percent}%`;
        progressBar.textContent = label || `${Math.round(percent)}%`;
    }

    showResult(result) {
        const container = document.getElementById('resultContainer');
        const output = document.getElementById('resultOutput');

        container.classList.remove('hidden');
        // Prefer raw JSON string payload for speed if provided
        if (result && result.strategyJSON) {
            const parsed = {
                infoSets: result.infoSets,
                position: result.position,
            };
            output.textContent = JSON.stringify(parsed, null, 2) + '\n\n' + result.strategyJSON;
        } else {
            output.textContent = JSON.stringify(result, null, 2);
        }

        this.updateStatus(`Solved! Found ${result.infoSets} information sets`, 'success');
    }

    enableSolveButton() {
        const btn = document.getElementById('solveBtn');
        btn.disabled = false;
    }
}

// Initialize client
const client = new PokerSolverClient();

// Set up solve button
document.getElementById('solveBtn').addEventListener('click', async () => {
    const position = document.getElementById('positionInput').value;
    const iterations = parseInt(document.getElementById('iterationsInput').value);

    if (!position) {
        client.updateStatus('Please enter a position', 'error');
        return;
    }

    try {
        await client.solve(position, iterations);
    } catch (error) {
        client.updateStatus(`Error: ${error.message}`, 'error');
    }
});
