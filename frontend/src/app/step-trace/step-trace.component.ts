import { CommonModule } from '@angular/common';
import { Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CryptoApiService } from '../services/crypto-api.service';
import { AlgorithmSummary, Trace } from '../models/trace.model';

@Component({
  selector: 'app-step-trace',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './step-trace.component.html',
  styleUrl: './step-trace.component.css',
})
export class StepTraceComponent {
  private readonly api = inject(CryptoApiService);

  readonly algorithms = signal<AlgorithmSummary[]>([]);
  readonly selectedAlgorithmId = signal<string | null>(null);
  readonly trace = signal<Trace | null>(null);
  readonly selectedStepId = signal<string | null>(null);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  readonly selectedStep = computed(() => {
    const t = this.trace();
    const id = this.selectedStepId();
    return t?.steps.find((s) => s.id === id) ?? null;
  });

  readonly selectedAlgorithm = computed(() =>
    this.algorithms().find((a) => a.id === this.selectedAlgorithmId()) ?? null,
  );

  constructor() {
    this.api.listAlgorithms().subscribe({
      next: (list) => {
        this.algorithms.set(list);
        if (list.length > 0) {
          this.selectedAlgorithmId.set(list[0].id);
        }
      },
      error: () => this.error.set('Could not reach the crypto engine backend on :8081.'),
    });
  }

  selectAlgorithm(id: string): void {
    this.selectedAlgorithmId.set(id);
    this.trace.set(null);
    this.selectedStepId.set(null);
  }

  run(): void {
    const id = this.selectedAlgorithmId();
    if (!id) {
      return;
    }
    this.loading.set(true);
    this.error.set(null);
    this.api.runAlgorithm(id).subscribe({
      next: (trace) => {
        this.trace.set(trace);
        this.selectedStepId.set(trace.steps[0]?.id ?? null);
        this.loading.set(false);
      },
      error: () => {
        this.error.set('The algorithm run failed. Is the Go backend running on :8081?');
        this.loading.set(false);
      },
    });
  }

  selectStep(id: string): void {
    this.selectedStepId.set(id);
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
