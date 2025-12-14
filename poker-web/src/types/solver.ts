/**
 * TypeScript type definitions for the Poker Solver WASM interface
 */

/**
 * Progress update during solving
 */
export interface Progress {
  iteration: number;
  total: number;
  percent: number;
}

/**
 * Individual action strategy with probability
 */
export interface ActionStrategy {
  action: string;
  probability: number;
}

/**
 * Strategy for a specific information set
 */
export interface Strategy {
  infoSet: string;
  actions: ActionStrategy[];
}

/**
 * Complete solver result
 */
export interface SolverResult {
  infoSets: number;
  strategies: Record<string, Strategy>;
  position?: string;
  iterations?: number;
}

/**
 * Position information returned by parser
 */
export interface PositionInfo {
  board: string[];
  pot: number;
  players: {
    position: string;
    stack: number;
    range?: string;
  }[];
  error?: string;
}

/**
 * WASM Poker Solver interface
 * Exposed by the Go WASM module via syscall/js
 */
export interface PokerSolver {
  /**
   * Solve a poker position
   * @param position - Position in FEN notation
   * @param iterations - Number of CFR/MCCFR iterations
   * @param onProgress - Optional progress callback
   * @returns Promise resolving to solver result
   */
  solve(
    position: string,
    iterations: number,
    onProgress?: (progress: Progress) => void
  ): Promise<SolverResult>;

  /**
   * Parse a position string into structured format
   * @param position - Position in FEN notation
   * @returns Position information or error
   */
  parsePosition(position: string): PositionInfo;
}

/**
 * Message types for Web Worker communication
 */
export type WorkerMessageType =
  | 'init'
  | 'solve'
  | 'parsePosition'
  | 'ready'
  | 'progress'
  | 'result'
  | 'error';

/**
 * Base message structure
 */
export interface WorkerMessage {
  type: WorkerMessageType;
  id?: number;
}

/**
 * Request to solve a position
 */
export interface SolveRequest extends WorkerMessage {
  type: 'solve';
  id: number;
  data: {
    position: string;
    iterations: number;
  };
}

/**
 * Request to parse a position
 */
export interface ParsePositionRequest extends WorkerMessage {
  type: 'parsePosition';
  id: number;
  data: {
    position: string;
  };
}

/**
 * Progress update from worker
 */
export interface ProgressMessage extends WorkerMessage {
  type: 'progress';
  id?: number;
  progress: Progress;
}

/**
 * Result from worker
 */
export interface ResultMessage extends WorkerMessage {
  type: 'result';
  id: number;
  result: SolverResult | PositionInfo;
}

/**
 * Error from worker
 */
export interface ErrorMessage extends WorkerMessage {
  type: 'error';
  id?: number;
  error: string;
}

/**
 * Worker ready notification
 */
export interface ReadyMessage extends WorkerMessage {
  type: 'ready';
}

/**
 * Union type of all possible worker messages
 */
export type WorkerIncomingMessage =
  | ReadyMessage
  | ProgressMessage
  | ResultMessage
  | ErrorMessage;

export type WorkerOutgoingMessage =
  | WorkerMessage
  | SolveRequest
  | ParsePositionRequest;
