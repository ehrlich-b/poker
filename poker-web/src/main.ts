/**
 * Main entry point for Poker Solver web UI
 */

import './styles/main.css';
import { PokerSolverClient } from './solver/client';

/**
 * Example positions for quick testing
 */
interface Example {
  position: string;
  iterations: number;
}

const examples: Example[] = [
  {
    position: 'BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d2s|>BTN',
    iterations: 5000,
  },
  {
    position: 'BTN:AdAc:S100/BB:QdQh:S100|P10|Kh9s4c7d|>BTN',
    iterations: 3000,
  },
  {
    position: 'BTN:AA,KK:S100/BB:QQ,JJ:S100|P10|Th9h2c5d8s|>BTN',
    iterations: 2000,
  },
];

/**
 * Load an example position into the form
 */
function loadExample(index: number): void {
  const example = examples[index];
  const positionInput = document.getElementById('positionInput') as HTMLInputElement;
  const iterationsInput = document.getElementById('iterationsInput') as HTMLInputElement;

  if (positionInput && iterationsInput) {
    positionInput.value = example.position;
    iterationsInput.value = example.iterations.toString();
  }
}

/**
 * Initialize the application
 */
function init(): void {
  // Initialize the solver client
  const client = new PokerSolverClient();

  // Set up solve button click handler
  const solveBtn = document.getElementById('solveBtn');
  solveBtn?.addEventListener('click', async () => {
    const positionInput = document.getElementById('positionInput') as HTMLInputElement;
    const iterationsInput = document.getElementById('iterationsInput') as HTMLInputElement;

    const position = positionInput?.value;
    const iterations = parseInt(iterationsInput?.value || '5000');

    if (!position) {
      client.updateStatus('Please enter a position', 'error');
      return;
    }

    try {
      await client.solve(position, iterations);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      client.updateStatus(`Error: ${errorMessage}`, 'error');
    }
  });

  // Set up example click handlers
  const exampleElements = document.querySelectorAll('.example');
  exampleElements.forEach((element) => {
    element.addEventListener('click', () => {
      const exampleId = element.getAttribute('data-example-id');
      if (exampleId !== null) {
        loadExample(parseInt(exampleId));
      }
    });
  });
}

// Start the app when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}
