// Package ecdh wraps Go's standard library crypto/ecdh implementation of the
// ECDH key exchange, recording every stage as an engine.Step so the frontend
// can walk Alice and Bob through it one operation at a time.
package ecdh

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"crypto-lab/internal/engine"
)

type Algorithm struct{}

func New() *Algorithm { return &Algorithm{} }

func (a *Algorithm) ID() string    { return "ecdh" }
func (a *Algorithm) Title() string { return "ECDH Key Exchange (P-256)" }
func (a *Algorithm) Description() string {
	return "Alice and Bob each generate an elliptic-curve key pair, exchange public keys over an open channel, and independently derive the same shared secret without ever transmitting it."
}

func hexVal(label string, b []byte, note string) engine.Value {
	return engine.Value{Label: label, Kind: engine.KindHex, Hex: hex.EncodeToString(b), Note: note}
}

func textVal(label, text, note string) engine.Value {
	return engine.Value{Label: label, Kind: engine.KindText, Text: text, Note: note}
}

func term(symbol, meaning string) engine.MathTerm {
	return engine.MathTerm{Symbol: symbol, Meaning: meaning}
}

// stepInput is every field of an engine.Step, gathered before recording so
// the call site reads as a labeled record instead of a wall of positional
// arguments.
type stepInput struct {
	ID, Title, Actor, What, Why, Math string
	MathLegend                        []engine.MathTerm
	Params, Inputs, Outputs           []engine.Value
	Dur                               time.Duration
}

