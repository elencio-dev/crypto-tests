# Cryptography Visual Lab

An interactive visual laboratory for cryptography. Instead of treating an
algorithm as a black box (`message → encryption → ciphertext`), this lab
opens it up: every algorithm runs as a sequence of inspectable **steps**,
each showing what happened, why, the math behind it (in plain language, not
just symbols), the real inputs/outputs, and how long it actually took to
execute.

The goal is to make cryptography *observable* — click through a key
exchange the way you'd step through a debugger, watch a diagram light up
node by node, and see genuine measured data instead of a diagram copied
from a textbook.

## How it's built

```
Angular (frontend)  →  the microscope: diagrams, step-by-step playback, explanations
Go (backend)         →  the lab engine: runs real cryptographic primitives and
                         records every stage as structured, inspectable data
```

The backend never fakes a value for the sake of the UI. Keys are generated
with `crypto/rand`, curve math runs through Go's standard library
`crypto/ecdh`, and every "Execution time" you see is a real
`time.Since(...)` measurement taken around that exact operation.

```
backend/
  cmd/server/main.go                     entrypoint — registers algorithms, starts the HTTP server
  internal/engine/
    types.go                             the core contract: Step, Trace, MathTerm, Algorithm interface
    registry.go                          maps algorithm IDs to implementations, lists them for the UI
  internal/algorithms/ecdh/ecdh.go       ECDH (P-256), wrapping crypto/ecdh, instrumented step by step
  internal/httpapi/server.go             REST layer: GET /api/algorithms, POST /api/algorithms/{id}/run

frontend/
  src/app/models/trace.model.ts          TypeScript mirror of the backend's JSON contract
  src/app/services/crypto-api.service.ts thin HTTP client for the two endpoints
  src/app/step-trace/                    the whole lab UI: diagram, step list, detail panel, playback controls
```

### The engine contract

Every algorithm implements one interface:

```go
type Algorithm interface {
    ID() string
    Title() string
    Description() string
    Background() string   // historical/mathematical context — where the algorithm comes from
    Run() (Trace, error)  // executes it and returns the full step-by-step record
}
```

A `Trace` is just an ordered list of `Step`s. Each `Step` carries:

- **What** happened and **why** it matters for security
- **Math** — the formula, plus a `MathLegend` explaining every symbol in it
  (so `a ←$ [1, n−1]` isn't just glyphs — `a`, `←$`, and the range each get
  a plain-English definition)
- **Inputs** / **Outputs** — the actual values that flowed through that
  operation (hex-encoded)
- **DurationUs** — real measured execution time in microseconds

This is the extension point: adding a new algorithm (HKDF, ChaCha20-Poly1305,
ML-KEM, ML-DSA, ...) means writing one new package under
`internal/algorithms/` that implements `Algorithm`, then registering it in
`main.go`. The frontend needs no changes to pick it up — the step list,
detail panel, legends, and playback controls are all driven by the JSON the
backend returns. Only a custom flow diagram (like ECDH's) is
algorithm-specific and would need its own UI.

## What's implemented today

**ECDH key exchange (NIST P-256)** — Alice and Bob each generate a key
pair, exchange public keys, and independently derive the same shared
secret, broken into 9 steps from curve selection through verification.
The UI offers two ways to explore it:

- **Diagram view** — a clickable flow diagram (Alice/Bob columns →
  exchange → shared secret) that highlights the node matching whichever
  step is selected.
- **Step-by-step view** — a step list + detail panel. Steps reveal one at a
  time as you click **Next** (or **Play** for hands-off auto-advance);
  **Prev**, **Show all**, and a progress indicator round out playback.

## Running it

Requires Go 1.25+ and Node 20+.

**Backend** (serves the API on `:8081`):

```bash
cd backend
go run ./cmd/server
```

**Frontend** (dev server on `:4200`, calls the backend directly at
`:8081` — CORS is enabled on the backend for this):

```bash
cd frontend
npm install
npm start
```

Then open [http://localhost:4200](http://localhost:4200) and click **Run**.

### API

```
GET  /api/algorithms          → list of { id, title, description, background }
POST /api/algorithms/{id}/run → executes it, returns the full Trace as JSON
```

## Roadmap

The longer-term idea is a full hybrid protocol pipeline — chaining several
algorithms' traces into one Alice → Bob workflow, e.g.:

```
[ ECDH ]   ──┐
             ├──► [ HKDF ] ──► [ ChaCha20-Poly1305 ] ──► [ Encrypted message ]
[ ML-KEM ] ──┘

[ ML-DSA ] ───────────────────────────────► [ Verify ]
```

That needs more algorithms in the registry first. Candidates, grouped the
way cryptography is usually taught (each category exercises the engine
contract differently, which is useful — a hash function has no `actor`
split at all, for instance):

- **Symmetric key** — AES, DES: single shared key, same operation
  encrypts and decrypts.
- **Asymmetric key** — RSA (keygen → encrypt → decrypt maps almost
  directly onto a step trace), ECDSA (signing, as opposed to ECDH's key
  agreement).
- **Hash functions** — SHA-256, MD5: no key, one-way, fixed-length
  digest from arbitrary-length input. A structurally different `Step`
  shape than ECDH/RSA, good for proving the contract generalizes.
- **Post-quantum** — ML-KEM (key encapsulation), ML-DSA (signatures) —
  needed for the hybrid pipeline above.
- **Classical ciphers** — Caesar, Vigenère, Atbash, ROT13, Rail Fence:
  low-effort to implement, good for teaching substitution vs.
  transposition before diving into modern algorithms.

Immediate next step, regardless of which of the above comes first: a
"workflow" concept that composes multiple algorithms' traces into one
end-to-end run, since every category above eventually feeds into the
hybrid pipeline.
