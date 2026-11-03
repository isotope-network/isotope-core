ISOTOPE Architecture
Overview
ISOTOPE is infrastructure for ethical, resilient, self-learning data and AI exchange.

text
[User] ←→ [Messenger / API] ←→ [ISOTOPE Node] ←→ P2P network ←→ [Other nodes]
                        │                    │
                  [Neural net]         [AI model]
                  [Memory]             [Ethical passport]
                  [Ethical engine]     [Inference]
Core structure (v2.0.6)
The ISOTOPE core is a library (package core).

text
node/
├── *.go              # package core (core)
├── main/main.go      # package main (desktop entry point)
└── mobile/mobile.go  # package mobile (gomobile wrapper)
v2.0.6 split: node.go (198 KB → 35 KB + 8 files):

node.go — core: Node struct, NewNode, InitP2P, Stop, StartMobile, StartHTTP, GetStatus, sendToRecipient, fallbackSend, offline queue, loadState.

node_p2p.go — handleStream, ping, reconnect, announceLoop, cleanup, processMessageInternal, Sphinx receiver.

node_e2e.go — E2E keys, encryption, QR, SetSeedPhrase, GetSeedPhrase.

node_files.go — chunkWorker, retryUnacked, pushFileStatus, GetChunk, FileAck, DeleteFile, GetFileStatuses.

node_service.go — service messages, statuses, batches, TTL.

node_contacts.go — contacts, temp, requests.

node_announce.go — ANNOUNCE, FIND, relay, Multiaddrs, Connect.

node_settings.go — settings, deleted, SendMessage/SendToPeer.

node_batch.go — batch, dummy, obfuscation, randomDelay.

node/seed.go — BIP39 → Ed25519 + X25519.

node/sphinx.go — Sphinx (build/process/extract), sleepExponential, flags, SphinxPayloadSize.

node/memory.go — Message, Memory, byID, fileChunks.

node/state.go — saveState, loadStateData, gzip.

node/contacts.go, node/settings.go, node/requests.go, node/deleted.go, node/dht.go, node/media.go, node/adapt.go.

Message structure (v2.0.6)
go
type Message struct {
    ID               string
    Recipient        string
    Sender           string
    SenderName       string        // v1.30 — for notifications
    Text             string
    PlainText        string
    TTL              int
    Priority         int
    Timestamp        int64
    Version          int           // 0 = history, 2 = E2E
    Type             MessageType   // 0 = regular, 1-9 = service
    Ref              string        // single reference
    Refs             []string      // array of references (batch)
    Status           MessageStatus // sent / delivered / read / hidden
    ReadLocally      bool          // I read the incoming
    TtlPeriodSeconds int           // 0 = never
    TtlMode          string        // "after_read" | "hard" | ""
    ExpiresAt        time.Time
    ReadEnabled      *bool
    MediaType        string        // "" | "text" | "voice" | "file" | "photo" | "batch" | "dummy"
    MediaData        string        // base64 (voice)
    FileName         string        // for files
    FileSize         int64         // for files
    Duration         int           // seconds, for voice
    MediaID          string        // for file chunks
    ChunkIndex       int           // 0, 1, 2, ...
    ChunkTotal       int           // total number of chunks
    MixingEnabled    bool          // v2.0.2 — mixing flag in Sphinx packet
}
Node methods (main, v2.0.6)
go
func (n *Node) SendMessage(text string, ttl int) (string, error)
func (n *Node) SendToPeer(recipient, text string, ttl int) (string, error)
func (n *Node) SendReadBatch(refs []string, recipient string) error
func (n *Node) MarkReadLocally(refs []string) error
func (n *Node) AddContact(...) error
func (n *Node) RemoveContact(peerID string) error
func (n *Node) RenameContact(peerID, localName string) error
func (n *Node) GetMessages() []Message
func (n *Node) GetPeers() []string
func (n *Node) nameForPeer(peerID string) string
func (n *Node) SetSeedPhrase(phrase string) error
func (n *Node) GetSeedPhrase() string
func (n *Node) SendVoiceByPath(...) error
func (n *Node) SendFileByPath(...) error
func (n *Node) SendPhotoByPath(...) error
func (n *Node) SendFileComplete(mediaID, sender string) error
func (n *Node) GetFileStatuses() string
func (n *Node) FileAck(mediaID string, indices []int) error
func (n *Node) SetMixingEnabled(enabled bool) error
func (n *Node) GetMixingEnabled() bool
func (n *Node) SetMaskingEnabled(enabled bool) error
func (n *Node) GetMaskingEnabled() bool
func (n *Node) SetDummyScreenOn(screenOn bool) error
Sphinx routing (v2.0.1)
Layered encryption. Each relay strips its own layer. A relay knows only the next hop (full peer.ID). It does not know the sender or the recipient.

