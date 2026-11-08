// node/node.go
package core

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	client "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
)

const protocolID = "/sbicore/1.0.0"
const syncProtocolID = "/sbicore/sync/1.0.0"
const pingProtocolID = "/sbicore/ping/1.0.0"

const OBFUSCATION_PREFIX = "[SHUF]"
const STEGO_PREFIX = "[STEGO]"
const REPLICA_PREFIX = "[REPLICA]"
const SPHINX_PREFIX = "[SPHINX]"
const RESTORE_PREFIX = "[RESTORE]"

const ANNOUNCE_PREFIX = "[ANNOUNCE]"
const FIND_PREFIX = "[FIND]"
const FOUND_PREFIX = "[FOUND]"
const NOT_FOUND_PREFIX = "[NOT_FOUND]"
const END_PREFIX = "[END]"

const ANNOUNCE_TTL = 8 * time.Minute

const SendSemaphoreCapacity = 10

const ChunkSize = 47 * 1024

const E2E_VERSION = 1

const MESSAGE_VERSION_E2E = 2

// MessageStatus — статус сообщения (для подтверждений доставки/прочтения).
type MessageStatus int

const (
	StatusSent      MessageStatus = 1
	StatusDelivered MessageStatus = 2
	StatusHidden    MessageStatus = 3
	StatusRead      MessageStatus = 4
)

// retryInfo — метаданные чанка, ожидающего подтверждения.
type retryInfo struct {
	mediaID    string
	mediaType  string
	chunkIndex int
	recipient  string
	msgID      string
	fileName   string
	fileSize   int64
	chunkTotal int
	ttlSeconds int
	ttlMode    string
	expiresAt  time.Time
	attempts   int
	lastSent   time.Time
}

// hookAckWindow — окно неподтверждённых чанков для одного mediaID.
type hookAckWindow struct {
	Indices map[int]time.Time
}

// announcedPeer — запись о пире: список его multiaddr + когда последний раз видели.
type announcedPeer struct {
	Multiaddrs []string  `json:"multiaddrs"`
	LastSeen   time.Time `json:"lastSeen"`
}

// qrDataV1 — формат QR-кода версии 1.
type qrDataV1 struct {
	V           int    `json:"v"`
	PeerID      string `json:"peerID"`
	Ed25519Pub  string `json:"ed25519_pub"`
	X25519Pub   string `json:"x25519_pub"`
	Signature   string `json:"signature"`
	DisplayName string `json:"display_name,omitempty"`
	ReadEnabled *bool  `json:"read_enabled,omitempty"`
}

// Config — конфигурация узла
type Config struct {
	EthHash           string
	Transports        []string
	Bootstrap         []string
	Port              int
	EnableMDNS        bool
	ListenIP          string
	EnableRelayServer bool
	IsRelay           bool
	UseOnion          bool
}

