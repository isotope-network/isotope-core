# ISOTOPE — Infrastructure for Ethical, Unstoppable, Self-Learning Data and AI Exchange

**Ask without revealing. Answer without exposing.**

ISOTOPE is an infrastructure that unites data exchange,
distributed AI, and human communication
in a single decentralized network.

---

## Three Pillars of ISOTOPE

**Data.**
Query-response without disclosure.
Banks, insurers, hospitals, and suppliers exchange
answers to questions without transferring the data itself.

**AI.**
Distributed inference.
Lightweight models run on network nodes.
Simple queries are handled by the nearest node.
Complex ones go to the data center.
Every model passes an ethical passport before loading.

**People.**
The messenger as the entry point.
Communication without censorship, blocking, or surveillance.
A user's node automatically participates in network work.

The foundation is the **Network**:
P2P, ethical hash, immunity, self-learning.

---

## How the Architecture Works

ISOTOPE is built on a single pattern:
**query-response without disclosure**.

A participant asks a question.
The network finds nodes capable of answering.
The answer arrives without access to the underlying data,
without revealing the source,
and without identifying the respondent.

- «Has this client defaulted in the last 2 years?» → Answer: «No»
- «What is the cardiovascular risk coefficient?» → Answer: «0.88»
- «Is this certificate valid?» → Answer: «No»
- «Translate this text» → The nearest node with a language model answers

Data is never transferred. Identity is never revealed.
Only the answer matters.

---

## Architectural Principles

**Ethical Hash.**
The digital DNA of the network.
Seven universal commandments,
transformed into a 100-dimensional vector.
Every message and every AI response
is compared to the standard.
The closer to the standard, the longer it lives.
The further, the faster it fades.
This is not censorship. This is an immune system.

**Weighted Memory.**
Messages are not deleted by command.
They receive weight.
Likes raise weight. Dislikes lower it.
Time erodes even the strongest signals.
When weight falls below a threshold, data moves to archive.
Lower — deleted forever.
The network breathes:
what matters stays, the random goes, the harmful is rejected.

**Collective Learning.**
The neural network learns from community feedback.
Every like and dislike shifts the weights.
The network is not programmed — it is trained.
No moderator. No banned word dictionaries.

**Decentralization.**
No server. No company. No single center.
Nodes discover each other via mDNS, DHT, Bluetooth.
The network lives as long as at least one node lives.

**Privacy by Default.**
Every message is end-to-end encrypted.
The relay sees only ciphertext.
The key belongs only to the sender and the recipient.

**Emergent Trust.**
Verified is a signal, not a pass.
Weight is earned, not proven.
Trust grows from interaction, not from authority.

---

## Architectural Properties

**Immunity activates with scale.**

| Nodes | Property |
|-------|----------|
| 2-3 | Secure P2P channel. No immunity |
| 5-10 | Early consensus. Outliers visible |
| 15-20 | Immunity activates. Weights begin to work |
| 50+ | Stable immunity. Collusion has no effect |
| 100+ | Unstoppable network. Self-healing |

This is not a flaw. This is an architectural property.
Scale activates immunity.

**Weight Access Model.**
Node weight is a universal pass.

| Weight | Rights |
|--------|--------|
| 0–0.3 | Read (teaser) |
| 0.3–0.5 | Full access |
| 0.5–0.7 | Comment, moderate |
| 0.7–1.0 | Vote, relay, replicas |

[Details →](docs/architecture/WEIGHT_ACCESS_MODEL.md)

---

## Security Layers

| Layer | What is protected | Status |
|-------|-------------------|--------|
| Obfuscation | Traffic masking | ✅ v1.13 |
| E2E encryption | Message content (from relay) | ✅ v1.24 |
| Signature | Sender authenticity | ✅ v1.24 |
| Metadata | Who talks to whom | 🔜 v2.0+ (Onion) |
| Traffic patterns | Timing, volume | 🔜 Padding, mixing |

**What works now:** message content is protected from relay and from interception. Sender authenticity is verified. Relay sees only ciphertext.

**What comes later:** metadata protection (Onion routing), traffic pattern protection (padding, mixing).

---

## What the Architecture Enables

The architecture supports any interaction requiring
decentralization, ethical filtering,
and trust without intermediaries.

**Ready-made scenarios.**
Solve a specific problem in one day.

| Scenario | Problem | Minimum Nodes |
|----------|---------|---------------|
| Certificate Verification | Fake certificates, slow checks | 3 |
| Credit Check | Clients leave, unwilling to disclose full history | 3-5 |
| Anonymous Voting | Distrust in results, fear of exposure | 20-30 |
| Anti-Fraud | Money stolen while verification is pending | 5-10 |
| Doctor Consultation | Cannot share patient data with a colleague | 2-5 |
| Simple AI Queries | AI servers overloaded at peak hours | 5-10 |

[More: 10 domains, 30+ scenarios →](docs/USE_CASES.md)

---

## The Messenger: Entry Point to the Network

A user installs the ISOTOPE messenger.
Their node automatically participates in all scenarios —
without asking, without distracting.

While the user chats with friends, their node:
- Verifies certificates for suppliers
- Helps banks catch fraudsters
- Answers simple AI queries
- Participates in anonymous scientific surveys
- Confirms votes in local communities

Once a day — a single notification:

> «Your node today: 12 certificates verified,
> 3 AI queries processed,
> 1 fraudster caught.
> Earned: 2.8 ISOTOPE.»

**Formula:** Install messenger. Chat. Network works. Tokens arrive.

---

## Status

