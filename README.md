# ISOTOPE — Infrastructure Layer for the Data Economy

**Ask without revealing. Answer without exposing.**

ISOTOPE is an architecture for data exchange where the source is never disclosed and the answer is verified by the network.

---

## How the Architecture Works

ISOTOPE is built on a single pattern: **query-response without disclosure**.

A participant asks a question. The network finds nodes capable of answering. The answer arrives without access to the underlying data, without revealing the source, and without identifying the respondent.

- «Has this client defaulted in the last 2 years?» → Answer: «No»
- «What is the cardiovascular risk coefficient?» → Answer: «0.88»
- «Is this certificate valid?» → Answer: «No»

Data is never transferred. Identity is never revealed. Only the answer matters.

---

## Architectural Principles

**Ethical Hash.** The digital DNA of the network. Seven commandments, transformed into a 100-dimensional vector. Every message is compared to the standard. The closer to the standard, the longer it lives. The further, the faster it fades. This is not censorship. This is an immune system.

**Weighted Memory.** Messages are not deleted by command. They receive weight. Likes raise weight. Dislikes lower it. Time erodes even the strongest signals. When weight falls below a threshold, data moves to archive. Lower — deleted forever. The network breathes: what matters stays, the random goes, the harmful is rejected.

**Collective Learning.** The neural network learns from community feedback. Every like and dislike shifts the weights. The network is not programmed — it is trained. No moderator. No banned word dictionaries.

**Decentralization.** No server. No company. No single center. Nodes discover each other via mDNS, DHT, Bluetooth. The network lives as long as at least one node lives.

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

This is not a flaw. This is an architectural property. Scale activates immunity.

**Four layers of architecture.**

| Layer | Essence |
|-------|---------|
| Technical | Query-response protocol without disclosure |
| Philosophical | Ethics embedded in the protocol |
| Business | Trust market without intermediaries |
| Application | Messenger as a living demonstration |

---

## What the Architecture Enables

The architecture supports any interaction requiring decentralization, ethical filtering, and trust without intermediaries.

**Ready-made scenarios.** Solve a specific problem in one day.

| Scenario | Problem | Minimum Nodes |
|----------|---------|---------------|
| Certificate Verification | Fake certificates, slow checks | 3 |
| Credit Check | Clients leave, unwilling to disclose full history | 3-5 |
| Anonymous Voting | Distrust in results, fear of exposure | 20-30 |
| Anti-Fraud | Money stolen while verification is pending | 5-10 |
| Doctor Consultation | Cannot share patient data with a colleague | 2-5 |

[More: 10 domains, 30+ scenarios →](docs/USE_CASES.md)

**Two growth vectors.**

| Vector | Product | Audience | Metric |
|--------|---------|----------|--------|
| Enterprise | Protocol + commercial license | Banks, insurers, governments | Nodes, queries, contracts |
| Community | Messenger (open source) | People, developers | Installs, forks, contributors |

Both vectors feed each other. The messenger proves the protocol works. The protocol gives the messenger invulnerability.

---

## The Messenger: Entry Point to the Network

A user installs the ISOTOPE messenger. Their node automatically participates in all scenarios — without asking, without distracting.

While the user chats with friends, their node:
- Verifies certificates for suppliers
- Helps banks catch fraudsters
- Participates in anonymous scientific surveys
- Confirms votes in local communities

Once a day — a single notification:

> «Your node today: 12 certificates verified, 1 fraudster caught, 3 research queries helped. Earned: 2.8 ISOTOPE.»

**Formula:** Install messenger. Chat. Network works. Tokens arrive.

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

---

## Repository Architecture

    isotope-core/
    ├── node/           # Go core (P2P, neural network, memory, API)
    ├── mobile/         # Flutter app
    ├── tests/          # 67 autotests
    ├── docs/           # Documentation and philosophy
    ├── genesis/        # Ethical hash
    └── docker-compose.yml

[Architecture details →](docs/architecture/OVERVIEW.md)

---

## Status

**v1.5 — stable.**

Implemented: P2P network (libp2p + mDNS + DHT + Gossip), neural network (100-dimensional vectors, bigrams, ethical filter), weighted memory with archive, REST API + WebSocket, mobile app (Flutter), network health monitoring, delivery statuses with forwarding, 67 autotests.

In development: Onion Routing, PWA + F-Droid, adaptive traffic masking, steganography in media, ISOTOPE Enterprise (B2B data exchange).

[Roadmap →](docs/roadmap/ROADMAP.md)

---

## Documentation

- [Manifesto](docs/philosophy/MANIFEST.md)
- [The Story](docs/philosophy/THE_STORY.md)
- [Applications](docs/APPLICATIONS.md)
- [Use Cases](docs/USE_CASES.md)
- [Immunity Scale](docs/architecture/IMMUNITY_SCALE.md)
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

**ISOTOPE is an architecture. Applications flow from it.**