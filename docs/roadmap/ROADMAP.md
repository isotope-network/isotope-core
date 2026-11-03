# ISOTOPE Roadmap

## Current Status

**Version:** v2.0.6 (seed phrase, Keystore, localization)

Core — library. Sphinx routing. Mixing. Masking. Voice/photo/files. Seed phrase. Android Keystore. i18n (en/ru). `node.go` split. Single source of truth. Offline queue. Notifications. Permissions per-action. WakeLock.

---

## Completed

### Core (v1.0–v1.18)

- P2P: libp2p + mDNS + DHT + Gossip
- Priority Gossip
- Associative memory
- WebSocket + TLS
- Obfuscation AES-GCM
- Voice steganography
- Onion Routing v2 (legacy)
- TTL
- Local encryption
- Self-healing
- Replication
- Self-adaptation
- Channels
- Weight model
- Emergent trust
- Ethical hash
- Neural network

### Refactoring (v1.18.1)

- package main → package core

### Mobile Version (v1.19.0)

- libp2p via FFI
- Stable PeerID
- Network change handling
- NSD discovery

### Mobile Stabilization (v1.21.0)

- Unified message ID
- Non-blocking Mobile calls
- Reconnect loop
- Unread badge

### Foreground Service (v1.22.0)

- Foreground Service
- Battery Optimization Whitelist
- Display peers via libp2p

### Relay-circuit (v1.23.0)

- Multi-address ANNOUNCE
- Relay-circuit
- Flush on reconnect
- Backoff reconnect
- Parallel dial
- TTL 5→8 min

### E2E Encryption (v1.24.0)

- X25519 + Ed25519 keys
- box.Seal, Version=2
- Ed25519 signature
- isotope_contacts.json
- Recipient, address-based routing

### UI 1.1–1.3 (v1.25.0)

- Terminology
- Single entry «Add contact»
- Empty state

### Stability and Statuses (v1.26.0)

- PlainText
- Auto-cleanup
- Self-QR blocked
- Relay: refresh + backoff
- Contact protocol (Stage 5)
- Onboarding (1.4)
- Ed25519 signature (4.4)
- Statuses (1.5, 1.5a)
- Send timer
- Settings UI
- Refactoring D
- VPS skip DHT/reconnectLoop

### Contact Protocol and Names (v1.27.0)

- Bootstrap-handshake
- tempContacts
- Push via messageHook
- Symmetry: confirmed
- Name system: Name / RemoteName / PeerID
- MyDisplayName
- Dialog «How should we introduce you?»

### TTL (v1.28.0)

- Periods: 10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never
- Modes: hard / after_read
- [TTL_UPDATE] (Type=8)
- Fallback 48 h
- FLAG_SECURE
- Contact deletion: quiet refusal
- Per-chat / per-peer
- Smart time format

### Single Source of Truth (v1.29.0)

- Single source of names
- Batch [READ] — Message.Refs
- Message.Status (migrated from MessageStatus)
- Message.ReadLocally
- Offline queue: pending until [DELIVERED]
- Multiline input

### Notifications and WakeLock (v1.30.0)

- System notifications (Message.SenderName, nameForPeer)
- Channel isotope_messages, importance HIGH
- Tap → Intent → MethodChannel → Dart
- Permissions per-action
- permission_service.dart
- CAMERA in manifest explicitly
- Battery — dialog with instruction
- SnackBar on denial
- ShowNotificationContent (Settings)
- WakeLock (PARTIAL_WAKE_LOCK)
- Offline queue fix (non-service only)

### Voice, Files, Photo (v1.31.0)

- Voice: recording, E2E, sending, player, persistence
- Files: 64 KB chunks, assembly, progress, batch [DELIVERED]
- Photo: compression, thumbnail, fullscreen

### File Transfer (v1.32.0)

- 64 KB chunks, backpressure, retry, idempotency
- Resume after restart
- O(1) indexes (1A), internal ack (1B)
- fallback WS↔TCP, gzip state (22x), getFilePath cache
- TypeFileComplete = 9 (2A)
- WAKE_LOCK permission

### Reliability and Transport (v1.33.0)

- O(1) indexes, FileAck, fallback WS↔TCP, gzip, cache
- TypeFileComplete, semaphore on relay
- Bugs 3–7 closed

### File Checkmarks (v1.34.0)

- 5 states, 4 icons
- [READ] via mediaID
- Progress in hidden mode

### Preview and Shade (v1.35.0)

- Progress in contacts, shade (3 toggles)
- Receiver progress
- Preview image/video
- unified filesDir

### Sphinx Routing (v2.0.1)

- Layered encryption
- Relay knows only next hop (full peer.ID)
- VPS not in chain (3 trusted contacts)
- Payload with length (4 bytes)

### Mixing (v2.0.2, v2.0.3)

- Flag in Sphinx packet (message property)
- Relay sleeps exponential (mean 5, max 30)
- Text and files

### Masking (v2.0.4)

- All packets 64 KB
- Text batch (10 = 1 packet)
- Dummy (10–120 sec, ×5 in background)
- Chunk 47 KB
- Double base64 removed for files

### Voice/Photo via Chunks (v2.0.5)

- isChunkedMediaType (file/voice/photo)
- SendVoiceByPath, SendPhotoByPath
- retryInfo.mediaType
- _VoicePlayer / _PhotoView with localFilePath

### Seed Phrase, Keystore, i18n (v2.0.6)

- BIP39 seed → Ed25519 + X25519
- Android Keystore (CBC, NoPadding)
- i18n en/ru (key screens)
- node.go split (198 KB → 35 KB + 8 files)

---

## NEXT STEP — BACKLOG

**Priority 1 (critical):**

