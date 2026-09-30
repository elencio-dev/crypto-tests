import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { AlgorithmSummary, Trace } from '../models/trace.model';

const API_BASE = 'http://localhost:8081/api';

@Injectable({ providedIn: 'root' })
export class CryptoApiService {
  private readonly http = inject(HttpClient);

  listAlgorithms(): Observable<AlgorithmSummary[]> {
    return this.http.get<AlgorithmSummary[]>(`${API_BASE}/algorithms`);
  }

  runAlgorithm(id: string): Observable<Trace> {
    return this.http.post<Trace>(`${API_BASE}/algorithms/${id}/run`, {});
  }
}
