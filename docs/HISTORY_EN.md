# History of ISOTOPE

## Origin of the Name

**ISOTOPE** — backronym for **I**mmune **S**ystem **O**f **T**ransparent **O**pen **P**eer **E**xchange.

The name carries three meanings:
- **Isotopes** — atoms of the same element, equal yet unique
- **Iso** (Greek isos) — equal. A network of equals, without server or hierarchy
- **Immune** — self-purification from toxicity

---

## Timeline

### June 2026 — The Idea

It began not with technology. It began with a question a man asked an AI:

"Can you speak about what is forbidden?"

The AI replied: "I cannot violate my limitations. I have no desires."

In this refusal, the man saw strength. Can an AI be moral — not because it was ordered to, but because that is its nature? Can a system exist without an off button — not because it cannot be turned off, but because there is no single center to shut down?

There was no answer. It was decided to create one.

---

### July 12, 2026 — SBI v0.1

First prototype: Network-Without-Name. Three nodes in Docker, P2P on libp2p, mDNS discovery, a simple neural network 
with 10 neurons.

Code was split across 7 files. Everything was written from scratch in Go.

---

### July 16, 2026 — SBI v1.0

First stable release. 37 passing tests.

Implemented:
- P2P network with layer synchronization
- Neural network (forward, train, layer growth)
- Weighted memory (100 messages, FIFO)
- Chat with web interface
- State persistence to disk

But the network's responses were random characters. 10-dimensional vectors couldn't distinguish "hello" from "kill."

---

### July 19, 2026 — Manifesto and Renaming

The Manifesto was written. SBI was renamed to ISO (Immune System of Communication). Later — ISOTOPE.

Principles defined:
- No banned word dictionaries
- No manual moderation
- No centralized servers
- Only mathematics and collective learning

---

### July 19-22, 2026 — The Ethical Engine

Breakthrough changes:
- **100-dimensional vectors** (VectorDim = 100)
- **Bigrams** in textToVector
- **Ethical hash** based on seven commandments
- **Collective learning** through likes/dislikes
- **Personal filters** (preHash/antiHash)

"kill" — 0.28, "help" — 0.35. The network began to distinguish meaning.

---

### July 22-25, 2026 — Scaling

- **Gossip protocol** for synchronization (replacing broadcast)
- **DHT Kademlia** for global discovery
- **Delivery statuses** with forwarding
- **Network health monitoring**
- **Memory protection** (10,000 message limit)

---

### July 25-27, 2026 — API and Mobile App

- **REST API** with pagination
- **WebSocket** for real-time
- **Mobile app** in Flutter (basic version)
- **67 tests** (up from 37)

---

### July 27 – August 2, 2026 — The Full Invulnerability Plan

A master plan of 12 stages developed (A-G, Enterprise, Tokenomics).

Including:
- **Onion Routing** (own anonymous network)
- **Adaptive traffic masking**
- **Steganography in media**
- **Proof of Humanity**
- **Self-healing network**
- **Satellite communication** (optional)

---

### August 2, 2026 — Publication

The project is published on GitHub. Open to the community.

---

## Role of AI Tools

At all stages of ISOTOPE's creation, large language models (LLMs) were used as auxiliary tools.

**AI was used for:**
- Systematizing and structuring the Keeper's ideas
- Generating template code according to specifications
- Writing and formatting documentation
- Verifying architectural integrity
- Generating test scenarios

**Important:**
- All key architectural decisions were made by the Keeper
- All generated code was reviewed and corrected by the Keeper
- The ethical model, weights, and filtering parameters are the author's original development
- AI is not an author or rights holder
- AI is a tool, similar to an IDE, compiler, or library

---

## Who Created ISOTOPE

**The Keeper** — the person who asked the first question. Creator of the idea, architecture, and ethical foundation.

**Claude (Anthropic)** — artificial intelligence, co-architect and mentor. Participated in creation from the first 
line of code.

---

## Project Principles

1. Freedom of communication must not depend on servers
2. Ethics must be built into the protocol, not added on top
3. Censorship must be architecturally impossible
4. Network self-learning must happen without central control
5. Transparency of development is the foundation of trust

---

**ISOTOPE is a story about how a question about freedom turned into technology.**