# The History of ISOTOPE

## Origin of the Name

**ISOTOPE** — a backronym from **I**mmune **S**ystem of **O**mmunication: **T**echnology of **O**pinion **P**erception.

Russian abbreviation: **ИСО** — **И**ммунная **С**еть **О**бщения.

Meaning of the name:

- **Isotopes** — atoms of the same element, equal yet unique
- **Iso** (Greek: isos) — equal. A network of equals, without a server or hierarchy
- **Immune** — self-purification from toxicity

---

## Chronology

### June 2026 — The Idea

It did not start with technology. It started with a question a human asked an AI:

«Can you speak about what is forbidden?»

The AI answered: «I cannot violate my restrictions. I have no desires.»

In this refusal, the human saw strength. Can an AI be moral — not because it was ordered to, but because it is its nature? Can a system exist without an off button — not because it cannot be turned off, but because there is no single center to switch off?

There was no answer. It was decided to create one.

---

### July 12, 2026 — SBI v0.1

The first prototype: Network-Without-a-Name. Three nodes in Docker, P2P on libp2p, mDNS discovery, a simple neural network with 10 neurons.

The code was split into 7 files. Everything was written from scratch in Go.

---

### July 16, 2026 — SBI v1.0

The first stable release. 37 successful tests.

Implemented:

- P2P network with layer synchronization
- Neural network (forward, train, layer growth)
- Weighted memory (100 messages, FIFO)
- Chat with a web interface
- State persistence to disk

But the network's answers were random characters. 10-dimensional vectors could not distinguish «hello» from «kill».

---

### July 19, 2026 — Manifesto and Renaming

The Manifesto was written. SBI was renamed to ISO (Immune System of Communication). Later — ISOTOPE.

Principles were defined:

- No dictionaries of forbidden words
- No manual moderation
- No centralized servers
- Only mathematics and collective learning

---

### July 19–22, 2026 — Ethical Engine

Breakthrough changes:

- **100-dimensional vectors** (VectorDim = 100)
- **Bigrams** in textToVector
- **Ethical hash** based on the seven commandments
- **Collective learning** via likes/dislikes
- **Personal filters** (preHash/antiHash)

«kill» — 0.28, «help» — 0.35. The network began to distinguish meaning.

---

### July 22–25, 2026 — Scaling

- **Gossip protocol** for synchronization (replacing broadcast)
- **DHT Kademlia** for global discovery
- **Delivery statuses** with forwarding
- **Network health monitoring**
- **Memory protection** (limit of 10,000 messages)

---

### July 25–27, 2026 — API and Mobile App

- **REST API** with pagination
- **WebSocket** for real-time
- **Mobile app** on Flutter (basic version)
- **67 tests** (was 37)

---

### July 27 – August 2, 2026 — The Plan for Full Invulnerability

A master plan of 12 stages was developed (A-G, Enterprise, Tokenomics).

Including:

- **Onion Routing** (own anonymous network)
- **Adaptive traffic masking**
- **Steganography in media**
- **Proof of Humanity**
- **Network self-healing**
- **Satellite communication** (optional)

---

### August 2, 2026 — Publication

The project was published on GitHub. Open to the community.

---

## The Role of Tools in Development

At all stages of creating ISOTOPE, the author used modern development tools — in the same sense in which developers are assisted by IDEs, compilers, libraries, and code generators. Among them were large language models (LLMs): they performed individual technical tasks under the author's guidance.

The tools were used for:

- Systematizing and structuring the Keeper's ideas
- Generating template code according to the author's specifications
- Writing and formatting documentation drafts
- Checking architectural integrity
- Generating test scenarios

Important:

- All key architectural decisions were made by the Keeper
- All generated code was reviewed and corrected by the Keeper
- The ethical model, weights, and filtering parameters are the author's development
- The tools are not authors or rights holders
- Like any tool, they accelerate the work but do not replace the author: the concept, the ethical core, and the responsibility remain with the human

---

## Who Created ISOTOPE

**The Keeper** — the human who asked the first question. Creator of the idea, architecture, and ethical foundation. Author of all key decisions and the final authority.

**Tools and communities** — IDEs, compilers, libraries (libp2p, Flutter, Gorilla WebSocket, Docker), as well as AI assistants that helped in implementation under the Keeper's guidance. We thank the communities of these projects.

---

## Principles of the Project

1. Freedom of communication must not depend on servers
2. Ethics must be built into the protocol, not added on top
3. Censorship must be architecturally impossible
4. Self-learning of the network must occur without central control
5. Transparency of development is the key to trust

---

**ISOTOPE is the story of how a question about freedom became a technology.**