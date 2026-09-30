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

func (a *Algorithm) Run() (engine.Trace, error) {
	traceStart := time.Now()
	curve := ecdh.P256()

	var steps []engine.Step
	record := func(id, title, actor, what, why, math string, params, inputs, outputs []engine.Value, dur time.Duration) {
		steps = append(steps, engine.Step{
			ID: id, Title: title, Actor: actor, What: what, Why: why, Math: math,
			Params: params, Inputs: inputs, Outputs: outputs, DurationUs: dur.Microseconds(),
		})
	}

	record("params", "Select curve parameters", "",
		"Pick the elliptic curve group every key and operation in this run belongs to.",
		"All later scalar multiplications happen on this curve; its security level and generator point G are fixed by the standard.",
		"E: y² = x³ − 3x + b (mod p), base point G, order n",
		[]engine.Value{textVal("Curve", "NIST P-256 (secp256r1)", "")},
		nil, nil, 0,
	)

	start := time.Now()
	privA, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return engine.Trace{}, err
	}
	record("alice-keygen", "Alice generates her private key", "alice",
		"Draw a random scalar a from [1, n−1] using a cryptographically secure RNG.",
		"The private key must be unpredictable; it is the only secret Alice needs to protect for the exchange to stay secure.",
		"a ←$ [1, n−1]",
		nil, nil,
		[]engine.Value{hexVal("Private key a", privA.Bytes(), "Never shown in a real system — displayed here for teaching only.")},
		time.Since(start),
	)

	start = time.Now()
	pubA := privA.PublicKey()
	record("alice-pubkey", "Alice derives her public key", "alice",
		"Multiply the base point G by her private scalar a.",
		"Anyone can see A; recovering a from A means solving the elliptic-curve discrete-log problem, believed to be computationally infeasible.",
		"A = a·G",
		nil,
		[]engine.Value{hexVal("Private key a", privA.Bytes(), "")},
		[]engine.Value{hexVal("Public key A", pubA.Bytes(), "")},
		time.Since(start),
	)

	start = time.Now()
	privB, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return engine.Trace{}, err
	}
	record("bob-keygen", "Bob generates his private key", "bob",
		"Draw a random scalar b from [1, n−1] using a cryptographically secure RNG.",
		"Same requirement as Alice's key: unpredictable and kept secret.",
		"b ←$ [1, n−1]",
		nil, nil,
		[]engine.Value{hexVal("Private key b", privB.Bytes(), "Never shown in a real system — displayed here for teaching only.")},
		time.Since(start),
	)

	start = time.Now()
	pubB := privB.PublicKey()
	record("bob-pubkey", "Bob derives his public key", "bob",
		"Multiply the base point G by his private scalar b.",
		"Public and safe to broadcast; only the discrete-log-hard direction is protected.",
		"B = b·G",
		nil,
		[]engine.Value{hexVal("Private key b", privB.Bytes(), "")},
		[]engine.Value{hexVal("Public key B", pubB.Bytes(), "")},
		time.Since(start),
	)

	record("exchange", "Exchange public keys", "",
		"Alice sends A to Bob; Bob sends B to Alice, over a channel that need not be secret.",
		"Only public keys cross the wire — an eavesdropper who records both A and B still cannot compute a·B or b·A without knowing a or b.",
		"Alice → B, Bob → A",
		nil,
		[]engine.Value{hexVal("Public key A", pubA.Bytes(), ""), hexVal("Public key B", pubB.Bytes(), "")},
		[]engine.Value{hexVal("Public key A (received by Bob)", pubA.Bytes(), ""), hexVal("Public key B (received by Alice)", pubB.Bytes(), "")},
		0,
	)

	start = time.Now()
	sharedA, err := privA.ECDH(pubB)
	if err != nil {
		return engine.Trace{}, err
	}
	record("alice-shared", "Alice computes the shared secret", "alice",
		"Multiply Bob's public key by her own private scalar.",
		"a·B = a·(b·G) = (ab)·G — the same curve point Bob will reach from the other direction.",
		"S = a·B",
		nil,
		[]engine.Value{hexVal("Private key a", privA.Bytes(), ""), hexVal("Public key B", pubB.Bytes(), "")},
		[]engine.Value{hexVal("Shared secret (Alice's view)", sharedA, "")},
		time.Since(start),
	)

	start = time.Now()
	sharedB, err := privB.ECDH(pubA)
	if err != nil {
		return engine.Trace{}, err
	}
	record("bob-shared", "Bob computes the shared secret", "bob",
		"Multiply Alice's public key by his own private scalar.",
		"b·A = b·(a·G) = (ab)·G — algebraically identical to Alice's result.",
		"S = b·A",
		nil,
		[]engine.Value{hexVal("Private key b", privB.Bytes(), ""), hexVal("Public key A", pubA.Bytes(), "")},
		[]engine.Value{hexVal("Shared secret (Bob's view)", sharedB, "")},
		time.Since(start),
	)

	match := hex.EncodeToString(sharedA) == hex.EncodeToString(sharedB)
	note := "Confirmed: both sides derived the identical secret without ever transmitting it."
	if !match {
		note = "Mismatch — this should never happen and indicates a bug in the exchange."
	}
	record("verify", "Verify both sides agree", "",
		"Compare Alice's and Bob's computed secrets.",
		"This confirms the algebraic identity a·B = b·A held for this run; a real protocol never compares secrets directly, it feeds the result straight into a KDF.",
		"a·B =? b·A",
		nil,
		[]engine.Value{hexVal("Shared secret (Alice)", sharedA, ""), hexVal("Shared secret (Bob)", sharedB, "")},
		[]engine.Value{textVal("Match", fmt.Sprintf("%v", match), note)},
		0,
	)

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
