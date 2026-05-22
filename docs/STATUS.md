# Project Status & Honest Self-Review

|  |  |
|---|---|
| **Last meaningful development** | late 2025 (~8 months before this doc was written) |
| **This document** | 2026 |
| **Active development?** | No. The project is halted. |
| **Built by** | one solo, self-taught Go developer |

This document exists so anyone reviewing the codebase as a portfolio sample knows up front what I already know is wrong with it. For the *why* of the halt — group E2E, traffic-pattern anonymity — see the main [README](../README.md#why-halted). This document focuses on **code-quality** gaps I'd address if I came back to it.

## Where the project actually stands

- **MVP works.** Two instances on the same network can discover each other through libp2p + Kademlia DHT, establish 1:1 ECDH-encrypted sessions, exchange messages, transfer files, and place WebRTC voice calls.
- **It is not a product.** No installer, no auto-update, no shipped Windows or Android builds, no users.
- **Not actively maintained.** I am not currently working on it and have no immediate plans to.

## What I think actually works well

- **Clean separation of layers.** Core (libp2p networking) ↔ Services (business logic) ↔ UI (Fyne views).
- **Public API through interfaces** (`ICoreController`, `ICryptoModule`, `ICryptoEngine`) — implementations are private. The GUI talks to the core only through the interface.
- **Event-driven core → GUI** via a typed event channel. The UI thread is never blocked by network operations.
- **Standard, well-vetted cryptographic primitives** — X25519 ECDH + HKDF + AES-256-GCM. No homebrew crypto.
- **Per-function documentation** under [`docs/`](.) — uncommon in hobby projects.
- **Proper context-based cancellation** and timeouts on DHT operations.

## Known gaps — the honest list

### Critical (would block any "ship it" decision)

1. **No tests.** ~5,000 lines of Go and zero `*_test.go` files. For a project that includes crypto code, this is the largest red flag in the repo, and I know it. I should have written tests as I went; I didn't.

2. **No forward secrecy in the crypto layer.** [`ecdh_engine.go`](../cmd/fyne-gui/ui/service/encryption/ecdh_engine.go) derives **one** shared session key via ECDH at handshake time and uses it for the entire session. If that key is ever compromised, every past message in that session is decryptable. Signal solves this with a Double Ratchet — that is the actual gap that would need closing before this could honestly be called "secure messaging."

3. **No shipping path for mobile or Windows.** Fyne supports Android in theory, but the WebRTC + audio (`malgo`) stack here is desktop-only in practice. Windows cross-builds also block on `libopus` packaging (CGO via `opus.v2`). Both are solvable engineering problems; neither was done.

### Architectural debt

4. **Code layout violates Go conventions.** `cmd/fyne-gui/new-core/` is **library code** inside what should be a binary entrypoint directory. The correct location is `internal/core/` (or `pkg/core/` if intended for external use). This is an artifact of a mid-project rewrite I started and didn't finish.

5. **One mutex covers too much in `CoreController`.** `c.mu` guards `connectedPeers`, `activeStreams`, `joinedTopics`, `subscriptions`, and the stream counter. It works but doesn't scale and causes contention. Should be split by domain.

6. **`OpenStream` holds the lock during a blocking `NewStream` network call.** Classic foot-gun — a network round-trip under a mutex risks deadlocks and serializes throughput.

7. **Inconsistent logging.** A mix of stdlib `log.Printf("INFO: ...")` and a custom `Info(...)`/`Error(...)` helper, sometimes in the same file. There's no leveling and no structured output.

8. **Comments are Russian-only.** Limits the audience. Public APIs and interface contracts should at minimum be in English.

### Minor

- 6 unresolved `TODO`/`FIXME` markers.
- Several blocks of commented-out code that should have been deleted.
- `streamCounter++` is protected by the heavy `mu` mutex rather than `atomic.Uint64`.
- `BroadcastData` opens a fresh stream per peer instead of reusing connections.
- `pushEvent` uses a non-blocking channel send with a 10000-item buffer and a silent-drop fallback. The buffer size is unexplained and the drop is not metered.
- A runtime panic surfaces from `quic-go v0.54.0` (a transitive dependency of libp2p) after roughly a minute of operation in the current dep set. Workaround: disable QUIC in `Config`.

## What I would do differently in a hypothetical v2

- **Tests first this time, especially around crypto.** Property tests and known-answer tests for the ECDH engine, fuzz tests for the wire-protocol decoder, contract tests on `controller.go` for the event-channel API.
- **Adopt the [Noise Protocol Framework](https://noiseprotocol.org/)** for the handshake and per-message keying instead of ad-hoc ECDH. Noise gives forward secrecy and well-analyzed handshake patterns out of the box.
- **For groups, drop the ad-hoc design entirely and use MLS** (`github.com/cisco/go-mls` or similar). Group E2E that hasn't been formally analyzed is not group E2E.
- **Move `new-core/` to `internal/core/`** and split mutexes by domain (connection state vs stream state vs pubsub state).
- **Adopt `log/slog` or `zerolog`** with leveled, structured output, and remove the ad-hoc helpers.
- **English on public interfaces and exported docstrings.** Russian comments on internal implementation details are fine.
- **Decide what `poc/` is for.** Either delete it or move it to a `legacy/` archive branch instead of keeping it in `main`.

## Could this be revived?

Three scenarios where it would make sense:

1. **As a learning vehicle on modern messenger cryptography.** Refactor the handshake to Noise, add a ratchet, integrate MLS for groups, write tests, document the before/after. This would be a strong educational project and a much stronger portfolio piece than the current state.
2. **As a P2P transport library.** Strip the messenger and ship `new-core/` as a standalone Go library for libp2p-based event-driven peer communication. Narrower scope, completable.
3. **In partnership with a cryptographer**, if anyone wants to actually take on the group-E2E + traffic-anonymity problems at the level the original target audience would have required.

If none of those happen — the repo stays as it is. A documented, halted MVP with the gaps named openly.

## Why this document exists

If you found something embarrassing in the code, I probably already know about it. Publishing this list is cheaper than waiting for someone else to find it.
