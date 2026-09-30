export type ValueKind = 'hex' | 'text' | 'number';

export interface CryptoValue {
  label: string;
  kind: ValueKind;
  hex?: string;
  text?: string;
  note?: string;
}

export interface MathTerm {
  symbol: string;
  meaning: string;
}

export interface Step {
  id: string;
  title: string;
  actor?: 'alice' | 'bob' | '';
  what: string;
  why: string;
  math?: string;
  mathLegend?: MathTerm[];
  params?: CryptoValue[];
  inputs?: CryptoValue[];
  outputs?: CryptoValue[];
  durationUs: number;
}

export interface Trace {
  algorithmId: string;
  title: string;
  steps: Step[];
  totalDurationUs: number;
}

export interface AlgorithmSummary {
  id: string;
  title: string;
  description: string;
}