// Node — основной узел сети
type Node struct {
	host                    host.Host
	dhtNode                 *DHTNode
	ethHash                 string
	preHash                 string
	antiHash                string
	lastSyncSent            time.Time
	lastSyncedLayers        [][]float64
	layersDirty             bool
	memory                  Memory
	assoc                   AssocMemory
	layers                  [][]float64
	msgCount                int
	nodeID                  int
	stateFile               string
	configTransports        []string
	configPort              int
	configBootstrap         []string
	configEnableMDNS        bool
	configListenIP          string
	configEnableRelayServer bool
	isRelay                 bool
	messageHook             func(string)
	mu                      sync.Mutex
	lastPing                map[string]time.Time
	deadPeers               map[string]bool
	adaptive                *AdaptiveParams
	channels                *ChannelStore

	// ANNOUNCE — справочник (используется на VPS)
	announcedPeers     map[string]announcedPeer
	announcedMu        sync.Mutex
	announceMultiaddrs []string

	// RELAY — резервация слота на relay-сервере (VPS)
	relayReservation   *client.Reservation
	relayMu            sync.Mutex
	relayPeerInfo      peer.AddrInfo
	lastRelayRefresh   time.Time
	lastReservationErr error

	// OFFLINE QUEUE — сообщения, ожидающие отправки при восстановлении связи
	pendingMessages []Message
	pendingMu       sync.Mutex
	pendingFile     string

	// E2E — два ключа: Ed25519 (подпись) и X25519 (шифрование).
	ed25519KeyFile string
	ed25519Priv    ed25519.PrivateKey
	ed25519Pub     ed25519.PublicKey

	x25519KeyFile string
	x25519Priv    [32]byte
	x25519Pub     [32]byte

	// SEED (v2.0.6) — BIP39 seed-фраза. Только в памяти. Не на диск, не в лог.
	// Если непустая — InitP2P строит Ed25519 identity + X25519 из seed,
	// пропускает loadOrGenerateE2EKeys и loadPrivateKey (RSA).
	// Устанавливается через SetSeedPhrase (mobile.go) ДО InitP2P.
	seedPhrase string

	// CONTACTS — данные пользователя (не состояние узла).
	contactsFile string
	contacts     *ContactsStore

	// TEMP CONTACTS — временные контакты (только в памяти).
	tempContacts   map[string]Contact
	tempContactsMu sync.Mutex

	// REQUESTS — входящие запросы на контакт (контакт-протокол, этап 5).
	requestsFile string
	requests     *RequestsStore

	// DELETED — удалённые контакты (список peerID).
	deletedFile string
	deleted     *DeletedStore

	// ABORTED — отменённые mediaID (v2.0.7+). Переживает перезапуск.
	abortedFile  string
	abortedStore *AbortedStore

	// myReadEnabled — настройка "делюсь ли я статусом прочтения".
	myReadEnabled bool

	// SETTINGS — пользовательские настройки (isotope_settings.json).
	settingsStore *SettingsStore

	// saveStateScheduled — защита от частых saveState (throttle 5 сек).
	saveStateScheduled bool

	// savePendingScheduled — защита от частых savePendingQueue (throttle 5 сек).
	savePendingScheduled bool

	// sendSem — глобальный семафор исходящих stream'ов (10 одновременно).
	sendSem chan struct{}

	// serviceSem — семафор сервисных сообщений ([DELIVERED], [READ], [CONTACT_*]).
	serviceSem chan struct{}

	// mixingSem — семафор для mixing-чанков (5 одновременно).
	mixingSem chan struct{}

	// sphinxFileSem — семафор для Sphinx-файлов (3 одновременно).
	sphinxFileSem chan struct{}

	// relaySem — семафор для relay-репликации на VPS (10 одновременно).
	// Без него go n.replicateMessage на каждый Sphinx — горутины копятся — OOM.
	relaySem chan struct{}

	// chunkQueue — очередь файловых чанков для воркер-пула.
	chunkQueue chan Message

	// mediaSendMu / mediaSendCond / activeMediaID — последовательная постановка
	// файлов в chunkQueue (вариант A, Публикатор 2026-10-29).
	mediaSendMu   sync.Mutex
	mediaSendCond *sync.Cond
	activeMediaID string

	// batchBuffers — буферы текстов для батча (v2.0.4 c).
	batchBuffers   map[string]*batchBuffer
	batchBuffersMu sync.Mutex

	// unackedChunks — чанки, отправленные, но не подтверждённые [DELIVERED].
	unackedChunks   map[string]retryInfo
	unackedChunksMu sync.Mutex

	// aborted — mediaID → true, если отправитель прервал отправку.
	// sendFileChunksAsync проверяет перед каждым чанком → return.
	aborted   map[string]bool
	abortedMu sync.Mutex

	// fileStatusLastPush — throttle для push [FILE_STATUS] в Dart.
	fileStatusLastPush map[string]time.Time
	fileStatusMu       sync.Mutex

	// completedFiles — mediaID → true, если file_complete уже отправлен.
	completedFiles map[string]bool

	// Батч [DELIVERED] — буфер refs по отправителю.
	pendingDelivered   map[string][]string
	pendingDeliveredMu sync.Mutex

	// pendingDeliveredTimers — один таймер на sender.
	pendingDeliveredTimers map[string]*time.Timer

	// pendingHookAcks — окно чанков, отправленных в hook, но не подтверждённых Dart.
	pendingHookAcks   map[string]*hookAckWindow
	pendingHookAcksMu sync.Mutex

	// pendingAborts — mediaID → время последней отправки [FILE_ABORT]/[FILE_CANCEL].
	// Отправитель шлёт до ACK ([FILE_ABORT_ACK]) — без giving up.
	pendingAborts       map[string]time.Time
	pendingAbortsMu     sync.Mutex
	pendingAbortsTimers map[string]*time.Timer

	// deliveredCount — счётчик доставленных чанков по MediaID (отправитель).
	deliveredCount   map[string]int
	deliveredCountMu sync.Mutex

	// acceptedCount — счётчик принятых чанков по MediaID (получатель).
	acceptedCount   map[string]int
	acceptedCountMu sync.Mutex

	// SPHINX (v2.0.1) — флаг.
	useOnion bool

	// MIXING (v2.0.2) — флаг. Задержки на relay.
	mixingEnabled bool

	// dummyScreenOn — состояние экрана (v2.0.4 d.1.1).
	dummyScreenOn bool

	// OFFERS — входящие офферы на приём файлов (право отказа).
	// Отдельный файл isotope_offers.json.
	offersFile string
	offers     *OffersStore
}