SphinxHopLayerSize = 249, SphinxHeaderSize = 747.

SphinxNextPeerIDLenSize = 1, SphinxNextPeerIDSize = 64.

SphinxPayloadSize = 65536 (64 KB).

SphinxPayloadLenSize = 4 — [length 4 BE][JSON][padding].

sendViaSphinx — direct only (Wi-Fi), filter /p2p-circuit, AddAddrs, NewStream. Fallback — sendToRecipient.

handleStream (relay) — FindPeerByID + direct (symmetrical).

processSphinxPacket — peer.Decode (not peer.ID).

VPS is not in the chain (when 3 trusted contacts exist).

Mixing (v2.0.2, v2.0.3)
v2.0.2 — Mixing (text)
Flag in the Sphinx packet — a property of the message, not a relay setting.

Message.MixingEnabled.

SphinxFlagsSize = 1, SphinxFlagMixing.

buildSphinxPacket(..., mixing bool) — flags in each relay layer. MAC covers flags.

processSphinxPacket → (peer.ID, bool, []byte, error).

sleepExponential — relay sleeps during mixing (mean 5 s, max 30 s).

v2.0.3 — Mixing for files
File chunks via Sphinx + mixing.

SphinxPayloadFileSize = 131072 (128 KB, double base64).

sphinxFileSem = 2.

SphinxTimeout = 15s, write deadline 30s.

Retry via Sphinx.

Masking (v2.0.4)
All packets are 64 KB. Text, file, photo, voice — indistinguishable.

Text batching (10 texts = 1 packet). Traffic ×1.4, not ×14.

Dummy traffic:

SphinxFlagDummy (bit 1). From the sender, not the relay.

sendDummy — random interval 10–120 s (×5 in background), random contact, random text (50–200 chars), random delay (1–60 s).

Recipient — drop (not memory, not hook, not [DELIVERED]).

maskingEnabled — toggle in Privacy.

Chunk 47 KB — fits into the 64 KB payload.

Double base64 removed for files. encryptBytesForRecipient, decryptBytesFromSender.

Voice, files, photos (v1.31–v2.0.5)
MediaType (string): "" | "text" | "voice" | "file" | "photo" | "batch" | "dummy".

Voice: Opus 24 kbps, 5 min ≈ 900 KB. One chunk (MediaType = "voice"). Player: play/pause, progress bar, seek.

Files: chunks 64 KB (v1.31–v2.0.3), 47 KB (v2.0.4+). MediaID, ChunkIndex, ChunkTotal. Assembled on disk.

Photos: compression 1600px JPEG 80. Thumbnail, fullscreen with zoom.

isChunkedMediaType (file/voice/photo) — instead of == "file".

retryInfo.mediaType.

_VoicePlayer / _PhotoView with localFilePath.

Seed phrase + Android Keystore + i18n (v2.0.6)
BIP39 seed (12/24 words) → Ed25519 (PeerID) + X25519 (encryption). Deterministic.

node/seed.go — GenerateSeedPhrase, ValidateSeedPhrase, SeedToEd25519, SeedToX25519.

Android Keystore — AES-256, CBC, NoPadding, alias isotope_seed_key. Seed survives restart.

