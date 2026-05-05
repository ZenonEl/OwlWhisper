# OwlWhisper 🦉

[![Language](https://img.shields.io/badge/language-Go-00ADD8.svg)](https://go.dev)
[![Status](https://img.shields.io/badge/status-discontinued-orange.svg)](#why-discontinued)
[![License](https://img.shields.io/badge/License-GPL%203.0-blue.svg)](LICENSE)

> **An experimental P2P messenger in Go.** Working PoC, but discontinued — see [Why discontinued](#why-discontinued) below.

## What this is

OwlWhisper started as an attempt to build a peer-to-peer messenger from scratch in Go: no central server, no operator who can read or hand over conversations. The PoC reached a working state — peers discover each other, establish connections, exchange messages.

## What got built

- **Networking core** (`pkg/`, `internal/`) — peer discovery, connection management, message routing.
- **CLI runner** (`cmd/owlwhisper`) — start a node, connect to peers, send/receive.
- **PoC fixtures** (`poc/`) — earlier exploration sketch, superseded by `cmd/`, `internal/`, and `pkg/`. Kept for historical reference.

The code in this repo represents a functioning peer-to-peer message exchange between two or more nodes on the same network. End-to-end encryption is **not included** — see below for why.

`pkg/interfaces/` and `pkg/config/` exist as unused architectural stubs from a more ambitious version of the project — left in place to honestly reflect the state at which the work was halted.

## Why discontinued

The project was halted after I evaluated what would actually be required to make it useful for the audiences I had in mind — primarily independent / opposition-aligned communication channels in regions with active state-level network surveillance.

Two engineering walls turned out to be much higher than the early scope suggested:

### 1. End-to-end encryption for large groups

Modern E2E protocols (Signal, MLS) are built around carefully designed cryptographic ratchets. For 1:1 chats, the math is well-understood and the libraries exist. For **groups at scale**, the picture is different: key distribution, member churn, forward secrecy across thousands of members, post-compromise security — these require deep cryptographic engineering, not just plugging in a library. Getting it wrong means worse-than-nothing security: false confidence with real exposure.

### 2. Traffic-pattern anonymity

Even with perfect E2E, a peer-to-peer messenger leaks **who talks to whom and when**. For state-level adversaries doing traffic correlation, the metadata graph is enough to identify participants of an opposition channel. Defending against this is its own discipline — onion routing, mixnets, traffic padding, cover traffic — and it's incompatible with the latency/UX expectations of a normal messaging app without significant investment.

I specifically explored layering the transport over Tor (so the peer behind a given libp2p ID would be harder to deanonymize), but the integration complexity, performance hit, and the fact that Tor itself doesn't fully neutralize traffic-pattern correlation made it the wrong tradeoff to ship as a "secure" messenger.

I made the call to stop rather than ship something that *looked* secure but wouldn't survive contact with the actual threat model.

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Language** | Go |
| **Module layout** | Standard Go (`cmd/`, `internal/`, `pkg/`) |
| **License** | GPL-3.0 |

## Lessons / Reflections

- **PoC is not the hard part.** The hard part is the security threat model when the audience is genuinely at risk. Don't ship messengers without doing the homework.
- **Cryptography is a specialized domain.** A confident self-taught dev can build a working P2P transport in Go in a few weeks. They can't deliver state-of-the-art group E2E + traffic anonymity in the same timeline. Knowing the gap is what mattered here.
- **There's no shame in halting a project for the right reason.** Better to leave a documented PoC than to ship insecure software to people whose freedom may depend on it.

## Could this be revived?

Possibly, but only with:
- A real cryptographer reviewing the protocol design.
- A clear scope: 1:1 messaging first, groups later, anonymity layer separately.
- Explicit acknowledgement that this is an unfunded research project, not a product.

If those preconditions are met — happy to discuss. Otherwise the repo stays as a documented experiment, and I may come back to it if and when the underlying problems have credible solutions.

## License

GPL-3.0 — see [LICENSE](LICENSE) for details.

The copyleft choice is deliberate: this codebase is aimed at independent / opposition-aligned communication tooling, and viral copyleft prevents downstream proprietary forks from undermining the audience the project was originally built for.

## Contact

- GitHub: [@ZenonEl](https://github.com/ZenonEl)
