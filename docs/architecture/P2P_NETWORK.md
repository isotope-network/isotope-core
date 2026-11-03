# ISOTOPE P2P Network

## Overview

The ISOTOPE P2P network is built on **libp2p** — a modular network stack developed by Protocol Labs (IPFS, Filecoin).

The network has no servers. Every node is equal.

---

## Network Architecture

### Transport Layer

| Transport | Purpose | Status |
|-----------|---------|--------|
| **TCP** | Primary transport | Works |
| **WebSocket + TLS** | HTTPS masquerading, DPI bypass | Works (WS 9001 blocked by Gorodok ISP) |
| **mDNS** | Local network discovery | Works |
| **Bluetooth** | Communication without internet | Planned |
| **Wi-Fi Direct** | Direct connection without router | Planned |

### Peer Discovery

| Mechanism | Scope | Status |
|-----------|-------|--------|
| **mDNS** | Local network | Works |
| **DHT Kademlia** | Global network (WAN) | Planned (15+ nodes) |
| **Bootstrap nodes** | First entry into the network | ISOTOPE VPS (bootstrap) |

### Layer Synchronization

| Mechanism | Description | Status |
|-----------|-------------|--------|
| **Gossip protocol** | Epidemic broadcast with TTL=3 | Works (for 3+ nodes) |
| **Delta synchronization** | Only changed layers | Works |
| **Throttling** | No more than once per 5 seconds | Works |

### Message Routing

| Mechanism | Description | Status |
|-----------|-------------|--------|
| **Gossip** | Broadcast to all subscribed nodes | Works |
| **Forwarding** | Relay through neighbors with TTL | Works |
| **Sphinx** | Layered encryption, 2 relays + recipient | Implemented (v2.0.1) |
| **Onion Routing (full)** | Chain of 3+ peers | Planned (v2.0+) |

---

## Protocols

### Main protocol: `/sbicore/1.0.0`

Used for message exchange between nodes.

Format:
MSGID:message text

text

### Synchronization protocol: `/sbicore/sync/1.0.0`

Used for exchanging neural network layers.

Format (JSON):

```json
{
  "ttl": 3,
  "delta": [
    {"index": 0, "data": [0.1, 0.2, ...]},
    {"index": 1, "data": [0.3, 0.4, ...]}
  ]
}
Node Lifecycle
Startup — generation of RSA (PeerID) + Ed25519 (signature) + X25519 (E2E)

State loading — from state_nodeX.json

Network connection — mDNS + DHT + bootstrap

Layer synchronization — gossip protocol

Message exchange — forwarding

Learning — each node learns locally

Fault Tolerance
No single point of failure — any node can fail, the network continues to work

Self-healing — if a relay peer fails, neighbors take over its duties (planned)

Autonomy — mDNS works without internet

Security
Encryption: Noise (built into libp2p)

Authentication: PeerID — RSA (until v2.0.6), Ed25519 — signature (v1.24)

E2E encryption: X25519 + box.Seal (v1.24)

Sybil attack protection: ethical hash + node weight

Current Version Limitations
DHT has not been tested in WAN — does not work in Docker behind NAT

Gossip for 3 nodes = broadcast — the effect will be visible at 10+ nodes

Onion Routing (full) — planned. Sphinx (v2.0.1) — implemented.