import { CommonModule } from '@angular/common';
import { Component, DestroyRef, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CryptoApiService } from '../services/crypto-api.service';
import { AlgorithmSummary, Trace } from '../models/trace.model';

const STEP_DELAY_MS = 900;

/**
 * The fixed set of boxes in the ECDH flow diagram. Diagram layout is
 * algorithm-specific (a ChaCha20 diagram will look nothing like this one),
 * so this mapping only applies while the ECDH trace is loaded.
 */
export type EcdhNode =
  | 'alice-priv'
  | 'bob-priv'
  | 'alice-pub'
  | 'bob-pub'
  | 'exchange'
  | 'alice-shared'
  | 'bob-shared'
  | 'shared-secret';

const ECDH_NODE_TO_STEP: Record<EcdhNode, string> = {
  'alice-priv': 'alice-keygen',
  'bob-priv': 'bob-keygen',
  'alice-pub': 'alice-pubkey',
  'bob-pub': 'bob-pubkey',
  exchange: 'exchange',
  'alice-shared': 'alice-shared',
  'bob-shared': 'bob-shared',
  'shared-secret': 'verify',
};

@Component({
  selector: 'app-step-trace',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './step-trace.component.html',
  styleUrl: './step-trace.component.css',
})
export class StepTraceComponent {
  private readonly api = inject(CryptoApiService);
  private timer: ReturnType<typeof setTimeout> | null = null;

  readonly algorithms = signal<AlgorithmSummary[]>([]);
  readonly selectedAlgorithmId = signal<string | null>(null);
  readonly trace = signal<Trace | null>(null);
  readonly selectedStepId = signal<string | null>(null);
  /** How many of the trace's steps have been revealed to the user so far. */
  readonly revealedCount = signal(0);
  readonly playing = signal(false);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  readonly totalSteps = computed(() => this.trace()?.steps.length ?? 0);
  readonly isComplete = computed(() => this.totalSteps() > 0 && this.revealedCount() >= this.totalSteps());

  readonly selectedStep = computed(() => {
    const t = this.trace();
    const id = this.selectedStepId();
    return t?.steps.find((s) => s.id === id) ?? null;
  });

  readonly selectedIndex = computed(() => {
    const t = this.trace();
    const id = this.selectedStepId();
    if (!t || !id) return 0;
    return t.steps.findIndex((s) => s.id === id) + 1;
  });

  readonly canStepBack = computed(() => this.selectedIndex() > 1);

  readonly selectedAlgorithm = computed(() =>
    this.algorithms().find((a) => a.id === this.selectedAlgorithmId()) ?? null,
  );

  readonly showEcdhDiagram = computed(() => this.trace()?.algorithmId === 'ecdh');

  constructor(destroyRef: DestroyRef) {
    this.api.listAlgorithms().subscribe({
      next: (list) => {
        this.algorithms.set(list);
        if (list.length > 0) {
          this.selectedAlgorithmId.set(list[0].id);
        }
      },
      error: () => this.error.set('Could not reach the crypto engine backend on :8081.'),
    });
    destroyRef.onDestroy(() => this.clearTimer());
  }

  selectAlgorithm(id: string): void {
    this.selectedAlgorithmId.set(id);
    this.resetTrace();
  }

  run(): void {
    const id = this.selectedAlgorithmId();
    if (!id) {
      return;
    }
    this.resetTrace();
    this.loading.set(true);
    this.error.set(null);
    this.api.runAlgorithm(id).subscribe({
      next: (trace) => {
        this.trace.set(trace);
        this.loading.set(false);
        // Reveal just the first step and wait for the user to step through
        // the rest with Next (or Play, for hands-off auto-advance).
        this.revealNext();
      },
      error: () => {
        this.error.set('The algorithm run failed. Is the Go backend running on :8081?');
        this.loading.set(false);
      },
    });
  }

  /** Auto-advance: reveal the next unseen step and, if still playing, schedule the one after it. */
  private revealNext(): void {
    const t = this.trace();
    if (!t) return;
    const next = this.revealedCount();
    if (next >= t.steps.length) {
      this.playing.set(false);
      return;
    }
    this.revealedCount.set(next + 1);
    this.selectedStepId.set(t.steps[next].id);
    if (this.playing() && next + 1 < t.steps.length) {
      this.timer = setTimeout(() => this.revealNext(), STEP_DELAY_MS);
    } else {
      this.playing.set(false);
    }
  }

  play(): void {
    if (!this.trace() || this.isComplete()) return;
    this.playing.set(true);
    this.revealNext();
  }

  pause(): void {
    this.playing.set(false);
    this.clearTimer();
  }

  stepForward(): void {
    this.pause();
    this.revealNext();
  }

  stepBack(): void {
    this.pause();
    const t = this.trace();
    if (!t) return;
    const idx = t.steps.findIndex((s) => s.id === this.selectedStepId());
    if (idx > 0) {
      this.selectedStepId.set(t.steps[idx - 1].id);
    }
  }

  showAll(): void {
    this.pause();
    const t = this.trace();
    if (!t) return;
    this.revealedCount.set(t.steps.length);
    if (!this.selectedStepId()) {
      this.selectedStepId.set(t.steps[0].id);
    }
  }

  selectStep(id: string): void {
    const t = this.trace();
    if (!t) return;
    const idx = t.steps.findIndex((s) => s.id === id);
    if (idx === -1 || idx >= this.revealedCount()) return; // not revealed yet
    this.pause();
    this.selectedStepId.set(id);
  }

  private resetTrace(): void {
    this.clearTimer();
    this.trace.set(null);
    this.selectedStepId.set(null);
    this.revealedCount.set(0);
    this.playing.set(false);
  }

  private clearTimer(): void {
    if (this.timer !== null) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }

  nodeState(node: EcdhNode): 'pending' | 'revealed' | 'active' {
    const t = this.trace();
    if (!t) return 'pending';
    const stepId = ECDH_NODE_TO_STEP[node];
    const idx = t.steps.findIndex((s) => s.id === stepId);
    if (idx === -1 || idx >= this.revealedCount()) return 'pending';
    return this.selectedStepId() === stepId ? 'active' : 'revealed';
  }

  selectNode(node: EcdhNode): void {
    this.selectStep(ECDH_NODE_TO_STEP[node]);
  }

  actorLabel(actor: string | undefined): string {
    if (actor === 'alice') return 'Alice';
    if (actor === 'bob') return 'Bob';
    return 'Shared';
  }

  formatDuration(us: number): string {
    if (us < 1000) return `${us} µs`;
    return `${(us / 1000).toFixed(2)} ms`;
  }
}