// NewNode — создаёт новый узел
func NewNode(cfg Config) *Node {
	port := cfg.Port
	if port == 0 {
		port = 9000
	}
	listenIP := cfg.ListenIP
	if listenIP == "" {
		listenIP = "0.0.0.0"
	}
	n := &Node{
		ethHash:                 cfg.EthHash,
		configTransports:        cfg.Transports,
		configPort:              port,
		configBootstrap:         cfg.Bootstrap,
		configEnableMDNS:        cfg.EnableMDNS,
		configListenIP:          listenIP,
		configEnableRelayServer: cfg.EnableRelayServer,
		isRelay:                 cfg.IsRelay || cfg.EnableRelayServer,
		memory:                  Memory{seen: make(map[string]bool)},
		announcedPeers:          make(map[string]announcedPeer),
		tempContacts:            make(map[string]Contact),
		myReadEnabled:           true,
		sendSem:                 make(chan struct{}, SendSemaphoreCapacity),
		serviceSem:              make(chan struct{}, 2),
		mixingSem:               make(chan struct{}, 5),
		sphinxFileSem:           make(chan struct{}, 2),
		relaySem:                make(chan struct{}, 10),
		chunkQueue:              make(chan Message, 1000),
		unackedChunks:           make(map[string]retryInfo),
		aborted:                 make(map[string]bool),
		fileStatusLastPush:      make(map[string]time.Time),
		pendingDelivered:        make(map[string][]string),
		deliveredCount:          make(map[string]int),
		acceptedCount:           make(map[string]int),
		pendingDeliveredTimers:  make(map[string]*time.Timer),
		completedFiles:          make(map[string]bool),
		pendingHookAcks:         make(map[string]*hookAckWindow),
		pendingAborts:           make(map[string]time.Time),
		pendingAbortsTimers:     make(map[string]*time.Timer),
		batchBuffers:            make(map[string]*batchBuffer),
		useOnion:                cfg.UseOnion,
		dummyScreenOn:           true,
	}
	n.mediaSendCond = sync.NewCond(&n.mediaSendMu)
	return n
}

// SetMessageHook — устанавливает колбэк для новых сообщений
func (n *Node) SetMessageHook(hook func(string)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messageHook = hook
}

func (n *Node) HandlePeerFound(peerInfo peer.AddrInfo) {
	ctx := context.Background()
	if err := n.host.Connect(ctx, peerInfo); err != nil {
		return
	}
	go func() {
		time.Sleep(3 * time.Second)
		n.broadcastLayers()
	}()
}

// e2eCleanupFlagPath — путь к флагу однократной E2E-очистки.
func (n *Node) e2eCleanupFlagPath() string {
	if n.stateFile == "" {
		return ""
	}
	return n.stateFile + ".e2e_cleanup"
}

// runE2ECleanupOnce — однократная очистка своих E2E-сообщений без PlainText.
func (n *Node) runE2ECleanupOnce() {
	flagPath := n.e2eCleanupFlagPath()
	if flagPath == "" {
		return
	}
	if _, err := os.Stat(flagPath); err == nil {
		return
	}

	all := n.memory.GetAll()
	removed := 0
	for _, msg := range all {
		if msg.IsOwn && msg.Version == MESSAGE_VERSION_E2E && msg.PlainText == "" {
			if n.memory.Remove(msg.ID) {
				removed++
			}
		}
	}

	_ = os.WriteFile(flagPath, []byte("true"), 0600)

	if removed > 0 {
		log.Printf("[E2E CLEANUP] removed %d own messages without plaintext", removed)
		go n.saveState()
	} else {
		log.Printf("[E2E CLEANUP] no messages to remove")
	}
}

