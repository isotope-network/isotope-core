# FAQ — Expert Questions

Honest answers to difficult questions. No fluff, with numbers, without embellishment.

---

## Cryptographer

**1. Your ethical hash is a SHA-256 of a string. Anyone who knows the string can compute the hash. That's not protection, it's just a password.**

Correct. The hash is not cryptographic protection but an access key. Like a Wi-Fi password. It's needed so that random people don't join the network. The real protection lies in decentralization and open code. Compromising one node does not expose the others.

**2. Where are the message signatures? Right now anyone can forge a senderID.**

As of v1.24 (E2E) — full Ed25519 signature for every message. `signature` in the QR, verification during exchange. `senderID` is taken from the libp2p connection, which is already authenticated.

**3. Why RSA keys instead of Ed25519? RSA is slow.**

PeerID is RSA (for compatibility with libp2p defaults). Signing is Ed25519 (v1.24). E2E encryption is X25519 (v1.24). Planned (v2.0.6+) — migrating PeerID to Ed25519 (seed phrase).

---

## Network Engineer

**4. Your gossip with TTL=3 is a flood. With 1000 nodes, a message will circulate in loops.**

Yes, the current gossip is naive. For 1000+ nodes, gossipsub is needed (already in libp2p). We deliberately started simple to debug the core. Gossipsub is planned.

**5. mDNS only works within a single subnet. On the internet it's useless.**

Correct. mDNS is for the local network. For the global network — DHT. Both work in parallel. mDNS provides connectivity without the internet, DHT — global discovery.

**6. WebSocket+TLS is not obfuscation. DPI sees the WebSocket handshake.**

WebSocket over TLS looks like regular HTTPS. Yes, DPI can analyze traffic more deeply, but that's already expensive and doesn't scale. Full obfuscation requires adaptive masking (Reality, 20+ strategies) — in development.

**7. Replication to 2 nodes is too little. If both fail, data is lost.**

Yes, this is minimal replication. The replication factor will grow as the network grows. With 50+ nodes, replication to 5-7 nodes becomes possible. Plus, replicas are stored only by nodes with weight > 0.5, which reduces the risk of loss.

**8. Self-adaptation every 5 minutes is slow. With a sharp load increase, the network won't keep up.**

For the current scale (4 nodes) — it's sufficient. The adaptation interval is itself adaptive: during metric anomalies, the check frequency increases. This is built into the auto-correction rules.

---

## ML Engineer

**9. 100-dimensional vectors with bigrams are not semantics. It's letter statistics. You don't understand the meaning of words.**

Honestly — yes. The current ethical engine distinguishes "kill" from "help" at the level of n-grams, not meaning. True understanding of meaning will come with AI Mesh, when BERT-level models are loaded onto the nodes. Right now we have a fast filter, not deep analysis.

**10. Your "learning" is just gradient descent on a single layer. That's not learning, it's weight fitting.**

True for the current version. One layer, adaptive lr. Full federated learning — v3.0. Right now we're verifying that the architecture itself works: weights update, synchronize, and influence filtering.

**11. Why 100 neurons? That's 4 orders of magnitude less than any modern model.**

100 dimensions is sufficient for bigram classification. Full models will run on AI Mesh nodes. The ethical engine is a fuse, not a brain.

---

## Lawyer

**12. Your ethical hash is someone's values imposed on everyone. In some countries this could be recognized as extremism.**

The hash is universal: don't kill, don't steal, don't lie. This is present in the laws of all countries. The formulations are taken from world religions, not from political declarations. Plus — anyone can fork the network with their own hash.

**13. Users are anonymous. If crimes are coordinated through ISOTOPE — who is responsible?**

No one. As with Tor, Bitcoin, Telegram. The technology is neutral. Our ethical filtering reduces risks but does not eliminate them entirely. Responsibility for use lies with the user.

**14. You don't store data, but you are the creator. A court could force you to change the code.**

A court could force me to change the code in my repository. But users have already downloaded the current version. Forks will continue to work. Decentralization is stronger than any court.

**15. The weighted access model is discrimination. New nodes have no rights.**

New nodes have basic rights: read, write. Rights grow with reputation — this is not discrimination but protection against spam and bots. In the real world, you don't give a stranger the keys to your apartment on the first day either.

---

## Business Angel

**16. What is your TAM/SAM/SOM? Without numbers, this is a hobby.**

The messenger market is $100+ billion. B2B data exchange — $80+ billion. AI inference — $50+ billion and growing. SAM — currently zero. We are at the pre-seed stage. Numbers will appear after the first Enterprise contracts.

**17. Who are your competitors? Signal? Telegram? Tox?**

No direct competitors. Signal is centralized. Telegram is censorable. Tox has no moderation. Briar, Session, Cwtch, Nostr are P2P but without an immune system. We are the only messenger with ethical immunity. In B2B, competitors are Hyperledger, Corda, but they are blockchain, while we are P2P with fading memory.

**18. What is your go-to-market? Without a plan, there's no investment.**

First the Community vector: the messenger as a demonstration. Then Enterprise: a consortium of banks (E0). Tokenomics — after 10,000 users. The plan exists, but it's at an early stage.

**19. Channels with weight levels — is that censorship? A newcomer sees only 30% of the content.**

No. It's protection against spam. A newcomer sees a teaser. Once they build weight, they get full access. In the real world, you don't get everything at once either: first an internship, then access to classified materials.

---

## Human Rights Advocate

**20. Your ethical filter can make mistakes. A good message can be filtered out. That's censorship of errors.**

Yes, it can. False positives are possible. But the user always sees their preHash/antiHash and can adjust them. Plus, likes/dislikes allow the community to correct the network. This is not a perfect filter but an adaptive one.

**21. Is anonymity in ISOTOPE protection for dissidents or for terrorists?**

For both. As with any technology. But the ethical hash cuts off obvious evil, and the community — the debatable. We cannot guarantee that ISOTOPE won't be used for harm. We can guarantee that it was not created for that.

**22. Why don't you work with human rights organizations?**

We do. Open code, documentation, the philosophical layer — all publicly available. Direct cooperation is possible once the project becomes more stable. Right now we are at the core stage.

---

## Philosopher

**23. You claim that technology can distinguish good from evil. That's naive. Good and evil are contextual.**

We do not claim that technology determines good and evil. We claim that the community, through likes/dislikes and the ethical hash, forms a collective representation. Technology does not judge — it reflects the judgment of people.

**24. ISOTOPE is a utopia. People are not ready for such freedom.**

People are not ready for freedom until they try it. We provide a tool. Whether to use it or not is everyone's choice.