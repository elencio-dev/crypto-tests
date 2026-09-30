// Package engine defines the shared contract every cryptographic algorithm
// exposes to the visualization layer: a named list of inspectable Steps,
// each with its own inputs, outputs, math, and rationale.
package engine

// ValueKind tells the frontend how to render a Value.
type ValueKind string

const (
	KindHex    ValueKind = "hex"
	KindText   ValueKind = "text"
	KindNumber ValueKind = "number"
)

// Value is one named piece of data attached to a step, as a parameter,
// input, or output.
type Value struct {
	Label string    `json:"label"`
	Kind  ValueKind `json:"kind"`
	Hex   string    `json:"hex,omitempty"`
	Text  string    `json:"text,omitempty"`
	Note  string    `json:"note,omitempty"`
}

// MathTerm names one symbol used in a step's Math expression and explains
// what it stands for, so the formula reads as more than a string of glyphs.
type MathTerm struct {
	Symbol  string `json:"symbol"`
	Meaning string `json:"meaning"`
}

// Step is one inspectable stage of an algorithm's execution: what happened,
// why, the values that flowed through it, and how long it took.
type Step struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Actor      string     `json:"actor,omitempty"` // "alice", "bob", or "" for a shared/system step
	What       string     `json:"what"`
	Why        string     `json:"why"`
	Math       string     `json:"math,omitempty"`
	MathLegend []MathTerm `json:"mathLegend,omitempty"`
	Params     []Value    `json:"params,omitempty"`
	Inputs     []Value    `json:"inputs,omitempty"`
	Outputs    []Value    `json:"outputs,omitempty"`
	DurationUs int64      `json:"durationUs"`
}

// Trace is the complete step-by-step record produced by running one algorithm.
type Trace struct {
	AlgorithmID     string `json:"algorithmId"`
	Title           string `json:"title"`
	Steps           []Step `json:"steps"`
	TotalDurationUs int64  `json:"totalDurationUs"`
}

// Algorithm is anything the engine can run end-to-end and produce a Trace for.
type Algorithm interface {
	ID() string
	Title() string
	Description() string
	Run() (Trace, error)
}