// InitP2P — инициализирует P2P
func (n *Node) InitP2P() error {
	if n.stateFile == "" {
		nodeIDStr := os.Getenv("NODE_ID")
		if nodeIDStr == "" {
			nodeIDStr = "1"
		}
		nodeID, _ := strconv.Atoi(nodeIDStr)
		n.nodeID = nodeID
		n.stateFile = fmt.Sprintf("state/state_node%d.json", nodeID)
	}

	n.adaptive = NewAdaptiveParams()
	n.channels = NewChannelStore()

	if err := n.loadState(); err != nil {
		log.Println("[INIT] Состояние не найдено, начинаем с нуля")
	}

	n.runE2ECleanupOnce()

	if n.pendingFile == "" {
		if n.stateFile != "" {
			n.pendingFile = n.stateFile + ".queue"
		}
	}
	n.loadPendingQueue()

	if n.stateFile != "" {
		_ = os.MkdirAll(n.sentDirPath(), 0700)
	}

	go n.resumeIncompleteFiles()

	// SEED (v2.0.6): если seed задан — Ed25519 (PeerID) + X25519 из seed.
	// loadOrGenerateE2EKeys и loadPrivateKey (RSA) при seed НЕ вызываются.
	// Иначе — как раньше (E2E из файлов + RSA identity).
	var seedIdentity crypto.PrivKey
	if n.seedPhrase != "" {
		sp, _, err := SeedToEd25519(n.seedPhrase)
		if err != nil {
			return fmt.Errorf("seed: SeedToEd25519: %w", err)
		}
		ed, ok := sp.(*crypto.Ed25519PrivateKey)
		if !ok {
			return fmt.Errorf("seed: unexpected libp2p key type")
		}
		raw, err := ed.Raw()
		if err != nil {
			return fmt.Errorf("seed: Ed25519 Raw: %w", err)
		}
		if len(raw) != ed25519.PrivateKeySize {
			return fmt.Errorf("seed: unexpected Ed25519 raw length %d", len(raw))
		}
		n.ed25519Priv = ed25519.PrivateKey(raw)
		n.ed25519Pub = n.ed25519Priv.Public().(ed25519.PublicKey)

		xPriv, xPub, err := SeedToX25519(n.seedPhrase)
		if err != nil {
			return fmt.Errorf("seed: SeedToX25519: %w", err)
		}
		n.x25519Priv = xPriv
		n.x25519Pub = xPub

		seedIdentity = sp
		log.Printf("[SEED] identity from seed (Ed25519)")
	} else {
		n.loadOrGenerateE2EKeys()
	}

	if n.contactsFile == "" && n.stateFile != "" {
		n.contactsFile = filepath.Join(filepath.Dir(n.stateFile), "isotope_contacts.json")
	}
	if n.contactsFile != "" {
		_ = os.MkdirAll(filepath.Dir(n.contactsFile), 0700)
		n.contacts = NewContactsStore(n.contactsFile)
	}

	if n.requestsFile == "" && n.stateFile != "" {
		n.requestsFile = filepath.Join(filepath.Dir(n.stateFile), "isotope_requests.json")
	}
	if n.requestsFile != "" {
		_ = os.MkdirAll(filepath.Dir(n.requestsFile), 0700)
		n.requests = NewRequestsStore(n.requestsFile)
	}

	if n.deletedFile == "" && n.stateFile != "" {
		n.deletedFile = filepath.Join(filepath.Dir(n.stateFile), "isotope_deleted.json")
	}
	if n.deletedFile != "" {
		_ = os.MkdirAll(filepath.Dir(n.deletedFile), 0700)
		n.deleted = NewDeletedStore(n.deletedFile)
	}

	// ABORTED — отменённые mediaID (v2.0.7+). Переживает перезапуск.
	if n.abortedFile == "" && n.stateFile != "" {
		n.abortedFile = filepath.Join(filepath.Dir(n.stateFile), "isotope_aborted.json")
	}
	if n.abortedFile != "" {
		_ = os.MkdirAll(filepath.Dir(n.abortedFile), 0700)
		n.abortedStore = NewAbortedStore(n.abortedFile)
	}

	// OFFERS — входящие офферы на приём файлов (право отказа).
	if n.offersFile == "" && n.stateFile != "" {
		n.offersFile = filepath.Join(filepath.Dir(n.stateFile), "isotope_offers.json")
	}
	if n.offersFile != "" {
		_ = os.MkdirAll(filepath.Dir(n.offersFile), 0700)
		n.offers = NewOffersStore(n.offersFile)
	}

	settingsFile := filepath.Join(filepath.Dir(n.stateFile), "isotope_settings.json")
	_ = os.MkdirAll(filepath.Dir(settingsFile), 0700)
	n.settingsStore = NewSettingsStore(settingsFile)
	n.myReadEnabled = n.settingsStore.GetMyReadEnabled()

	var priv crypto.PrivKey
	if seedIdentity != nil {
		priv = seedIdentity
	} else {
		keyBytes, err := n.loadPrivateKey()
		if err != nil {
			priv, _, _ = crypto.GenerateKeyPair(crypto.RSA, 2048)
			keyBytes, _ = crypto.MarshalPrivateKey(priv)
			n.savePrivateKey(keyBytes)
		} else {
			priv, _ = crypto.UnmarshalPrivateKey(keyBytes)
		}
	}

	listenAddr := fmt.Sprintf("/ip4/%s/tcp/%d", n.configListenIP, n.configPort)
	listenWS := fmt.Sprintf("/ip4/%s/tcp/%d/ws", n.configListenIP, n.configPort+1)

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddr, listenWS),
		libp2p.Identity(priv),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	}

	if len(n.configTransports) == 0 {
		opts = append(opts, libp2p.Transport(websocket.New))
	} else {
		for _, t := range n.configTransports {
			switch t {
			case "ws":
				opts = append(opts, libp2p.Transport(websocket.New))
			case "tcp":
				opts = append(opts, libp2p.Transport(tcp.NewTCPTransport))
			}
		}
	}

	host, err := libp2p.New(opts...)
	if err != nil {
		return err
	}
	n.host = host

	if n.contacts != nil {
		myID := n.host.ID().String()
		if _, ok := n.contacts.Get(myID); ok {
			_ = n.contacts.Remove(myID)
			log.Printf("[CONTACTS] removed self-contact %s (auto-cleanup)", myID)
		}
	}

	n.host.Network().Notify(&nodeNotifiee{node: n})

	n.host.SetStreamHandler(protocolID, n.handleStream)
	n.host.SetStreamHandler(syncProtocolID, n.handleSyncStream)
	n.host.SetStreamHandler(pingProtocolID, n.handlePingStream)

	if n.configEnableRelayServer {
		if _, err := relay.New(host); err != nil {
			log.Printf("[RELAY] Failed: %v", err)
		}
	}

	if n.configEnableMDNS {
		mdnsService := mdns.NewMdnsService(n.host, "_isotope._tcp.local", n)
		mdnsService.Start()
	}

	log.Println("[INIT] Node started with ID:", host.ID())

	bootstrapPeers := n.loadBootstrapPeers()

	if !n.configEnableRelayServer {
		for _, addr := range bootstrapPeers {
			go func(addr string) {
				peerInfo, err := peer.AddrInfoFromString(addr)
				if err != nil {
					return
				}
				ctx := context.Background()
				_ = n.host.Connect(ctx, *peerInfo)
				go func(peerID peer.ID) {
					time.Sleep(2 * time.Second)
					n.ExchangePeers(peerID.String())
				}(peerInfo.ID)

				rctx, rcancel := context.WithTimeout(context.Background(), 30*time.Second)
				err = n.ReserveRelaySlot(rctx, *peerInfo)
				rcancel()
				if err != nil {
					log.Printf("[RELAY] reserve failed on %s: %v", peerInfo.ID, err)
				}
			}(addr)
		}
	} else {
		for _, addr := range bootstrapPeers {
			go func(addr string) {
				peerInfo, err := peer.AddrInfoFromString(addr)
				if err != nil {
					return
				}
				ctx := context.Background()
				n.host.Connect(ctx, *peerInfo)
				go func(peerID peer.ID) {
					time.Sleep(2 * time.Second)
					n.ExchangePeers(peerID.String())
				}(peerInfo.ID)
			}(addr)
		}
	}

	if !n.isRelay {
		dhtNode, err := NewDHT(host)
		if err == nil {
			n.dhtNode = dhtNode
			dhtNode.JoinDHT(bootstrapPeers)
		}
	}

	if len(n.layers) == 0 {
		n.layers = append(n.layers, make([]float64, VectorDim))
		hashVec := hashToVector(n.ethHash)
		for i := range n.layers[0] {
			if i < len(hashVec) {
				n.layers[0][i] = hashVec[i] * 0.1
			}
		}
	}

	if n.settingsStore != nil {
		n.useOnion = n.settingsStore.GetSphinxEnabled()
	}
	if n.useOnion {
		log.Printf("[SPHINX] enabled (relays: %d)", len(n.selectSphinxRelays("")))
	}

	if n.settingsStore != nil && n.settingsStore.GetMixingEnabled() {
		log.Printf("[MIXING] enabled (sender-side flag)")
	}

	if !n.isRelay {
		for i := 0; i < SendSemaphoreCapacity; i++ {
			go n.chunkWorker()
		}
		n.retryLoop()
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				n.flushAllBatches()
			}
		}()
		go func() {
			for {
				time.Sleep(n.dummyInterval())
				n.sendDummy()
			}
		}()
	}

	n.pingPeers()
	if !n.isRelay {
		n.reconnectLoop()
	}
	n.announceLoop()
	n.cleanupLoop()
	n.cleanupSentLoop()
	if !n.isRelay {
		n.cleanupPendingAcksLoop()
	}
	n.relayLoop()
	n.StartAdaptation()

	go func() {
		time.Sleep(5 * time.Second)
		n.requestRestore()
	}()

	return nil
}

