# ISOTOPE Security Policy

## Report a Vulnerability

If you discover a vulnerability in ISOTOPE, please:

1. **Do not publish it.** Do not create a public issue.
2. Send a description to: **security@isotope.network**
3. Encrypt the message with our PGP key (request the key by email)
4. We will respond within 72 hours

---

## Supported Versions

| Version | Security Support |
|---------|-----------------|
| v1.5    | Full            |
| v1.4    | Critical fixes  |
| < v1.4  | Not supported   |

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

ISOTOPE is an open-source project without commercial funding. We do not offer bug bounties. 
But we publicly thank researchers who help us.

---

## Cryptography

ISOTOPE uses:

- **Ed25519** — for signing messages and node keys
- **SHA-256** — for hashing and identifiers
- **Noise** — for P2P connection encryption (built into libp2p)
- **E2E encryption** — planned

We do not invent our own cryptography. We use proven standards.

---

**Security is not a feature. It is the foundation of ISOTOPE.**