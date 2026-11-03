# ISOTOPE Security Policy

## Report a Vulnerability

If you discover a vulnerability in ISOTOPE, please:

1. **Do not publish it.** Do not create a public issue.
2. Send a description to: **keeper@isotope.network**
3. Encrypt the message with our PGP key (request the key by email)
4. We will respond within 72 hours

---

## Supported Versions

| Version | Security Support |
|---------|-----------------|
| v2.0.6  | Full            |
| v2.0.x  | Critical fixes  |
| < v2.0  | Not supported   |

---

## What We Consider a Vulnerability

- Leakage of user IP address contrary to anonymity settings
- Ability to read others' messages (E2E violation)
- Bypassing the ethical filter
- Attack on bootstrap nodes
- De-anonymization through metadata
- DoS attacks on the network
- DHT poisoning

## What Is NOT a Vulnerability

- Social engineering
- Physical access to the device
- Attacks on user infrastructure (compromised router, malware)
- Self-compiled binaries with modified code
- Installing a malicious node (the network is protected by the ethical hash)

---

## Processing Procedure

1. You submit a report
2. We confirm receipt
3. We analyze and reproduce
4. We release a fix
5. We publish an advisory with thanks (if you agree)

---

## Reward

ISOTOPE is an open-source project without commercial funding. We do not offer bug bounties. But we publicly thank researchers who help us.

---

## Cryptography

ISOTOPE uses:

- **Ed25519** — for signing messages and node keys
- **X25519** — for E2E encryption (box.Seal)
- **SHA-256** — for hashing and identifiers
- **Noise** — for P2P connection encryption (built into libp2p)
- **E2E encryption** — implemented (v1.24)
- **Sphinx routing** — implemented (v2.0.1)
- **Mixing** — implemented (v2.0.2)
- **Masking** — implemented (v2.0.4)

We do not invent our own cryptography. We use proven standards.

---

## Security Layers

| Layer | What is protected | Status |
|-------|-------------------|--------|
| Obfuscation | Traffic masking | ✅ v1.13 |
| E2E encryption | Message content (from relay) | ✅ v1.24 |
| Signature | Sender authenticity | ✅ v1.24 |
| Sphinx | Route hidden from relays | ✅ v2.0.1 |
| Mixing | Timing hidden | ✅ v2.0.2 |
| Masking | Type, volume, pattern hidden | ✅ v2.0.4 |
| Metadata (full) | Who talks to whom | 🔜 v2.0.6+ (adaptive masking) |
| Traffic patterns (full) | Timing, volume | 🔜 Padding, mixing |

**What works now:** message content is protected from relay and from interception. Sender authenticity is verified. Relay sees only ciphertext. Route is hidden from relays (Sphinx). Timing is hidden (Mixing). Type, volume, and pattern are hidden (Masking).

**What comes later:** adaptive masking (20+ strategies, Reality, fallback), full metadata protection.

---

## Known Limitations

- Metadata at relay: if Sphinx goes through a relay (circuit, VPS), the relay sees the fact of connection (who with whom, when, how much). Sphinx protects from relays inside the chain, but not from transport.
- Global observer: if an observer sees all traffic (national DPI, ISP), they can correlate timings, volumes, patterns. Mixing and masking reduce the risk but do not eliminate it completely.
- Compromised device: if the phone is infected (root, Accessibility, keylogger), no application will protect. FLAG_SECURE and TTL — partially. Keylogger Protection — planned (v2.1.0).
- ISP blocking: currently ISOTOPE traffic is visible to DPI ([SPHINX] prefix, plain WS 9001). The ISP can block. Adaptive masking — planned (grant).
- Device loss / app removal: seed phrase recovers PeerID and X25519. Not contacts and not message history. Contacts — separately (backup — backlog). Message history — only on the phone.
- Scale: currently 4 phones. Immunity (15+) does not activate. DHT (15+) — planned.

---

**Security is not a feature. It is the foundation of ISOTOPE.**