// StartHTTP — запускает HTTP-сервер
func (n *Node) StartHTTP(port int) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/send", n.handleSend)
	http.HandleFunc("/messages", n.handleMessages)
	http.HandleFunc("/status", n.handleStatus)
	http.HandleFunc("/peers/full", n.handlePeersFull)
	http.HandleFunc("/dht/find", n.handleDHTFindPeer)
	http.HandleFunc("/dht/provide", n.handleDHTProvide)
	http.HandleFunc("/dht/info", n.handleDHTInfo)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// StartMobile — запускает узел для мобильного
func (n *Node) StartMobile(stateFile string) error {
	if stateFile == "" {
		return fmt.Errorf("stateFile is required")
	}
	n.stateFile = stateFile
	n.adaptive = NewAdaptiveParams()
	n.channels = NewChannelStore()
	return n.InitP2P()
}

// Stop — корректно завершает узел
func (n *Node) Stop() error {
	if n.dhtNode != nil {
		n.dhtNode.Close()
	}
	n.saveState()
	n.savePendingQueue()
	if n.host != nil {
		return n.host.Close()
	}
	return nil
}

// GetStatus — возвращает JSON-статус
func (n *Node) GetStatus() string {
	if n.host == nil {
		return `{"id":"","peers":0,"memory":0,"layers":0,"requests":0}`
	}
	reqCount := 0
	if n.requests != nil {
		reqCount = n.requests.CountPending()
	}
	return fmt.Sprintf(`{"id":"%s","peers":%d,"memory":%d,"layers":%d,"requests":%d}`,
		n.host.ID().String(),
		len(n.host.Network().Peers()),
		n.memory.Count(),
		len(n.layers),
		reqCount,
	)
}