- File acceptance consent (right of refusal) — architecture agreed
- «Save» for files (MediaStore/SAF)
- «Forward» — send received/saved file to another contact
- App backup (`isotope_media/` + state + contacts + settings)

**Priority 2 (important):**

- File queue order — sequential
- pendingDelivered — limit
- Heap 114 MB at start — lazy loading

**Priority 3 (minor):**

- `_fileLastChunkId`, `last_chunk_id` — remove
- sendReplicaToPeer — log only for text

**Priority 4 (infrastructure):**

- Samsung Android 10 — crash (deferred indefinitely)
- DHT Provide — falls (few peers, 15+)
- ANNOUNCE signal after seed recovery + contacts backup + replication

---

## In Progress / Upcoming

### Priority 1 (next step)

- 🔜 File acceptance consent (right of refusal)
- 🔜 «Save» for files
- 🔜 «Forward»
- 🔜 App backup

### Priority 2

- 🔜 File queue order
- 🔜 pendingDelivered — limit
- 🔜 Heap 114 MB at start

### Priority 3

- 🔜 `_fileLastChunkId`, `last_chunk_id` — remove
- 🔜 sendReplicaToPeer — log only for text

### Priority 4 (v2.1+)

- 🔜 Adaptive masking (20+ strategies, Reality, fallback)
- 🔜 BLE + Wi-Fi Direct (offline communication)
- 🔜 15+ node tests
- 🔜 ANNOUNCE signal + contacts backup
- 🔜 Keylogger Protection (SecureKeyboard)
- 🔜 DHT on mobile (at 15+ nodes)

---

## Future Versions

### v2.1 — Offline Communication and Adaptive Masking

- BLE + Wi-Fi Direct
- Adaptive masking (20+ strategies, Reality, fallback)
- 15+ node tests
- ANNOUNCE signal + contacts backup

### v2.5 — Ethical Hash Deeper

- Multilingual ethical hash
- Local weight
- Seed phrase (PeerID recovery)
- Emergent trust deeper

### v3.0 — ISOTOPE AI Mesh

- Distributed AI inference
- Swarm Inference
- Semantic routing
- Federated learning
- Anonymous data collection
- Ethical passport of models
- **Intelligence without an owner**

### v4.0 — Full Autonomy

- Network without internet
- Mesh communities (isotope.zone)
- Self-organization without bootstrap
- Full immunity (100+ nodes)

---

## Security Layers

| Layer | What is protected | Status |
|-------|-------------------|--------|
| Obfuscation | Traffic masking | ✅ v1.13 |
| E2E encryption | Content | ✅ v1.24 |
| Signature | Authenticity | ✅ v1.24 |
| Sphinx | Route hidden from relay | ✅ v2.0.1 |
| Mixing | Timing hidden | ✅ v2.0.2 |
| Masking | Type, volume, pattern hidden | ✅ v2.0.4 |
| Metadata (full) | Who talks to whom | 🔜 v2.0.6+ (adaptive masking) |
| Traffic patterns (full) | Timing, volume | 🔜 Padding, mixing |

---

## Deferred

- BLE — unstable (planned, v2.1)
- IPFS for website — blocked
- Samsung Android 10 — crash (deferred indefinitely)
- DHT Provide — with few peers (15+)
- TTL for tempContacts (5 min) — rejected by Keeper
- [PROFILE_UPDATE] — rejected (phishing)
- 6-10 minutes Xiaomi connection — solved by fallback WS↔TCP (temporary)

---

## Infrastructure

### VPS bootstrap/relay

- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap: /ip4/186.246.31.176/tcp/9000/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi (TCP; WS 9001 — fallback, blocked by Gorodok provider; fallback automatic)
- Ports: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- Role: temporary infrastructure

### Phones (test)

- Current PeerIDs — in handoff (change on app removal).

### VPS Update

```bash
pkill -f isotope-node
sleep 1
pgrep -f isotope-node          # empty
cd /root/isotope-core && git pull
cd /root/isotope-core/node && go build -o isotope-node ./main
nohup env NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true ./isotope-node > isotope.log 2>&1 &
sleep 3
pgrep -f isotope-node          # one, new
curl -s http://127.0.0.1:8081/status
Do NOT delete /root/isotope/state/.

Build .aar
bash
cd /d D:\isotope\node
del isotope.aar
gomobile bind -target=android -androidapi 21 -ldflags "-checklinkname=0" -o isotope.aar ./mobile
copy /y isotope.aar D:\isotope\mobile\android\app\libs\
Build APK
bash
cd /d D:\isotope\mobile
flutter build apk --debug
Order: edit → check → commit → CI → .aar (if Go) → APK → install → VPS → test.

Strategy
E2E and signatures — v1.24 (closed)

UI and stability — v1.25–v1.26 (closed)

Contacts and identification — v1.27 (closed)

Right to be forgotten — v1.28 (closed)

Single source of truth — v1.29 (closed)

Notifications and WakeLock — v1.30 (closed)

Voice, files, photo — v1.31 (closed)

File transfer — v1.32 (closed)

Reliability and transport — v1.33 (closed)

File checkmarks — v1.34 (closed)

Preview and shade — v1.35 (closed)

Sphinx — v2.0.1 (closed)

Mixing — v2.0.2–v2.0.3 (closed)

Masking — v2.0.4 (closed)

Voice/photo via chunks — v2.0.5 (closed)

Seed, Keystore, i18n — v2.0.6 (closed)

Backlog (save, forward, backup, file acceptance) — now

Adaptive masking, BLE, 15+ tests — v2.1

AI Mesh — v3.0

Full autonomy — v4.0

Each stage is a new level of decentralization. VPS is turned off when DHT and hole punching cover its role.