InitP2P: if seedPhrase != "" — Ed25519 identity (from seed), X25519 from seed, not loadOrGenerateE2EKeys, not loadPrivateKey (RSA). Otherwise — RSA.

i18n: LocaleProvider, l10n.yaml, app_en.arb, app_ru.arb (~250 keys). Language switcher in Settings.

Mobile wrapper (mobile.go)
Status: working (v2.0.6)

Methods (return JSON strings)

Method	Description
Start(stateFile)	Start node
Send(text, ttl)	Send (broadcast)
SendToPeer(recipient, text, ttl)	Send to contact
SendReadBatch(refs, recipient)	Batch [READ]
MarkReadLocally(refs)	Mark as read locally
GetMessages()	List of messages
GetPeers()	List of peers
GetWeight()	Node weight
GetStatus()	JSON status
GetMultiaddrs()	List of addresses
ConnectToPeer(addr)	Connect
ConnectToPeerWithFallback(addrs)	With fallback
Announce(json)	Publish addresses
FindPeerByID(peerID)	Find peer
AddContact(...)	Add contact
RemoveContact(peerID)	Remove contact
RenameContact(peerID, name)	Rename
SendContactRequest(peerID, name)	Request
AcceptRequestByID(id)	Accept
RejectRequestByID(id)	Reject
GetRequests()	List of requests
SetMyDisplayName(name)	Display name
GetMyDisplayName()	Get display name
SetTtl(period, mode)	TTL
GetTtl()	Get TTL
GetShowNotificationContent()	Notification setting
SetShowNotificationContent(bool)	Notification setting
GetMyQRData()	QR data
GenerateSeedPhrase(words)	Generate seed phrase
SetSeedPhraseBeforeStart(phrase)	Seed before Start
SetSeedPhrase(phrase)	Seed after Start
GetSeedPhrase()	Get seed
SendVoiceByPath(path, ttl)	Voice
SendFileByPath(path, ttl)	File
SendPhotoByPath(path, ttl)	Photo
SendFileComplete(mediaID, sender)	[FILE_COMPLETE]
GetFileStatuses()	File statuses
FileAck(mediaID, indicesJSON)	Internal ack
GetMixingEnabled() / SetMixingEnabled(bool)	Mixing
GetMaskingEnabled() / SetMaskingEnabled(bool)	Masking
SetDummyScreenOn(screenOn)	Dummy (screenOn)
GetNotificationSettings() / SetNotificationSettings(json)	Notification shade
E2E encryption (v1.24)
Keys
Key	Purpose	Type	File
PeerID	Identification	RSA 2048	isotope_state.json.key
Ed25519	Signature	Ed25519	isotope_state.json.ed25519.key
X25519	E2E encryption	X25519	isotope_state.json.x25519.key
QR format (v1)
json
{
  "v": 1,
  "peerID": "QmX...",
  "ed25519_pub": "base64...",
  "x25519_pub": "base64...",
  "signature": "",
  "display_name": "Peter",
  "read_enabled": true
}
Encryption
box.Seal (X25519 + XSalsa20-Poly1305).

Format: nonce(24) || ciphertext.

Version = 2 — E2E.

Version = 0 — history.

Signature
Ed25519 over peerID || x25519_pub.

verified — UI signal, not a pass.

Single source of truth (v1.29)
Principle: if a field is derivable from Message — it lives in Message.

Message.Status (v1.29)
Was: MessageStatus — a separate map in Node and State.

Now: Message.Status.

Message deleted → status goes away.

Memory.SetStatus(id, status).

Migration: on loadState, old messageStatus → Message.Status.

Message.ReadLocally (v1.29)
Was: _readSent in Dart.

Now: Message.ReadLocally bool.

Memory.MarkReadLocally(refs).

Dart: _sendReadBatchFor → markReadLocally before sendReadBatch.

Migration: _readSent → markReadLocally, flag read_sent_migrated.

Message.Refs (v1.29)
Was: [READ] one by one.

Now: Message.Refs []string.

Ref — single reference.

Refs — batch.

Backward compatibility: Refs empty → read Ref.