**v1.27.0 — stable (contacts and identity).**

Implemented:
- P2P network: libp2p + mDNS + DHT + Gossip
- Priority Gossip
- Associative memory
- WebSocket + TLS
- Obfuscation: AES-GCM + random delays
- Voice steganography: LSB in WAV
- Onion Routing v2
- Neural network: 100-dimensional vectors, bigrams, ethical filter
- Weighted memory with archive and auto-cleanup
- Expiring messages (TTL)
- Local encryption (AES-256-GCM)
- Self-healing: heartbeat, auto-restart
- Replication
- Self-adaptation
- Channels with weight levels
- REST API + WebSocket
- Mobile app (Flutter + gomobile FFI)
- Stable PeerID on mobile
- Network change handling
- NodeInfo model with heartbeat
- Log transfer from Go core to Flutter
- Dynamic port search
- NSD discovery
- Network health monitoring
- 67 autotests
- 5 nodes in docker-compose

**E2E encryption (v1.24):**
- X25519 keypairs for message encryption (box.Seal)
- Ed25519 keypairs for signatures
- QR format v:1 with ed25519_pub, x25519_pub, signature
- E2E encryption of message content (nonce || ciphertext)
- Version 2 messages — E2E
- Relay sees only ciphertext
- Contact verification via Ed25519 signature over peerID || x25519_pub
- Verified flag: signal in UI, not a pass

**Contact protocol (v1.27):**
- Bootstrap-handshake: [CONTACT_HELLO] → [CONTACT_HELLO_ACK] → [CONTACT_REQUEST] → [CONTACT_ACCEPT]
- Open service messages (HELLO, ACK) always via bootstrap
- E2E messages (REQUEST, ACCEPT) via circuit → bootstrap fallback
- tempContacts: temporary in-memory contacts for decryption
- Push events to UI via messageHook (not polling)
- Symmetry: both sides confirmed: true

**Identity and names (v1.27):**
- Name (local) / RemoteName (contact's representation) / PeerID (fallback)
- UI: Name → RemoteName → PeerID
- MyDisplayName in Settings — default representation
- QR contains display_name
- Dialog «How should we introduce you?» when sending request
- Profile in settings → «Your name»
- Long tap on contact → bottom sheet: Open / Rename / Delete
- Security warning on request

**Statuses (v1.26):**
- ✓ / ✓✓ / ✓🔒 / ✓✓ (colored)
- Hidden is terminal
- read_enabled — symmetric

**Send timer (v1.26):**
- 0/3/5/10 sec delay
- Draft on back
- Cancel button

**Deferred:**
- BLE — unstable, disabled
- Samsung Android 10 — crash
- DHT Provide — falls with few peers
- messageStatus growth — cleanup needed
- TTL for tempContacts (5 min)
- [PROFILE_UPDATE]

In development:
- Circuit direct when 15+ peers
- Batch [READ]
- Metadata protection — Onion (v2.0+)
- ISOTOPE Enterprise (B2B data exchange)
- ISOTOPE AI Mesh (distributed AI inference)

[Roadmap →](docs/roadmap/ROADMAP.md)

---

## Quick Start

### Requirements
- Docker and Docker Compose
- Go 1.21+ (for building from source)
- Flutter 3.x (for mobile app)

### Run a Node

    git clone https://github.com/isotope-network/isotope-core.git
    cd isotope-core
    docker compose build --no-cache
    docker compose up -d

Nodes:
- http://localhost:8081 (node 1)
- http://localhost:8082 (node 2)
- http://localhost:8083 (node 3)
- http://localhost:8084 (node 4)
- http://localhost:8085 (node 5)

### Build from Source

    go build -o isotope-node ./node/main
    ./isotope-node --config config.json

### Use as a Library

import core "sbimain"

config := core.Config{
    NodeID:     "node-1",
    Port:       9001,
    Transports: []string{"ws"},
}

node := core.NewNode(config)
core.InitP2P(node)
core.StartHTTP(node)
defer core.Stop(node)

---

## Repository Architecture

    isotope-core/
    ├── node/           # Go core (P2P, neural network, memory, API)
    │   ├── *.go        # package core (library)
    │   ├── main/       # entry point (package main)
    │   └── mobile/     # gomobile binding
    ├── mobile/         # Flutter app
    ├── tests/          # Autotests
    ├── docs/           # Documentation and philosophy
    ├── genesis/        # Ethical hash
    └── docker-compose.yml

[Architecture details →](docs/architecture/OVERVIEW.md)

---

## Documentation

- [Manifesto](docs/philosophy/MANIFEST.md)
- [The Story](docs/philosophy/THE_STORY.md)
- [Applications](docs/APPLICATIONS.md)
- [Use Cases](docs/USE_CASES.md)
- [Immunity Scale](docs/architecture/IMMUNITY_SCALE.md)
- [Weight Access Model](docs/architecture/WEIGHT_ACCESS_MODEL.md)
- [FAQ — Expert Questions](docs/FAQ.md)
- [Contributing](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security](SECURITY.md)
- [Mirrors](docs/MIRRORS.md)

---

## License

Core: **AGPL v3**

Commercial license: [LICENSE.COMMERCIAL.md](LICENSE.COMMERCIAL.md)

---

## Contacts

- Website: [isotope.network](https://isotope.network)
- Zone: [isotope.zone](https://isotope.zone)
- GitHub: [github.com/isotope-network](https://github.com/isotope-network)
- Email: keeper@isotope.network

---

**ISOTOPE is an infrastructure.
Data. AI. People.**