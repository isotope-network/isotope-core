# ISOTOPE — Immune System Of Transparent Open Peer Exchange

**Communication that cannot be blocked, tracked, or destroyed.**

ISOTOPE is a decentralized P2P messenger with ethical immunity, anonymous routing, and adaptive traffic masking.

---

## Features

- **Serverless P2P** — impossible to block or shut down
- **Ethical Immunity** — neural network filters violence and spam automatically
- **Onion Routing** — anonymity on your own peers, no Tor required
- **Adaptive Masking** — traffic indistinguishable from regular HTTPS
- **Weighted Memory** — important data lives, garbage fades
- **Self-Adaptation** — network adjusts parameters under load
- **E2E Encryption** — no one reads messages except the recipient
- **Offline Operation** — mDNS, Bluetooth mesh, Wi-Fi Direct
- **Bypass Blocking** — PWA, F-Droid, direct APK download

---

## Applications

ISOTOPE is not just a messenger. It is a protocol for any interaction requiring decentralization, ethical filtering, and an economy of trust.

| Domain | Application |
|--------|-------------|
| Global Economy | Cross-border payments without SWIFT, decentralized data market, credit scoring |
| Legal | International arbitration, supply chain certification, digital inheritance |
| Technology | IoT without cloud, federated AI training, P2P DNS without root servers |
| Society & Government | Electronic voting, direct charity, emergency communication |
| Science | Decentralized peer review, research consortiums |

[Details on each direction →](docs/APPLICATIONS.md)

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

### Mobile App

    cd mobile
    flutter pub get
    flutter run

---

## How It Works

- **Ethical Hash** — digital DNA of the network based on seven commandments
- **Neural Network** — 100-dimensional vectors compare messages to the standard
- **Collective Learning** — likes/dislikes teach the network to distinguish good from evil
- **Weighted Memory** — important lives longer, spam fades and gets deleted
- **Onion Routing** — three random peers form an anonymous chain
- **Self-Adaptation** — network collects metrics and adjusts parameters

---

## Architecture

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

Implemented:
- P2P network: libp2p + mDNS + DHT + Gossip
- Neural network: 100-dimensional vectors, bigrams, ethical filter
- Memory: weighted, with archive and auto-cleanup
- REST API + WebSocket
- Mobile app (Flutter)
- Network health monitoring
- Delivery statuses with forwarding
- 67 autotests

In development:
- Onion Routing
- PWA + F-Droid
- Adaptive traffic masking
- Steganography in media
- ISOTOPE Enterprise (B2B data exchange)

[Roadmap →](docs/roadmap/ROADMAP.md)

---

## Documentation

- [Manifesto](docs/philosophy/MANIFEST.md)
- [The Story](docs/philosophy/THE_STORY.md)
- [Contributing](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security](SECURITY.md)
- [Applications](docs/APPLICATIONS.md)
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

**ISOTOPE is not a messenger. It is communication that cannot be detected, blocked, or destroyed.**