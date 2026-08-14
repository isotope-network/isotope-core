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

**v1.9 — stable.**

Implemented:
- P2P network: libp2p + mDNS + DHT + Gossip
- Priority Gossip: urgent messages propagate faster
- Associative memory: nodes remember who asks whom
- WebSocket + TLS: traffic indistinguishable from HTTPS
- Neural network: 100-dimensional vectors, bigrams, ethical filter
- Weighted memory with archive and auto-cleanup
- REST API + WebSocket
- Mobile app (Flutter)
- Network health monitoring
- Delivery statuses with forwarding
- 67 autotests
- 5 nodes in docker-compose
- Onion Routing foundation

In development:
- Onion Routing (full chain of 3+ peers)
- PWA + F-Droid
- Adaptive traffic masking
- Steganography in media
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

---

## Repository Architecture

    isotope-core/
    ├── node/           # Go core (P2P, neural network, memory, API)
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