// handlePeersFull — диагностика: пиры из Peerstore + announced + connected.
func (n *Node) handlePeersFull(w http.ResponseWriter, r *http.Request) {
	if n.host == nil {
		http.Error(w, `{"error":"host not started"}`, http.StatusServiceUnavailable)
		return
	}

	type peerInfoJSON struct {
		PeerID      string   `json:"peerID"`
		Multiaddrs  []string `json:"multiaddrs"`
		Connected   bool     `json:"connected"`
		Announced   []string `json:"announced,omitempty"`
		AnnouncedAt string   `json:"announcedAt,omitempty"`
	}

	connected := make(map[string]bool)
	for _, p := range n.host.Network().Peers() {
		connected[p.String()] = true
	}

	announced := make(map[string]announcedPeer)
	n.announcedMu.Lock()
	for k, v := range n.announcedPeers {
		announced[k] = v
	}
	n.announcedMu.Unlock()

	seen := make(map[string]bool)
	var result []peerInfoJSON

	for _, p := range n.host.Peerstore().Peers() {
		pid := p.String()
		seen[pid] = true
		info := n.host.Peerstore().PeerInfo(p)
		var addrs []string
		for _, a := range info.Addrs {
			addrs = append(addrs, a.String())
		}
		entry := peerInfoJSON{
			PeerID:     pid,
			Multiaddrs: addrs,
			Connected:  connected[pid],
		}
		if ap, ok := announced[pid]; ok {
			entry.Announced = ap.Multiaddrs
			entry.AnnouncedAt = ap.LastSeen.Format(time.RFC3339)
		}
		result = append(result, entry)
	}

	for pid, ap := range announced {
		if seen[pid] {
			continue
		}
		result = append(result, peerInfoJSON{
			PeerID:      pid,
			Connected:   connected[pid],
			Announced:   ap.Multiaddrs,
			AnnouncedAt: ap.LastSeen.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetHost — возвращает libp2p host
func (n *Node) GetHost() host.Host {
	return n.host
}

// SetDHT — устанавливает DHT
func (n *Node) SetDHT(d *DHTNode) {
	n.dhtNode = d
}

// GetDHT — возвращает DHT
func (n *Node) GetDHT() *DHTNode {
	return n.dhtNode
}

// truncate — обрезает строку для лога.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// sendToRecipient — адресная отправка.
func (n *Node) sendToRecipient(msg Message, data []byte, isFileChunk bool) bool {
	targetID, err := peer.Decode(msg.Recipient)
	if err != nil {
		log.Printf("[REPLICA] invalid Recipient %q: %v", msg.Recipient, err)
		return false
	}

	isService := n.isServiceType(msg.Type)
	acquire := func() func() {
		if isService {
			n.serviceSem <- struct{}{}
			return func() { <-n.serviceSem }
		}
		if !isFileChunk {
			n.sendSem <- struct{}{}
			return func() { <-n.sendSem }
		}
		return func() {}
	}

	if n.isPeerDead(msg.Recipient) {
		release := acquire()
		defer release()
		return n.fallbackSend(msg.Recipient, data, isFileChunk)
	}

	inPeers := false
	for _, p := range n.host.Network().Peers() {
		if p == targetID {
			inPeers = true
			break
		}
	}
	if inPeers {
		if isFileChunk {
			n.sendWithFallback(targetID, data, msg.Recipient, isFileChunk)
		} else {
			go func() {
				release := acquire()
				defer release()
				n.sendWithFallback(targetID, data, msg.Recipient, isFileChunk)
			}()
		}
		return true
	}

	release := acquire()
	defer release()
	return n.fallbackSend(msg.Recipient, data, isFileChunk)
}

// tryDirectSend — синхронная попытка прямого stream с коротким timeout.
func (n *Node) tryDirectSend(targetID peer.ID, data []byte, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	s, err := n.host.NewStream(ctx, targetID, protocolID)
	if err != nil {
		return false
	}
	defer s.Close()
	if _, err := fmt.Fprintf(s, "%s%s\n", REPLICA_PREFIX, string(data)); err != nil {
		return false
	}
	return true
}

// sendWithFallback — синхронная (вызывается из горутины).
func (n *Node) sendWithFallback(targetID peer.ID, data []byte, recipient string, isFileChunk bool) {
	if n.tryDirectSend(targetID, data, 3*time.Second) {
		return
	}
	log.Printf("[REPLICA] direct send to %s failed (phantom circuit?), fallback", targetID)
	n.fallbackSend(recipient, data, isFileChunk)
}

// fallbackSend — синхронная. Логика fallback:
//   - relay → announcedPeers;
//   - клиент → bootstrap.
func (n *Node) fallbackSend(recipient string, data []byte, isFileChunk bool) bool {
	if n.isRelay {
		if addrs, ok := n.lookupPeer(recipient); ok && len(addrs) > 0 {
			go func(pid string, addrList []string, payload []byte, isFileChunk bool) {
				for _, a := range addrList {
					pi, err := peer.AddrInfoFromString(a)
					if err != nil {
						continue
					}
					if pi.ID.String() != pid {
						continue
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					if err := n.host.Connect(ctx, *pi); err != nil {
						cancel()
						continue
					}
					cancel()
					n.sendReplicaToPeer(pi.ID, payload, isFileChunk)
					return
				}
			}(recipient, addrs, data, isFileChunk)
			return true
		}
		log.Printf("[STATUS] relay: recipient not found in announcedPeers: %s", recipient)
		return false
	}

	bootstrapPeers := n.loadBootstrapPeers()
	for _, addr := range bootstrapPeers {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		if n.host.ID() == pi.ID {
			continue
		}
		if isFileChunk {
			n.sendReplicaToPeer(pi.ID, data, isFileChunk)
		} else {
			go func(pid peer.ID) {
				n.sendSem <- struct{}{}
				defer func() { <-n.sendSem }()
				n.sendReplicaToPeer(pid, data, isFileChunk)
			}(pi.ID)
		}
		return true
	}

	log.Printf("[STATUS] recipient not found: %s", recipient)
	return false
}

// sendReplicaToPeer — отправка данных пиру.
func (n *Node) sendReplicaToPeer(targetID peer.ID, data []byte, isFileChunk bool) {
	if !isFileChunk {
		myID := ""
		if n.host != nil {
			myID = n.host.ID().String()
		}
		var msgPreview string
		if len(data) > 200 {
			msgPreview = string(data[:200])
		} else {
			msgPreview = string(data)
		}
		log.Printf("[REPLICA] sendReplicaToPeer target=%s me=%s data=%s", targetID, myID, msgPreview)
		randomDelay(10, 30)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, err := n.host.NewStream(ctx, targetID, protocolID)
	if err != nil {
		if !isFileChunk {
			log.Printf("[REPLICA] NewStream to %s failed: %v", targetID, err)
		}
		return
	}
	defer s.Close()
	s.Write([]byte(REPLICA_PREFIX))
	s.Write(data)
	s.Write([]byte("\n"))
}

// sendServiceViaBootstrap — отправка сервисного сообщения через bootstrap (relay).
func (n *Node) sendServiceViaBootstrap(msg Message) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if msg.Recipient == "" {
		return fmt.Errorf("recipient is required")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	bootstrapPeers := n.loadBootstrapPeers()
	if len(bootstrapPeers) == 0 {
		return fmt.Errorf("no bootstrap peer available")
	}

	myID := n.host.ID()
	var lastErr error
	for _, addr := range bootstrapPeers {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			lastErr = fmt.Errorf("invalid bootstrap addr %q: %w", addr, err)
			continue
		}
		if pi.ID == myID {
			continue
		}

		n.serviceSem <- struct{}{}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		s, err := n.host.NewStream(ctx, pi.ID, protocolID)
		if err != nil {
			cancel()
			<-n.serviceSem
			lastErr = fmt.Errorf("NewStream to %s failed: %w", pi.ID, err)
			continue
		}
		_, err = fmt.Fprintf(s, "%s%s\n", REPLICA_PREFIX, string(data))
		s.Close()
		cancel()
		<-n.serviceSem
		if err != nil {
			lastErr = fmt.Errorf("write to %s failed: %w", pi.ID, err)
			continue
		}
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no reachable bootstrap peer")
	}
	return lastErr
}

// findMyMessageByID — находит моё сообщение по ID в памяти.
func (n *Node) findMyMessageByID(id string) (Message, bool) {
	if id == "" {
		return Message{}, false
	}
	return n.memory.GetByID(id)
}

// loadPendingQueue — загружает очередь из файла.
func (n *Node) loadPendingQueue() {
	if n.pendingFile == "" {
		return
	}
	data, err := os.ReadFile(n.pendingFile)
	if err != nil {
		return
	}
	var queue []Message
	if err := json.Unmarshal(data, &queue); err != nil {
		log.Printf("[QUEUE] load failed: %v", err)
		return
	}
	n.pendingMu.Lock()
	n.pendingMessages = queue
	n.pendingMu.Unlock()
	log.Printf("[QUEUE] loaded %d pending messages", len(queue))
}

// savePendingQueue — сохраняет очередь в файл.
func (n *Node) savePendingQueue() {
	if n.pendingFile == "" {
		return
	}
	n.pendingMu.Lock()
	queue := make([]Message, len(n.pendingMessages))
	copy(queue, n.pendingMessages)
	n.pendingMu.Unlock()

	data, err := json.Marshal(queue)
	if err != nil {
		log.Printf("[QUEUE] marshal failed: %v", err)
		return
	}
	if err := os.WriteFile(n.pendingFile, data, 0600); err != nil {
		log.Printf("[QUEUE] save failed: %v", err)
	}
}

// enqueuePending — добавляет сообщение в очередь.
func (n *Node) enqueuePending(msg Message) {
	if isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 {
		msg.Text = ""
		msg.PlainText = ""
	}
	n.pendingMu.Lock()
	n.pendingMessages = append(n.pendingMessages, msg)
	count := len(n.pendingMessages)
	n.pendingMu.Unlock()
	log.Printf("[QUEUE] enqueued %s (total: %d)", msg.ID, count)
	n.scheduleSavePendingQueue()
}

// scheduleSavePendingQueue — отложенная запись очереди (throttle 5 сек).
func (n *Node) scheduleSavePendingQueue() {
	n.pendingMu.Lock()
	if n.savePendingScheduled {
		n.pendingMu.Unlock()
		return
	}
	n.savePendingScheduled = true
	n.pendingMu.Unlock()

	go func() {
		time.Sleep(5 * time.Second)
		n.pendingMu.Lock()
		n.savePendingScheduled = false
		n.pendingMu.Unlock()
		n.savePendingQueue()
	}()
}

// hasPending — есть ли что-то в очереди.
func (n *Node) hasPending() bool {
	n.pendingMu.Lock()
	defer n.pendingMu.Unlock()
	return len(n.pendingMessages) > 0
}

// flushPending — переотправляет всё из очереди.
func (n *Node) flushPending() {
	n.pendingMu.Lock()
	queue := make([]Message, len(n.pendingMessages))
	copy(queue, n.pendingMessages)
	n.pendingMu.Unlock()

	if len(queue) == 0 {
		return
	}

	log.Printf("[QUEUE] flushing %d pending messages", len(queue))
	for _, msg := range queue {
		n.tryReplicate(msg)
	}
	log.Printf("[QUEUE] flushed, remaining: %d", len(queue))
}

// removePendingByRef — удаляет сообщение из очереди по ID.
func (n *Node) removePendingByRef(ref string) {
	if ref == "" {
		return
	}
	n.pendingMu.Lock()
	before := len(n.pendingMessages)
	var remaining []Message
	for _, m := range n.pendingMessages {
		if m.ID != ref {
			remaining = append(remaining, m)
		}
	}
	n.pendingMessages = remaining
	after := len(n.pendingMessages)
	n.pendingMu.Unlock()
	if before != after {
		log.Printf("[QUEUE] removed %s from pending (delivered)", ref)
		n.savePendingQueue()
	}
}

// tryReplicate — попытка отправить одно сообщение (для flushPending).
func (n *Node) tryReplicate(msg Message) bool {
	if n.host == nil {
		return false
	}
	if isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0 &&
		msg.Text == "" && msg.PlainText == "" && msg.MediaID != "" {
		if err := n.fillFileChunkFromSent(&msg); err != nil {
			log.Printf("[QUEUE] fillFileChunkFromSent %s[%d] failed: %v",
				msg.MediaID, msg.ChunkIndex, err)
			return false
		}
	}
	msg.ReplicatedFrom = msg.Sender
	msg.IsOwn = false
	data, err := json.Marshal(msg)
	if err != nil {
		return false
	}

	if msg.Recipient != "" {
		isFileChunk := isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0
		return n.sendToRecipient(msg, data, isFileChunk)
	}

	peers := n.host.Network().Peers()
	var alive []peer.ID
	for _, p := range peers {
		if p.String() == msg.Sender {
			continue
		}
		alive = append(alive, p)
	}
	if len(alive) == 0 {
		return false
	}
	replicaCount := 2
	if len(alive) < replicaCount {
		replicaCount = len(alive)
	}
	sent := false
	for i := 0; i < replicaCount; i++ {
		peerID := alive[i]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		s, err := n.host.NewStream(ctx, peerID, protocolID)
		cancel()
		if err != nil {
			continue
		}
		fmt.Fprintf(s, "%s%s\n", REPLICA_PREFIX, string(data))
		s.Close()
		sent = true
	}
	return sent
}

// loadState — загружает состояние из файла.
func (n *Node) loadState() error {
	data, err := n.loadStateData()
	if err != nil {
		return err
	}
	var state struct {
		Messages      []Message                `json:"messages"`
		Layers        [][]float64              `json:"layers"`
		MsgCount      int                      `json:"msgCount"`
		PreHash       string                   `json:"preHash"`
		AntiHash      string                   `json:"antiHash"`
		RoutingTable  []string                 `json:"routingTable"`
		MessageStatus map[string]MessageStatus `json:"messageStatus"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	if len(state.MessageStatus) > 0 {
		for i := range state.Messages {
			if status, ok := state.MessageStatus[state.Messages[i].ID]; ok {
				state.Messages[i].Status = status
			}
		}
		log.Printf("[STATUS] migrated %d statuses to Message.Status", len(state.MessageStatus))
	}
	for i := range state.Messages {
		state.Messages[i].Time = migrateTime(state.Messages[i].Time)
	}
	for _, msg := range state.Messages {
		n.memory.Add(msg)
	}
	n.layers = state.Layers
	n.msgCount = state.MsgCount
	n.preHash = state.PreHash
	n.antiHash = state.AntiHash
	n.layersDirty = true

	if len(state.RoutingTable) > 0 {
		if n.dhtNode != nil {
			data, _ := json.Marshal(state.RoutingTable)
			n.dhtNode.LoadRoutingTable(data)
		}
	}
	return nil
}

// handleReplicaData — обработка данных RESTORE.
func (n *Node) handleReplicaData(data string) {
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, REPLICA_PREFIX) {
			payload := strings.TrimPrefix(line, REPLICA_PREFIX)
			var replicaMsg Message
			if err := json.Unmarshal([]byte(payload), &replicaMsg); err == nil {
				if replicaMsg.Version < 2 {
					if plaintext, ok := n.deobfuscate(replicaMsg.Text); ok {
						replicaMsg.Text = plaintext
					}
				}
				replicaMsg.ReplicatedAt = time.Now()
				replicaMsg.IsOwn = false
				replicaMsg.SenderName = n.nameForPeer(replicaMsg.Sender)
				if n.memory.Add(replicaMsg) {
					if n.isRelay {
						n.relayReplicate(replicaMsg)
					} else {
						go n.replicateMessage(replicaMsg)
					}
					if n.messageHook != nil {
						data, _ := json.Marshal(replicaMsg)
						n.messageHook(string(data))
					}
				}
			}
		}
	}
}

// node/node.go
