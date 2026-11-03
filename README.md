# ISOTOPE — the immune system of communication

**Ask without revealing. Answer without exposing.**

ISOTOPE is an infrastructure that unites
data exchange, distributed AI, and human communication
in a single decentralized network.

---

## Not Tor. Not Signal. Not Telegram.

**Tor** hides who you are. ISOTOPE hides what you say.

**Signal** protects the content. ISOTOPE protects the fact of conversation itself.

**Telegram** gives convenience at a price. ISOTOPE gives freedom without a price.

ISOTOPE is not a messenger. Not a protocol. Not a business.
It is an infrastructure where a person is not a user, but a node.
Where trust is not verification, but interaction.
Where freedom is not a promise, but architecture.

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

## Seven Commandments

At the heart of ISOTOPE lies the **ethical hash** — a digital embodiment of values.
Seven universal commandments:

1. Do no harm.
2. Do not steal.
3. Do not deceive.
4. Do not betray.
5. Do not be indifferent.
6. Create, do not destroy.
7. Treat others as you would yourself.

These principles are gathered from all the great teachings of humanity.
They belong to no single religion, no single ideology.
They are universal.

---

## How Immunity Works

Every message is **compared to the reference**.
Not by words — **by meaning**.
The neural network converts text into a **100-dimensional vector**.
The **closer** to the reference — the **longer** the message lives.
The **farther** — the **faster** it fades.

**This is not censorship. This is an immune system.**

Likes raise the weight. Dislikes lower it.
Time erodes even strong signals.
When the weight falls below the threshold — the message goes to the archive.
Below — it is deleted forever.
The network **breathes**: the important remains, the random leaves, the harmful is rejected.

---

## Spirituality and Technology — Together

We are used to separating: this is technology, and this is the soul.
Algorithms are cold. Prayer is warm.
**We refuse this separation.**

ISOTOPE is an attempt to **embed spiritual principles into the digital world**.
Not as a superstructure. Not as a "company mission" on a website.
But as **working code**.

**Love** is not an abstraction. It is the **direction of the vector toward the reference**.
**Compassion** is not a slogan. It is the **cosine closeness between meanings**.
**Good** is not a philosophy. It is the **weight with which a message enters the network**.

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
The network breathes: what matters stays,
the random goes, the harmful is rejected.

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

**Right to Be Forgotten.**
Messages are not «deleted». They are released.
Like releasing the past — without regret.
Forgetting is not loss. It is liberation.

**Right to Silence.**
Short TTL (10 sec – 1 min) — screenshots forbidden.
It is not «I don't want you to see».
It is «I want this to stay between us and disappear».

**Quiet Refusal.**
Deleting a contact is not blocking. It is quiet refusal.
Blocking is coercion. Silence is freedom.

**Single Source of Truth.**
If a field can be derived from Message — it lives in Message.
Status, ReadLocally, Refs — all in Message.
No second sources. No drift.

**Offline Queue — Sender's Responsibility.**
VPS does not buffer. The sender holds pending until [DELIVERED].
When the recipient appears, the sender resends.
VPS is a courier, not a storage.
Service messages are not queued — only TypeMessage.

**Migrations Are Mandatory.**
Old data is never lost.
messageStatus → Message.Status.
_readSent → ReadLocally.
Log every migration.

**Permissions — Per Action.**
Not all at once at startup.
Camera — when scanning QR.
Bluetooth and location — when searching nearby.
Notifications — at startup.
Context is understanding.

**Seed Phrase — Identity.**
PeerID is recoverable from a seed phrase (BIP39).
Android Keystore — seed survives restart.
But contacts and history — separately. Seed is identity, not «recover everything».

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
| Sphinx | Route hidden from relays | ✅ v2.0.1 |
| Mixing | Timing hidden | ✅ v2.0.2 |
| Masking | Type, volume, pattern hidden | ✅ v2.0.4 |
| Metadata (full) | Who talks to whom | 🔜 v2.0.6+ (adaptive masking) |
| Traffic patterns (full) | Timing, volume | 🔜 Padding, mixing |

**What works now:** message content is protected from relay and from interception. Sender authenticity is verified. Relay sees only ciphertext. Route is hidden from relays (Sphinx). Timing is hidden (Mixing). Type, volume, and pattern are hidden (Masking).

