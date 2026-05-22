# OwlWhisper 🦉

[![Language](https://img.shields.io/badge/language-Go-00ADD8.svg)](https://go.dev)
[![Status](https://img.shields.io/badge/status-halted-orange.svg)](#why-halted)
[![License](https://img.shields.io/badge/License-GPL%203.0-blue.svg)](LICENSE)

> **Near-complete P2P messenger MVP in Go** — libp2p + Fyne desktop GUI + WebRTC voice calls + ECDH 1:1 encryption + protobuf wire protocol. Halted before group E2E and traffic-pattern anonymity — see [Why halted](#why-halted).

## What this is

OwlWhisper is an attempt to build a peer-to-peer desktop messenger from scratch in Go — no central server, no operator who can read or hand over conversations. The work reached a near-complete MVP: peers discover each other over libp2p + Kademlia DHT, exchange typed protobuf messages, transfer files in chunked streams, and place WebRTC voice calls. 1:1 messages are encrypted with ECDH key agreement.

The project is **halted** — not because the MVP didn't work, but because the gap between *"MVP works on my LAN"* and *"safe for the audience this was built for"* turned out to be funded-research wide. See [Why halted](#why-halted).

## What got built

| Layer | Status | Implementation |
|-------|--------|---------------|
| Peer discovery | ✅ | libp2p + Kademlia DHT + mDNS |
| Connection mgmt | ✅ | libp2p host, multi-transport |
| Wire protocol | ✅ | protobuf — 8 typed modules (chat, file_transfer, identity, signaling, capabilities, command, envelope, ping) |
| 1:1 encryption | ✅ | ECDH key agreement |
| Voice calls | ✅ | WebRTC (pion/webrtc/v4) + Opus codec |
| Audio I/O | ✅ | malgo (miniaudio bindings) |
| File transfer | ✅ | Chunked stream-based send/receive |
| Desktop GUI | ✅ | Fyne v2 — contacts, profiles, chat, search, file cards, call controls |
| Per-function docs | ✅ | See [`docs/`](docs/) |
| Group E2E (Signal/MLS-class) | ❌ | Not built — see [below](#1-end-to-end-encryption-for-groups) |
| Traffic-pattern anonymity | ❌ | Not built — see [below](#2-traffic-pattern-anonymity) |

The CLI predecessor lives in [`poc/`](poc/) and is kept for historical reference. A small Python interop experiment (key generation, basic protocol consumption) lives in [`examples/`](examples/).

## Why halted

The project was halted after I evaluated what would actually be required to make it useful for the audiences I had in mind — primarily independent / opposition-aligned communication channels in regions with active state-level network surveillance.

The MVP reached a working state. Two engineering walls beyond it turned out to be much higher than the early scope suggested:

### 1. End-to-end encryption for groups

1:1 ECDH is tractable and it's in the code. **Groups at scale** are a different category: Signal/MLS-level cryptographic ratchets with proper key distribution, member churn handling, forward secrecy across large rosters, post-compromise security — these require deep cryptographic engineering, not just plugging in a library. Getting it wrong means worse-than-nothing security: false confidence with real exposure.

### 2. Traffic-pattern anonymity

Even with perfect E2E, a peer-to-peer messenger leaks **who talks to whom and when**. For state-level adversaries doing traffic correlation, the metadata graph is enough to identify participants of an opposition channel. Defending against this is its own discipline — onion routing, mixnets, traffic padding, cover traffic.

I specifically explored layering the libp2p transport over Tor (so the peer behind a given peer ID would be harder to deanonymize), but the integration complexity, the performance hit on a latency-sensitive UX, and the fact that Tor alone doesn't fully neutralize traffic-pattern correlation made it the wrong tradeoff to ship as a "secure" messenger.

I made the call to stop rather than ship something that *looked* secure but wouldn't survive contact with the actual threat model.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.24 |
| P2P networking | libp2p + Kademlia DHT + mDNS |
| Wire protocol | protobuf (8 typed modules) |
| 1:1 encryption | ECDH key agreement |
| Voice calls | pion/webrtc/v4 + Opus |
| Audio I/O | malgo |
| Desktop GUI | Fyne v2 |
| Build/CI | Standard Go toolchain |
| License | GPL-3.0 |

## Repo layout

```
cmd/fyne-gui/                 desktop GUI binary
├── new-core/                 networking, protocol, encryption
│   ├── protocol/             protobuf .proto + generated .pb.go
│   └── *.go                  controller, discovery, node, crypto, etc.
└── ui/                       Fyne views and services
    └── service/              chat, contact, call, file, crypto, identity services
        └── encryption/       ECDH engine

docs/                         per-function reference (16 .md files)
examples/                     Python client experiment
poc/                          earlier CLI exploration (superseded)
```

## Lessons / Reflections

- **An MVP is achievable solo.** A self-taught dev can ship a working P2P transport + GUI + 1:1 crypto + voice calls in a reasonable timeline. That part is just engineering and persistence.
- **The hard part is the threat model.** When the audience is genuinely at risk, *"looks secure"* is dangerous. Group E2E and metadata anonymity are specialized cryptographic engineering — they don't yield to general-purpose hard work.
- **There's no shame in halting for the right reason.** A documented MVP that admits its gaps is more useful than a shipped product that lies about them.

## Could this be revived?

Possibly, but only with:
- A real cryptographer reviewing the protocol design — especially the group-messaging story.
- A clear scope: ship 1:1 first as a separate product, groups as a v2, anonymity layer in a separate transport.
- Explicit acknowledgement that this is an unfunded research project, not a consumer messenger.

If those preconditions are met — happy to discuss. Otherwise the repo stays as a documented experiment, and I may come back to it if and when the underlying problems have credible solutions.

## License

GPL-3.0 — see [LICENSE](LICENSE) for details.

The copyleft choice is deliberate: this codebase is aimed at independent / opposition-aligned communication tooling, and viral copyleft prevents downstream proprietary forks from undermining the audience the project was originally built for.

## Contact

- GitHub: [@ZenonEl](https://github.com/ZenonEl)