SendReadBatch(refs, recipient).

Offline queue (v1.29)
Problem: offline recipient. VPS does not buffer.

Solution: pending until [DELIVERED].

replicateMessage — always enqueuePending when addressed.

flushPending — does not delete, only resends.

removePendingByRef(ref) — on [DELIVERED].

Triggers: ConnectedF (Notifiee) + announceLoop (4 min).

Fix (v1.30): enqueuePending — only for non-service (!isServiceType). Reason: service messages have no [DELIVERED] → infinite handshake loop.

System notifications (v1.30)
Message.SenderName
SenderName field in Message.

Filled in 4 places where an incoming message is created.

nameForPeer(peerID) — priority: self → MyDisplayName; contact → Name → RemoteName; otherwise empty.

Kotlin onMessage
If !isForeground → showMessageNotification.

Channel isotope_messages, importance HIGH.

Tap → Intent EXTRA_PEER_ID → onNewIntent → MethodChannel → Dart openChatStream.

ShowNotificationContent
Settings.ShowNotificationContent bool.

Default true.

Privacy toggle in settings.

Per-action permissions (v1.30)
permission_service.dart — new file.

ActivityCompat.requestPermissions in Kotlin, onRequestPermissionsResult → result to Dart.

Requests:

POST_NOTIFICATIONS — at start.

CAMERA — on QR scan.

BLUETOOTH_*, FINE_LOCATION, NEARBY_WIFI_DEVICES — on "Find nearby".

Battery — dialog with instructions + openAppSettings.

On refusal: SnackBar with "Settings" button.

AndroidManifest.xml: CAMERA added explicitly.

WakeLock (v1.30)
IsotopeService.kt — PARTIAL_WAKE_LOCK (isotope:network).

acquireWakeLock in onCreate, releaseWakeLock in onDestroy.

Why: Foreground Service alone does not prevent CPU sleep. Without WakeLock, Go libp2p routines freeze in the background.

Multiline input (v1.30)
chat_screen.dart — TextField in ConstrainedBox(maxHeight: 140).

maxLines: null, minLines: 1.

keyboardType: TextInputType.multiline.

textInputAction: TextInputAction.newline.

onSubmitted removed. Sending — only via button.

Contact protocol (v1.27)
A contact is not an entry in a list. It is a mutual promise to hear. You offer. I accept. Or decline. No coercion. No imposition.

Bootstrap handshake
[CONTACT_HELLO] — open. PeerID A + public keys A.

[CONTACT_HELLO_ACK] — open. PeerID B + public keys B.

[CONTACT_REQUEST] — E2E. Full payload.

[CONTACT_ACCEPT] — E2E. Confirmation + display name B.

Transport separation
Message type	Transport
Open service (HELLO, ACK)	Bootstrap (always)
E2E (REQUEST, ACCEPT, messages)	Circuit → bootstrap fallback
tempContacts
Created on [CONTACT_HELLO] if B does not know A.

Needed only to decrypt [CONTACT_REQUEST].

Deleted after requests.Add.

Contact removal — silent refusal
Removal is not a block. It is a silent refusal. A block is coercion. Silence is freedom.

RemoveContact — full cleanup.

isotope_deleted.json — deleted contacts do not return.

B does not know about the removal.

Naming system (v1.27)
PeerID — my technical essence. RemoteName — my display name for others. Name — my name for myself.

Three levels — three freedoms: to be oneself, to be understood, to be recognized.

Two fields per contact
Field	Set by	Transmitted	Priority
Name	Me	No	Highest
RemoteName	Contact	Yes	Medium
—	—	—	Fallback: PeerID
UI: Name → RemoteName → PeerID.

Single source of names (v1.29)
_contactNames removed from connect_screen.

_displayName delegates to chatProvider.nameFor(peerID).

Message statuses (v1.26, v1.34)
Five states, four icons:

Icon	Meaning
⏳	Sending
✓	Sent
✓✓	Delivered. No read will come
✓🔒	Delivered. Recipient does not share status
✓✓ (color)	Read
read_enabled
In contact.