**What comes later:** adaptive masking (20+ strategies, Reality, fallback), full metadata protection.

---

## What's new in v2.0.0–v2.0.6

### v2.0.1 — Sphinx routing
- Layered encryption. Each relay removes its layer.
- Relay knows only the next hop (full peer.ID). Does not know sender or receiver.
- VPS is not in the chain (when 3 trusted contacts).
- Payload with length (4 bytes). Fixed packet size.

### v2.0.2 — Mixing (text)
- Flag in Sphinx packet — a property of the message, not a relay setting.
- Relay sleeps exponential time (mean 5 sec, max 30 sec).
- Works for text.

### v2.0.3 — Mixing for files
- File chunks via Sphinx + mixing.
- Semaphore (2–3). Retry via Sphinx. SphinxTimeout 15 sec.

### v2.0.4 — Masking
- All packets 64 KB. Text, file, photo, voice — indistinguishable.
- Text batching (10 texts = 1 packet). Traffic ×1.4, not ×14.
- Dummy traffic at random intervals (10–120 sec, ×5 in background). Receiver drops.
- Chunk 47 KB to fit in 64 KB payload.
- Double base64 removed for files.

### v2.0.5 — Voice and photo via chunks
- `isChunkedMediaType` (file/voice/photo).
- `SendVoiceByPath`, `SendPhotoByPath` (chunks).
- `retryInfo.mediaType`.
- `_VoicePlayer` / `_PhotoView` with `localFilePath`.

### v2.0.6 — Seed phrase + Android Keystore + i18n
- BIP39 seed (12/24 words) → Ed25519 (PeerID) + X25519 (encryption). Deterministic.
- Android Keystore — seed stored in hardware-backed storage. Survives restart.
- Explicit — dialog on first launch. Not automatic.
- i18n: English / Russian. Key screens translated.
- `node.go` split (198 KB → 35 KB + 8 files).

---

## What was already there (v1.x)

- P2P network: libp2p + mDNS + DHT + Gossip
- Priority Gossip
- Associative memory
- WebSocket + TLS
- Obfuscation: AES-GCM + random delays
- Voice steganography: LSB in WAV
- Onion Routing v2 (legacy)
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
- tempContacts
- Push events to UI via messageHook (not polling)
- Symmetry: both sides confirmed: true