func (a *Algorithm) Run() (engine.Trace, error) {
	traceStart := time.Now()
	curve := ecdh.P256()

	var steps []engine.Step
	record := func(s stepInput) {
		steps = append(steps, engine.Step{
			ID: s.ID, Title: s.Title, Actor: s.Actor, What: s.What, Why: s.Why,
			Math: s.Math, MathLegend: s.MathLegend,
			Params: s.Params, Inputs: s.Inputs, Outputs: s.Outputs,
			DurationUs: s.Dur.Microseconds(),
		})
	}

	record(stepInput{
		ID: "params", Title: "Select curve parameters",
		What: "Pick the elliptic curve group every key and operation in this run belongs to.",
		Why:  "All later scalar multiplications happen on this curve; its security level and generator point G are fixed by the standard.",
		Math: "E: y² = x³ − 3x + b (mod p), base point G, order n",
		MathLegend: []engine.MathTerm{
			term("E", "the elliptic curve: the set of points (x, y) satisfying this equation"),
			term("p", "a large prime — all coordinates are computed modulo p"),
			term("b", "the curve coefficient fixed by the P-256 standard"),
			term("G", "the base point (generator): every key is some multiple of G"),
			term("n", "the order of G — how many distinct points G can generate"),
		},
		Params: []engine.Value{textVal("Curve", "NIST P-256 (secp256r1)", "")},
	})

	start := time.Now()
	privA, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return engine.Trace{}, err
	}
	record(stepInput{
		ID: "alice-keygen", Title: "Alice generates her private key", Actor: "alice",
		What: "Draw a random scalar a from [1, n−1] using a cryptographically secure RNG.",
		Why:  "The private key must be unpredictable; it is the only secret Alice needs to protect for the exchange to stay secure.",
		Math: "a ←$ [1, n−1]",
		MathLegend: []engine.MathTerm{
			term("a", "Alice's private key — a random integer"),
			term("←$", "\"sampled uniformly at random from\" (not deterministic, not attacker-predictable)"),
			term("[1, n−1]", "the valid range: any scalar from 1 up to the curve's order minus 1"),
		},
		Outputs: []engine.Value{hexVal("Private key a", privA.Bytes(), "Never shown in a real system — displayed here for teaching only.")},
		Dur:     time.Since(start),
	})

	start = time.Now()
	pubA := privA.PublicKey()
	record(stepInput{
		ID: "alice-pubkey", Title: "Alice derives her public key", Actor: "alice",
		What: "Multiply the base point G by her private scalar a.",
		Why:  "Anyone can see A; recovering a from A means solving the elliptic-curve discrete-log problem, believed to be computationally infeasible.",
		Math: "A = a·G",
		MathLegend: []engine.MathTerm{
			term("A", "Alice's public key — a point on the curve, safe to share"),
			term("a·G", "elliptic-curve scalar multiplication: G added to itself a times (via double-and-add, not regular multiplication)"),
		},
		Inputs:  []engine.Value{hexVal("Private key a", privA.Bytes(), "")},
		Outputs: []engine.Value{hexVal("Public key A", pubA.Bytes(), "")},
		Dur:     time.Since(start),
	})

	start = time.Now()
	privB, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return engine.Trace{}, err
	}
	record(stepInput{
		ID: "bob-keygen", Title: "Bob generates his private key", Actor: "bob",
		What: "Draw a random scalar b from [1, n−1] using a cryptographically secure RNG.",
		Why:  "Same requirement as Alice's key: unpredictable and kept secret.",
		Math: "b ←$ [1, n−1]",
		MathLegend: []engine.MathTerm{
			term("b", "Bob's private key — a random integer"),
			term("←$", "\"sampled uniformly at random from\""),
			term("[1, n−1]", "the valid range: any scalar from 1 up to the curve's order minus 1"),
		},
		Outputs: []engine.Value{hexVal("Private key b", privB.Bytes(), "Never shown in a real system — displayed here for teaching only.")},
		Dur:     time.Since(start),
	})

	start = time.Now()
	pubB := privB.PublicKey()
	record(stepInput{
		ID: "bob-pubkey", Title: "Bob derives his public key", Actor: "bob",
		What: "Multiply the base point G by his private scalar b.",
		Why:  "Public and safe to broadcast; only the discrete-log-hard direction is protected.",
		Math: "B = b·G",
		MathLegend: []engine.MathTerm{
			term("B", "Bob's public key — a point on the curve, safe to share"),
			term("b·G", "elliptic-curve scalar multiplication of the base point by b"),
		},
		Inputs:  []engine.Value{hexVal("Private key b", privB.Bytes(), "")},
		Outputs: []engine.Value{hexVal("Public key B", pubB.Bytes(), "")},
		Dur:     time.Since(start),
	})

	record(stepInput{
		ID: "exchange", Title: "Exchange public keys",
		What: "Alice sends A to Bob; Bob sends B to Alice, over a channel that need not be secret.",
		Why:  "Only public keys cross the wire — an eavesdropper who records both A and B still cannot compute a·B or b·A without knowing a or b.",
		Math: "Alice → B, Bob → A",
		MathLegend: []engine.MathTerm{
			term("→", "sent over the network — this direction carries no secret"),
		},
		Inputs:  []engine.Value{hexVal("Public key A", pubA.Bytes(), ""), hexVal("Public key B", pubB.Bytes(), "")},
		Outputs: []engine.Value{hexVal("Public key A (received by Bob)", pubA.Bytes(), ""), hexVal("Public key B (received by Alice)", pubB.Bytes(), "")},
	})

	start = time.Now()
	sharedA, err := privA.ECDH(pubB)
	if err != nil {
		return engine.Trace{}, err
	}
	record(stepInput{
		ID: "alice-shared", Title: "Alice computes the shared secret", Actor: "alice",
		What: "Multiply Bob's public key by her own private scalar.",
		Why:  "a·B = a·(b·G) = (ab)·G — the same curve point Bob will reach from the other direction.",
		Math: "S = a·B",
		MathLegend: []engine.MathTerm{
			term("S", "the shared secret point — its x-coordinate becomes the raw shared secret"),
			term("a·B", "Alice's private scalar times Bob's public point"),
		},
		Inputs:  []engine.Value{hexVal("Private key a", privA.Bytes(), ""), hexVal("Public key B", pubB.Bytes(), "")},
		Outputs: []engine.Value{hexVal("Shared secret (Alice's view)", sharedA, "")},
		Dur:     time.Since(start),
	})

	start = time.Now()
	sharedB, err := privB.ECDH(pubA)
	if err != nil {
		return engine.Trace{}, err
	}
	record(stepInput{
		ID: "bob-shared", Title: "Bob computes the shared secret", Actor: "bob",
		What: "Multiply Alice's public key by his own private scalar.",
		Why:  "b·A = b·(a·G) = (ab)·G — algebraically identical to Alice's result.",
		Math: "S = b·A",
		MathLegend: []engine.MathTerm{
			term("S", "the shared secret point — its x-coordinate becomes the raw shared secret"),
			term("b·A", "Bob's private scalar times Alice's public point"),
		},
		Inputs:  []engine.Value{hexVal("Private key b", privB.Bytes(), ""), hexVal("Public key A", pubA.Bytes(), "")},
		Outputs: []engine.Value{hexVal("Shared secret (Bob's view)", sharedB, "")},
		Dur:     time.Since(start),
	})

	match := hex.EncodeToString(sharedA) == hex.EncodeToString(sharedB)
	note := "Confirmed: both sides derived the identical secret without ever transmitting it."
	if !match {
		note = "Mismatch — this should never happen and indicates a bug in the exchange."
	}
	record(stepInput{
		ID: "verify", Title: "Verify both sides agree",
		What: "Compare Alice's and Bob's computed secrets.",
		Why:  "This confirms the algebraic identity a·B = b·A held for this run; a real protocol never compares secrets directly, it feeds the result straight into a KDF.",
		Math: "a·B =? b·A",
		MathLegend: []engine.MathTerm{
			term("=?", "checked for equality — this comparison is for this demo only, not part of the real protocol"),
		},
		Inputs:  []engine.Value{hexVal("Shared secret (Alice)", sharedA, ""), hexVal("Shared secret (Bob)", sharedB, "")},
		Outputs: []engine.Value{textVal("Match", fmt.Sprintf("%v", match), note)},
	})

	if !match {
		return engine.Trace{}, errors.New("shared secrets did not match")
	}

	return engine.Trace{
		AlgorithmID:     a.ID(),
		Title:           a.Title(),
		Steps:           steps,
		TotalDurationUs: time.Since(traceStart).Microseconds(),
	}, nil
}