In Settings (my_read_enabled).

Transmitted with each message.

In QR.

hidden — terminal
Lock = "no read will come".

Does not roll back.

File checkmarks (v1.34)
Same 5 states, 4 icons as for text.

⏳ — queued.

✓ (gray) — sent.

✓✓ (two gray) — delivered + recipient has "Open" (file assembled) — only after [FILE_COMPLETE].

✓🔒 (one gray + lock) — delivered, recipient does not share read_enabled.

✓✓ (two green) — read ([READ] received).

GetFileStatuses — status = max across chunks (read > hidden > delivered > sent).

[READ] for files — via mediaID.

TTL — the right to be forgotten (v1.28)
A message is not "deleted". It is released. As the past is released — without regret. Forgetting is not loss. It is liberation.

Periods
10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never.

Default: never ("Not deleted").

Modes
"After read" — respect for the recipient. "Hard" — sender's control.

hard:

Timer from receipt.

Sender: from [DELIVERED].

Recipient: from receipt.

after_read:

Timer from read.

Both share → synchronously.

Recipient does not share → auto-hard + [TTL_UPDATE].

Sender does not share → auto-hard + [TTL_UPDATE].

Fallback 48 hours
Fallback is honesty. If the recipient does not share — the network decides for itself.

[TTL_UPDATE] (Type=8)
Version = 0.

Payload: expires_in_seconds.

On auto-hard.

FLAG_SECURE — the right to silence
A secret is protection. A long one is not a secret.

Short TTLs (10s – 1 min) — screenshot prohibition. This is the right to silence. Silence is also freedom.

Badge + preview
They live exactly as long as the message. All together.

Send timer (v1.26)
Delay 0/3/5/10 sec.

Message immediately with circular progress.

"Cancel" button.

Back — draft.

Home — timer continues.

Migrations (v1.29)
Mandatory. Old data is not lost.

messageStatus → Message.Status.

_readSent → ReadLocally.

Each migration is logged.

File transfer (v1.32–v1.33)
Chunks 64 KB (v1.31–v2.0.3), 47 KB (v2.0.4+).

Backpressure: chunkQueue (buffer 1000) + 10 chunkWorker.

sent/<mediaID>.bin — copy on disk, source for retry.

Batch [DELIVERED]: queueDelivered (2 sec / 10 refs). Idempotency: [DELIVERED] is always sent.

Retry unackedChunks: retryInfo — metadata (~200 bytes). Message restored from sent/ via fillFileChunkFromSent. Backoff up to 1 hour.

Progress — push, not pull: file_status (throttle 1 sec), file_complete (once, completedFiles).

Recovery after restart: resumeIncompleteFiles, GetFileStatuses.

O(1) indexes (v1.33): byID, fileChunks. rebuildIndexes after DeleteExpired/PurgeDead.

Internal ack (v1.33): hookAckWindow, FileAck, cleanupPendingAcksLoop (30 sec tick, 120 sec timeout).

Fallback WS↔TCP (v1.33): expandTransportAlternatives — port computed dynamically (TCP = port, WS = port+1).

gzip state (v1.33): JSON → gzip → (encrypt) → file. 24.4 MB → 1.1 MB (22x).

getFilePath cache (v1.33): MediaStorage._pathCache, invalidatePathCache.

TypeFileComplete = 9 (v1.33): isServiceType (Type 9 added), handleFileComplete, SendFileComplete.

Relay-circuit (v1.23+)
Problem: nodes behind NAT cannot accept incoming connections.

Solution: relay via VPS.

Node reserves a slot (client.Reserve).

VPS forwards, does not store.

ANNOUNCE automatically adds the relay address.

FIND fallback — relay address.

Reservation update (v1.25):

Notifiee ConnectedF — on reconnect.

relayLoop 30 seconds.

Exponential backoff.

Conditions for disabling VPS:

DHT covers 15+ nodes.

Hole punching works for most NATs.

2–3 independent relay nodes.