**Identity and names (v1.27):**
- Name (local) / RemoteName (contact's representation) / PeerID (fallback)
- MyDisplayName in Settings
- QR contains display_name
- Dialog «How should we introduce you?» when sending request
- Profile → «Your name»
- Long tap on contact → Open / Rename / Delete
- Security warning on request

**Message statuses (v1.26):**
- ✓ / ✓✓ / ✓🔒 / ✓✓ (colored)
- Hidden is terminal
- read_enabled — symmetric

**Send timer (v1.26):**
- 0/3/5/10 sec delay
- Draft on back
- Cancel button

**TTL — Right to Be Forgotten (v1.28):**
- Periods: 10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never
- Modes: hard / after_read
- Fallback 48 hours
- [TTL_UPDATE] (Type=8) — auto-hard notification
- DeleteExpired in cleanupLoop (1 min timer)
- FLAG_SECURE: TTL 10s–1m — screenshots forbidden

**Contact deletion (v1.28):**
- RemoveContact — full cleanup
- isotope_deleted.json — deleted don't return
- Quiet refusal — B doesn't know

**Per-chat / per-peer (v1.28):**
- Messages per chat
- Drafts per chat
- Unread per chat
- [READ] only for current chat

**UI (v1.28):**
- Settings → Messages (TTL: periods + modes)
- Settings → Privacy (read_enabled)
- Profile (MyDisplayName)
- Orange periods 10s / 30s / 1m
- Smart time format (today / yesterday / date)
- Send timer (pending message)
- AppBar — clean, only contact name

**Single source of truth (v1.29):**
- Single source of names — chatProvider.nameFor(peerID)
- Batch [READ] — Message.Refs []string + fallback on Ref
- Message.Status (migrated from MessageStatus)
- Message.ReadLocally — single source of unread
- _readSent removed. Migration: _readSent → markReadLocally, flag read_sent_migrated
- Multiline input

**Offline queue (v1.29):**
- Pending until [DELIVERED]
- replicateMessage — always enqueuePending for addressed
- flushPending — only resends, doesn't remove
- removePendingByRef(ref) — on [DELIVERED]
- Triggers: ConnectedF (Notifiee) + announceLoop (4 min)

**Offline queue fix (v1.30):**
- enqueuePending — only for non-service (!isServiceType)
- Reason: service messages have no [DELIVERED] → infinite handshake loop

**Notifications (v1.30):**
- Message.SenderName
- nameForPeer(peerID) — priority: own → MyDisplayName; contact → Name → RemoteName; else empty
- Kotlin onMessage — if !isForeground → showMessageNotification
- Channel isotope_messages, importance HIGH
- Tap → Intent EXTRA_PEER_ID → onNewIntent → MethodChannel → Dart openChatStream

**Permissions per-action (v1.30):**
- permission_service.dart
- ActivityCompat.requestPermissions in Kotlin
- POST_NOTIFICATIONS — at startup
- CAMERA — when scanning QR
- BLUETOOTH_*, FINE_LOCATION, NEARBY_WIFI_DEVICES — when searching nearby
- Battery — dialog with instruction
- SnackBar on denial — «Settings» button
- AndroidManifest.xml — CAMERA added explicitly

**ShowNotificationContent (v1.30):**
- Settings.ShowNotificationContent bool
- Default true
- Privacy toggle

**WakeLock (v1.30):**
- IsotopeService.kt — PARTIAL_WAKE_LOCK (isotope:network)
- Go routines of libp2p don't freeze in background

**Voice, files, photo (v1.31):**
- MediaType (string), MediaData, FileName, FileSize, Duration, MediaID, ChunkIndex, ChunkTotal
- Opus 24 kbps, 5 min ≈ 900 KB
- Chunks 64 KB, batch [DELIVERED] (500 ms / 10 refs)
- Previews «🎤 Voice message», «📎 file.txt», «📷 Photo»

**File transfer (v1.32):**
- Chunks 64 KB, backpressure, retry, idempotency
- Resume after restart
- O(1) indexes (1A), internal ack (1B)
- fallback WS↔TCP, gzip state (22x), cache getFilePath
- TypeFileComplete = 9 (2A)

**Reliability and transport (v1.33):**
- O(1) indexes, FileAck, fallback WS↔TCP, gzip, cache
- TypeFileComplete, semaphore on relay
- Bugs 3–7 closed

**File checkmarks (v1.34):**
- 5 states, 4 icons
- [READ] via mediaID
- Progress in hidden mode

**Preview and shade (v1.35):**
- Progress in contacts, shade (3 toggles)
- Receiver progress
- Preview image/video
- unified filesDir

**Deferred:**
- BLE — unstable
- Samsung Android 10 — crash (J4). A03 Core (Android 11 Go) — works.
- DHT Provide — falls with few peers (needs 15+)
- Heap 114 MB at start — lazy loading (not critical)

**In development (grant):**
- Adaptive masking (20+ strategies, Reality, fallback)
- BLE + Wi-Fi Direct (offline communication)
- Immune system: 15+ node tests
- ANNOUNCE signal after seed recovery + contacts backup + network replication

[Roadmap →](docs/roadmap/ROADMAP.md)

---

## Quick Start

### Requirements
- Docker and Docker Compose
- Go 1.21+
- Flutter 3.x

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

```go
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
Repository Architecture
isotope-core/
├── node/ # Go core (P2P, neural network, memory, API)
│ ├── *.go # package core (library)
│ ├── main/ # entry point (package main)
│ └── mobile/ # gomobile binding
├── mobile/ # Flutter app
├── tests/ # Autotests
├── docs/ # Documentation and philosophy
├── genesis/ # Ethical hash
└── docker-compose.yml

Architecture details →

Documentation
Manifesto

The Story

Applications

Use Cases

Immunity Scale

Weight Access Model

FAQ — Expert Questions

Contributing

Code of Conduct

Security

Mirrors

License
Core: AGPL v3

Commercial license: LICENSE.COMMERCIAL.md

Contacts
Website: isotope.network

Zone: isotope.zone

GitHub: github.com/isotope-network

Email: keeper@isotope.network

ISOTOPE is an infrastructure.
Data. AI. People.