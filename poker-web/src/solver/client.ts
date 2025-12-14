/**
 * Poker Solver Client
 * Manages communication with the Web Worker that runs the WASM solver
 */

import type {
  Progress,
  SolverResult,
  PositionInfo,
  WorkerIncomingMessage,
  WorkerOutgoingMessage,
  SolveRequest,
  ParsePositionRequest,
} from '../types/solver';

/**
 * Pending request tracking
 */
interface PendingRequest {
  resolve: (value: SolverResult | PositionInfo) => void;
  reject: (error: Error) => void;
}

/**
 * Poker Solver Client
 * Provides a clean async API for solving poker positions via WASM
 */
export class PokerSolverClient {
  private worker: Worker | null = null;
  private requestId = 0;
  private pendingRequests = new Map<number, PendingRequest>();

  constructor() {
    this.init();
  }

  /**
   * Initialize the Web Worker and WASM module
   */
  private init(): void {
    this.worker = new Worker(
      new URL('./worker.ts', import.meta.url)
      // Note: Vite will auto-detect and bundle as a classic worker
    );

    this.worker.onmessage = (e: MessageEvent<WorkerIncomingMessage>) => {
      this.handleMessage(e.data);
    };

    this.worker.onerror = (error: ErrorEvent) => {
      this.updateStatus(`Worker error: ${error.message}`, 'error');
    };

    // Initialize WASM
    this.worker.postMessage({ type: 'init' } as WorkerOutgoingMessage);
  }

  /**
   * Handle messages from the Web Worker
   */
  private handleMessage(data: WorkerIncomingMessage): void {
    const { type } = data;

    switch (type) {
      case 'ready':
        this.updateStatus('Solver ready!', 'success');
        this.enableSolveButton();
        break;

      case 'progress':
        this.handleProgress(data.progress);
        break;

      case 'result':
        this.handleResult(data.id, data.result);
        break;

      case 'error':
        this.handleError(data.id, data.error);
        break;
    }
  }

  /**
   * Handle progress updates
   */
  private handleProgress(progress: Progress): void {
    const { iteration, total, percent } = progress;
    this.updateProgress(percent, `${iteration}/${total}`);
  }

  /**
   * Handle successful result
   */
  private handleResult(id: number, result: SolverResult | PositionInfo): void {
    const request = this.pendingRequests.get(id);
    if (request) {
      request.resolve(result);
      this.pendingRequests.delete(id);
    }

    this.hideProgress();
    if (this.isSolverResult(result)) {
      this.showResult(result);
    }
  }

  /**
   * Type guard for SolverResult
   */
  private isSolverResult(result: SolverResult | PositionInfo): result is SolverResult {
    return 'infoSets' in result && 'strategies' in result;
  }

  /**
   * Handle error
   */
  private handleError(id: number | undefined, error: string): void {
    if (id !== undefined) {
      const request = this.pendingRequests.get(id);
      if (request) {
        request.reject(new Error(error));
        this.pendingRequests.delete(id);
      }
    }

    this.updateStatus(error, 'error');
    this.hideProgress();
  }

  /**
   * Solve a poker position
   * @param position - Position in FEN notation
   * @param iterations - Number of CFR/MCCFR iterations
   * @returns Promise resolving to solver result
   */
  solve(position: string, iterations: number): Promise<SolverResult> {
    return new Promise((resolve, reject) => {
      const id = ++this.requestId;

      this.pendingRequests.set(id, {
        resolve: resolve as (value: SolverResult | PositionInfo) => void,
        reject,
      });

      const message: SolveRequest = {
        type: 'solve',
        id,
        data: { position, iterations },
      };

      this.worker?.postMessage(message);

      this.showProgress();
      this.updateStatus(
        `Solving ${position.substring(0, 50)}...`,
        'info'
      );
    });
  }

  /**
   * Parse a position string
   * @param position - Position in FEN notation
   * @returns Promise resolving to position info
   */
  parsePosition(position: string): Promise<PositionInfo> {
    return new Promise((resolve, reject) => {
      const id = ++this.requestId;

      this.pendingRequests.set(id, {
        resolve: resolve as (value: SolverResult | PositionInfo) => void,
        reject,
      });

      const message: ParsePositionRequest = {
        type: 'parsePosition',
        id,
        data: { position },
      };

      this.worker?.postMessage(message);
    });
  }

  // UI Methods

  /**
   * Update status bar
   */
  updateStatus(message: string, type: 'info' | 'success' | 'error' = 'info'): void {
    const statusBar = document.getElementById('statusBar');
    if (statusBar) {
      statusBar.textContent = message;
      statusBar.className = `status ${type}`;
    }
  }

  /**
   * Show progress bar
   */
  showProgress(): void {
    const progressContainer = document.getElementById('progressContainer');
    const solveBtn = document.getElementById('solveBtn') as HTMLButtonElement | null;

    progressContainer?.classList.remove('hidden');
    if (solveBtn) {
      solveBtn.disabled = true;
    }
  }

  /**
   * Hide progress bar
   */
  hideProgress(): void {
    const progressContainer = document.getElementById('progressContainer');
    const solveBtn = document.getElementById('solveBtn') as HTMLButtonElement | null;

    progressContainer?.classList.add('hidden');
    if (solveBtn) {
      solveBtn.disabled = false;
    }
    this.updateProgress(0, '');
  }

  /**
   * Update progress bar
   */
  updateProgress(percent: number, label: string): void {
    const progressBar = document.getElementById('progressBar');
    if (progressBar) {
      progressBar.style.width = `${percent}%`;
      progressBar.textContent = label || `${Math.round(percent)}%`;
    }
  }

  /**
   * Show solver result
   */
  showResult(result: SolverResult): void {
    const container = document.getElementById('resultContainer');
    const output = document.getElementById('resultOutput');

    if (container && output) {
      container.classList.remove('hidden');
      output.textContent = JSON.stringify(result, null, 2);

      this.updateStatus(
        `Solved! Found ${result.infoSets} information sets`,
        'success'
      );
    }
  }

  /**
   * Enable solve button
   */
  enableSolveButton(): void {
    const btn = document.getElementById('solveBtn') as HTMLButtonElement | null;
    if (btn) {
      btn.disabled = false;
    }
  }
}