Immunity
Immunity is the network's ability to preserve itself without central control.

Technical immunity
Three levels of protection.

Replication: to 2+ live nodes.

Self-healing: heartbeat.

Fallback: local IP → relay.

Exponential backoff.

Social immunity
Weight from the ethical hash.

Likes/dislikes.

Time erodes.

Archive → deletion.

Unity
Technical keeps the network alive. Social keeps it clean.

Scale
Nodes	Technical	Social
2–3	Protected channel	Personal trust
5–10	Early consensus	First ratings
15–20	Immunity activates	Collective cleansing
50+	Stable immunity	Stable norms
100+	Unkillable network	Self-regulation
Key components
1. P2P network (libp2p)
Transport: TCP + WebSocket + TLS.

Discovery: mDNS + DHT + NSD.

Sync: Gossip.

Routing: Sphinx (v2.0.1).

Relay-circuit.

Masking (v2.0.4).

Reconnect, Flush.

2. Ethical engine
Vector: 100 dimensions.

Layers: every 20 messages.

Hash: 7 commandments.

3. Memory
Weighted, FIFO with archive.

Weight, aging, archive.

TTL.

AES-256-GCM encryption.

PlainText.

Status, ReadLocally.

O(1) indexes (v1.33).

4. Channels
Thresholds: full=0.3, comment=0.5, vote=0.7.

5. Security
Sphinx (v2.0.1).

Mixing (v2.0.2, v2.0.3).

Masking (v2.0.4).

Steganography.

Self-healing.

E2E: X25519 + box.Seal.

Signature: Ed25519.

FLAG_SECURE.

6. Weight model
Weight	Rights
0–0.3	Read
0.3–0.5	Full access
0.5–0.7	Comment
0.7–1.0	Vote, relay
7. Self-adaptation
node/adapt.go.

avgWeight, lowWeightRatio, highWeightRatio.

Background adaptation.

Status: architecture (v2.5+).

8. AI layer
Status: v3.0+.

9. API
REST: /send, /messages, /status, /health, /feedback.

WebSocket: /ws.

Filters.

Stego.

Protection levels
Level	What is protected	Status
Obfuscation	Traffic masking	✅ v1.13
E2E encryption	Content	✅ v1.24
Signature	Authenticity	✅ v1.24
Sphinx	Route hidden from relay	✅ v2.0.1
Mixing	Timings hidden	✅ v2.0.2
Masking	Type, volume, pattern hidden	✅ v2.0.4
Metadata (full)	Who talks to whom	🔜 v2.0.6+ (adaptive masking)
Timing patterns (full)	Timing, volume	🔜 Padding, mixing
Infrastructure
VPS bootstrap/relay
IP: 186.246.31.176.

PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi.

Bootstrap: /ip4/186.246.31.176/tcp/9000/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi (TCP; WS 9001 — fallback, blocked by the provider in Gorodok; fallback automatic — 3d82934). Temporary, until traffic masking (v2.0+).

Ports: 9000 (TCP), 9001 (WS), 8081 (HTTP API).

Do NOT delete /root/isotope/state/.

Phones (test)
Current PeerIDs are in the handoff (they change when the app is wiped).

Xiaomi, Huawei, Samsung — 4 phones.

VPS update
bash
cd /root/isotope-core
git pull
git --no-pager log --oneline -3
cd /root/isotope-core/node
go build -o isotope-node ./main
kill $(pgrep -f isotope-node)
sleep 1
pgrep -f isotope-node          # empty
nohup env NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true ./isotope-node > isotope.log 2>&1 &
sleep 3
pgrep -f isotope-node          # one new PID
curl -s http://127.0.0.1:8081/status
⚠️ pkill -f isotope-node on the VPS does NOT work. Only kill $(pgrep -f isotope-node).

Building the .aar
bash
cd /d D:\isotope\node
del isotope.aar
gomobile bind -target=android -androidapi 21 -ldflags "-checklinkname=0" -o isotope.aar ./mobile
copy /y isotope.aar D:\isotope\mobile\android\app