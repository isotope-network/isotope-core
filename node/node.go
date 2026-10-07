// node/node.go
package core

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
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
const RESTORE_PREFIX = "[RESTORE]"

const ANNOUNCE_PREFIX = "[ANNOUNCE]"
const FIND_PREFIX = "[FIND]"
const FOUND_PREFIX = "[FOUND]"
const NOT_FOUND_PREFIX = "[NOT_FOUND]"
const END_PREFIX = "[END]"

const ANNOUNCE_TTL = 8 * time.Minute

// E2E_VERSION — версия формата QR-обмена E2E-ключами.
// 0 — старый формат (только PeerID). 1 — JSON с ключами.
const E2E_VERSION = 1

// MESSAGE_VERSION — версия формата сообщения.
// 0 — история / broadcast (открытое).
// 2 — E2E-шифрованное (Text содержит base64(nonce||ciphertext)).
const MESSAGE_VERSION_E2E = 2

// MessageStatus — статус сообщения (для подтверждений доставки/прочтения).
// 1 = отправлено, 2 = доставлено, 3 = скрыто, 4 = прочитано.
// Приоритет: read (4) > hidden (3) > delivered (2) > sent (1).
// setMessageStatus не понижает — только повышает.
type MessageStatus int

const (
	StatusSent      MessageStatus = 1
	StatusDelivered MessageStatus = 2
	StatusHidden    MessageStatus = 3 // скрыто (получатель не делится статусом)
	StatusRead      MessageStatus = 4 // прочитано (высший приоритет)
)

// announcedPeer — запись о пире: список его multiaddr + когда последний раз видели.
type announcedPeer struct {
	Multiaddrs []string  `json:"multiaddrs"`
	LastSeen   time.Time `json:"lastSeen"`
}

// qrDataV1 — формат QR-кода версии 1.
type qrDataV1 struct {
	V          int    `json:"v"`
	PeerID     string `json:"peerID"`
	Ed25519Pub string `json:"ed25519_pub"`
	X25519Pub  string `json:"x25519_pub"`
	Signature  string `json:"signature"`
	// DisplayName — представление владельца QR (как он хочет быть представлен).
	DisplayName string `json:"display_name,omitempty"`
	ReadEnabled *bool  `json:"read_enabled,omitempty"` // nil — не передан (дефолт true)
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
	IsRelay           bool // true для relay-узла (VPS): форвардит Version=2
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
	isRelay                 bool // true для relay-узла (VPS)
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
	lastReservationErr error // последняя ошибка резервации (nil = успех)

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

	// CONTACTS — данные пользователя (не состояние узла).
	// Отдельный файл isotope_contacts.json.
	contactsFile string
	contacts     *ContactsStore

	// TEMP CONTACTS — временные контакты (только в памяти).
	// Создаются при получении [CONTACT_HELLO] от A. Нужны, чтобы
	// расшифровать [CONTACT_REQUEST] (E2E). После requests.Add — удаляются.
	// Не сохраняются на диск, не попадают в UI.
	tempContacts   map[string]Contact
	tempContactsMu sync.Mutex

	// REQUESTS — входящие запросы на контакт (контакт-протокол, этап 5).
	// Отдельный файл isotope_requests.json. До accept/reject.
	requestsFile string
	requests     *RequestsStore

	// DELETED — удалённые контакты (список peerID).
	// Отдельный файл isotope_deleted.json. Фильтр для UI — удалённые
	// не показываются, даже если есть в сети / пишут.
	deletedFile string
	deleted     *DeletedStore

	// myReadEnabled — настройка "делюсь ли я статусом прочтения".
	// true (по умолчанию) — отправляю [READ] и вижу чужие [READ].
	// false — не отправляю [READ], чужие не отображаю (✓✓🔒).
	// Кэш для быстрого чтения. Источник истины — settingsStore.
	myReadEnabled bool

	// SETTINGS — пользовательские настройки (isotope_settings.json).
	// Отдельно от state: это предпочтения пользователя, не состояние сети.
	settingsStore *SettingsStore

	// saveStateScheduled — защита от частых saveState (throttle 5 сек).
	saveStateScheduled bool

	// sendSem — глобальный семафор исходящих stream'ов (10 одновременно).
	// Защита от перегруза libp2p при flushPending / SendFile.
	sendSem chan struct{}

	// Батч [DELIVERED] — буфер refs по отправителю.
	// Копим msg_id чанков, отправляем одним [DELIVERED] с Refs.
	pendingDelivered   map[string][]string
	pendingDeliveredMu sync.Mutex
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
	return &Node{
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
		sendSem:                 make(chan struct{}, 10),
		pendingDelivered:        make(map[string][]string),
	}
}

// SetMessageHook — устанавливает колбэк для новых сообщений
func (n *Node) SetMessageHook(hook func(string)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messageHook = hook
}

func (n *Node) getObfuscationKey() []byte {
	hash := sha256.Sum256([]byte(n.ethHash))
	return hash[:]
}

func (n *Node) obfuscate(msg string) string {
	key := n.getObfuscationKey()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(cryptorand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, []byte(msg), nil)
	return OBFUSCATION_PREFIX + base64.StdEncoding.EncodeToString(ciphertext)
}

func (n *Node) deobfuscate(data string) (string, bool) {
	if !strings.HasPrefix(data, OBFUSCATION_PREFIX) {
		return data, false
	}
	encoded := strings.TrimPrefix(data, OBFUSCATION_PREFIX)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return data, false
	}
	key := n.getObfuscationKey()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return data, false
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return data, false
	}
	return string(plaintext), true
}

func randomDelay(minMs, maxMs int) {
	n, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(maxMs-minMs)))
	delay := time.Duration(minMs+int(n.Int64())) * time.Millisecond
	time.Sleep(delay)
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

// ============================================================
// E2E — два ключа: Ed25519 (подпись) + X25519 (шифрование).
// Этап 4.3.1. Публичные ключи не хранятся — вычисляются из приватных.
// ============================================================

// loadOrGenerateE2EKeys — загружает / генерирует Ed25519 и X25519 ключи.
func (n *Node) loadOrGenerateE2EKeys() {
	if n.stateFile == "" {
		log.Printf("[KEY] stateFile not set, E2E keys skipped")
		return
	}
	if n.ed25519KeyFile == "" {
		n.ed25519KeyFile = n.stateFile + ".ed25519.key"
	}
	if n.x25519KeyFile == "" {
		n.x25519KeyFile = n.stateFile + ".x25519.key"
	}

	if err := n.loadOrGenerateEd25519Key(); err != nil {
		log.Printf("[KEY] ed25519 failed: %v", err)
	}
	if err := n.loadOrGenerateX25519Key(); err != nil {
		log.Printf("[KEY] x25519 failed: %v", err)
	}
}

// loadOrGenerateEd25519Key — загружает или генерирует Ed25519-ключ (подпись).
func (n *Node) loadOrGenerateEd25519Key() error {
	if data, err := os.ReadFile(n.ed25519KeyFile); err == nil {
		if len(data) == ed25519.PrivateKeySize {
			n.ed25519Priv = ed25519.PrivateKey(data)
			n.ed25519Pub = n.ed25519Priv.Public().(ed25519.PublicKey)
			log.Printf("[KEY] loaded ed25519 (pub=%s...)",
				truncate(base64.StdEncoding.EncodeToString(n.ed25519Pub), 16))
			return nil
		}
		log.Printf("[KEY] regenerated ed25519 key (wrong size: %d)", len(data))
	}

	pub, priv, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		return fmt.Errorf("ed25519 keygen failed: %w", err)
	}
	n.ed25519Priv = priv
	n.ed25519Pub = pub

	if err := os.WriteFile(n.ed25519KeyFile, priv, 0600); err != nil {
		return fmt.Errorf("ed25519 save failed: %w", err)
	}

	log.Printf("[KEY] generated ed25519 (pub=%s...)",
		truncate(base64.StdEncoding.EncodeToString(pub), 16))
	return nil
}

// loadOrGenerateX25519Key — загружает или генерирует X25519-ключ (шифрование).
func (n *Node) loadOrGenerateX25519Key() error {
	if data, err := os.ReadFile(n.x25519KeyFile); err == nil {
		if len(data) == 32 {
			copy(n.x25519Priv[:], data)
			pub, err := deriveX25519Public(n.x25519Priv)
			if err != nil {
				log.Printf("[KEY] x25519 derive failed: %v", err)
			} else {
				n.x25519Pub = pub
			}
			log.Printf("[KEY] loaded x25519 (pub=%s...)",
				truncate(base64.StdEncoding.EncodeToString(n.x25519Pub[:]), 16))
			return nil
		}
		log.Printf("[KEY] regenerated x25519 key (wrong size: %d)", len(data))
	}

	pub, priv, err := box.GenerateKey(cryptorand.Reader)
	if err != nil {
		return fmt.Errorf("x25519 keygen failed: %w", err)
	}
	n.x25519Priv = *priv
	n.x25519Pub = *pub

	if err := os.WriteFile(n.x25519KeyFile, n.x25519Priv[:], 0600); err != nil {
		return fmt.Errorf("x25519 save failed: %w", err)
	}

	log.Printf("[KEY] generated x25519 (pub=%s...)",
		truncate(base64.StdEncoding.EncodeToString(n.x25519Pub[:]), 16))
	return nil
}

// deriveX25519Public — вычисляет публичный ключ из приватного X25519.
func deriveX25519Public(priv [32]byte) ([32]byte, error) {
	pubBytes, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return [32]byte{}, err
	}
	var result [32]byte
	copy(result[:], pubBytes)
	return result, nil
}

// e2eCleanupFlagPath — путь к флагу однократной E2E-очистки.
// Отдельный файл рядом со state: isotope_state.json.e2e_cleanup.
func (n *Node) e2eCleanupFlagPath() string {
	if n.stateFile == "" {
		return ""
	}
	return n.stateFile + ".e2e_cleanup"
}

// runE2ECleanupOnce — однократная очистка своих E2E-сообщений без PlainText.
// Удаляет мусор: сообщения, у которых Text = шифротекст, а PlainText не был
// сохранён (до фикса). Работает один раз — флаг в отдельном файле.
func (n *Node) runE2ECleanupOnce() {
	flagPath := n.e2eCleanupFlagPath()
	if flagPath == "" {
		return
	}
	// Уже сделано?
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

	// Флаг — до saveState. Чтобы повторный запуск не задублировал очистку.
	_ = os.WriteFile(flagPath, []byte("true"), 0600)

	if removed > 0 {
		log.Printf("[E2E CLEANUP] removed %d own messages without plaintext", removed)
		go n.saveState()
	} else {
		log.Printf("[E2E CLEANUP] no messages to remove")
	}
}

// GetEd25519PublicKey — возвращает Ed25519-публичный ключ в base64.
func (n *Node) GetEd25519PublicKey() string {
	if len(n.ed25519Pub) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(n.ed25519Pub)
}

// GetX25519PublicKey — возвращает X25519-публичный ключ в base64.
func (n *Node) GetX25519PublicKey() string {
	return base64.StdEncoding.EncodeToString(n.x25519Pub[:])
}

// ============================================================
// 4.4 — ПОДПИСЬ. Ed25519 подписывает peerID || x25519_pub.
// Связка peerID + x25519_pub защищает от подмены:
// злоумышленник не сможет подписать чужой x25519_pub своим ключом
// и выдать себя за другого — peerID в подписи не совпадёт.
// ============================================================

// signMyX25519 — подпись peerID || x25519_pub приватным Ed25519-ключом.
// Возвращает base64-подпись или "" при ошибке / отсутствии ключей.
func (n *Node) signMyX25519() string {
	if n.host == nil || len(n.ed25519Priv) == 0 {
		return ""
	}
	peerID := n.host.ID().String()
	// peerID || x25519_pub
	msg := make([]byte, 0, len(peerID)+32)
	msg = append(msg, []byte(peerID)...)
	msg = append(msg, n.x25519Pub[:]...)

	sig := ed25519.Sign(n.ed25519Priv, msg)
	return base64.StdEncoding.EncodeToString(sig)
}

// verifyContactSignature — проверяет подпись контакта.
// Подпись — над peerID || x25519_pub.
// Возвращает true, только если все три декодировались корректно
// и подпись валидна.
func verifyContactSignature(peerID, ed25519PubB64, x25519PubB64, signatureB64 string) bool {
	if peerID == "" || ed25519PubB64 == "" || x25519PubB64 == "" || signatureB64 == "" {
		return false
	}

	edPub, err := base64.StdEncoding.DecodeString(ed25519PubB64)
	if err != nil || len(edPub) != ed25519.PublicKeySize {
		return false
	}
	xPub, err := base64.StdEncoding.DecodeString(x25519PubB64)
	if err != nil || len(xPub) != 32 {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}

	msg := make([]byte, 0, len(peerID)+32)
	msg = append(msg, []byte(peerID)...)
	msg = append(msg, xPub...)

	return ed25519.Verify(ed25519.PublicKey(edPub), msg, sig)
}

// GetMyQRData — возвращает JSON для QR-кода версии 1.
// Signature — подпись peerID || x25519_pub. Заполняется на 4.4.
func (n *Node) GetMyQRData() string {
	if n.host == nil {
		return ""
	}
	readEnabled := n.myReadEnabled
	displayName := n.GetMyDisplayName()
	data, err := json.Marshal(qrDataV1{
		V:           E2E_VERSION,
		PeerID:      n.host.ID().String(),
		Ed25519Pub:  base64.StdEncoding.EncodeToString(n.ed25519Pub),
		X25519Pub:   base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		Signature:   n.signMyX25519(),
		DisplayName: displayName,
		ReadEnabled: &readEnabled,
	})
	if err != nil {
		log.Printf("[KEY] QR marshal failed: %v", err)
		return ""
	}
	return string(data)
}

// truncate — обрезает строку для лога.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ============================================================
// E2E-ШИФРОВАНИЕ (этап 4.3.3)
// ============================================================

// encryptForRecipient — шифрует text для получателя.
// recipient — PeerID получателя. Требуется контакт с x25519_pub.
// Возвращает payload: base64(nonce(24) || ciphertext).
func (n *Node) encryptForRecipient(recipient, text string) (string, error) {
	if n.contacts == nil {
		return "", fmt.Errorf("contacts store not initialized")
	}

	contact, ok := n.contacts.Get(recipient)
	if !ok {
		return "", fmt.Errorf("contact not found: %s", recipient)
	}
	if contact.X25519Pub == "" {
		return "", fmt.Errorf("contact has no E2E key: %s", recipient)
	}

	recipientPubBytes, err := base64.StdEncoding.DecodeString(contact.X25519Pub)
	if err != nil || len(recipientPubBytes) != 32 {
		return "", fmt.Errorf("invalid x25519_pub for %s", recipient)
	}
	var recipientPub [32]byte
	copy(recipientPub[:], recipientPubBytes)

	var nonce [24]byte
	if _, err := cryptorand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("nonce generation failed: %w", err)
	}

	ciphertext := box.Seal(nil, []byte(text), &nonce, &recipientPub, &n.x25519Priv)
	payload := append(nonce[:], ciphertext...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

// decryptFromSender — расшифровывает payload от отправителя.
// payload — base64(nonce(24) || ciphertext).
func (n *Node) decryptFromSender(sender, payload string) (string, error) {
	if n.contacts == nil {
		return "", fmt.Errorf("contacts store not initialized")
	}

	contact, ok := n.GetContactTemp(sender)
	if !ok {
		return "", fmt.Errorf("contact not found: %s", sender)
	}
	if contact.X25519Pub == "" {
		return "", fmt.Errorf("contact has no E2E key: %s", sender)
	}

	senderPubBytes, err := base64.StdEncoding.DecodeString(contact.X25519Pub)
	if err != nil || len(senderPubBytes) != 32 {
		return "", fmt.Errorf("invalid x25519_pub for %s", sender)
	}
	var senderPub [32]byte
	copy(senderPub[:], senderPubBytes)

	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(raw) < 24 {
		return "", fmt.Errorf("invalid payload from %s", sender)
	}

	var nonce [24]byte
	copy(nonce[:], raw[:24])
	ciphertext := raw[24:]

	plaintext, ok := box.Open(nil, ciphertext, &nonce, &senderPub, &n.x25519Priv)
	if !ok {
		return "", fmt.Errorf("decryption failed for %s", sender)
	}
	return string(plaintext), nil
}

// ============================================================
// RELAY-RESERVATION (клиент)
// ============================================================

// ReserveRelaySlot — резервирует слот на relay-сервере.
func (n *Node) ReserveRelaySlot(ctx context.Context, relayAddrInfo peer.AddrInfo) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}

	resv, err := client.Reserve(ctx, n.host, relayAddrInfo)
	if err != nil {
		n.relayMu.Lock()
		n.lastReservationErr = err
		n.relayMu.Unlock()
		return fmt.Errorf("relay reserve failed: %w", err)
	}

	n.relayMu.Lock()
	n.relayReservation = resv
	n.relayPeerInfo = relayAddrInfo
	n.lastReservationErr = nil
	n.relayMu.Unlock()

	log.Printf("[RELAY] reserved slot, expires=%s", resv.Expiration)
	return nil
}

// relayLoop — периодически проверяет наличие резервации и обновляет её.
// Интервал — 30 секунд. Проверяет не только таймер истечения,
// но и сам факт наличия резервации (resv == nil — пересоздаём).
func (n *Node) relayLoop() {
	go func() {
		backoff := 2 * time.Second
		const maxBackoff = 30 * time.Second

		for {
			time.Sleep(backoff)

			n.relayMu.Lock()
			resv := n.relayReservation
			relayInfo := n.relayPeerInfo
			n.relayMu.Unlock()

			if relayInfo.ID == "" {
				continue
			}

			needReserve := false
			if resv == nil {
				needReserve = true
			} else if time.Until(resv.Expiration) < 5*time.Minute {
				needReserve = true
			}

			if !needReserve {
				// Всё хорошо, сбрасываем backoff.
				backoff = 2 * time.Second
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := n.ReserveRelaySlot(ctx, relayInfo)
			cancel()
			if err != nil {
				log.Printf("[RELAY] reserve failed (backoff=%s): %v", backoff, err)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			// Успех — сброс.
			backoff = 2 * time.Second
		}
	}()
}

// GetRelayAddrs — возвращает список relay-адресов для анонса.
func (n *Node) GetRelayAddrs() []string {
	n.relayMu.Lock()
	defer n.relayMu.Unlock()
	if n.relayReservation == nil {
		return nil
	}
	if n.relayPeerInfo.ID == "" {
		return nil
	}
	myID := n.host.ID().String()

	var result []string
	for _, addr := range n.loadBootstrapPeers() {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		if pi.ID != n.relayPeerInfo.ID {
			continue
		}
		full := addr + "/p2p-circuit/p2p/" + myID
		result = append(result, full)
		log.Printf("[RELAY] built relay addr: %s", full)
	}
	return result
}

// ============================================================
// OFFLINE QUEUE — очередь сообщений при потере связи
// ============================================================

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

func (n *Node) enqueuePending(msg Message) {
	n.pendingMu.Lock()
	n.pendingMessages = append(n.pendingMessages, msg)
	count := len(n.pendingMessages)
	n.pendingMu.Unlock()
	log.Printf("[QUEUE] enqueued %s (total: %d)", msg.ID, count)
	n.savePendingQueue()
}

func (n *Node) hasPending() bool {
	n.pendingMu.Lock()
	defer n.pendingMu.Unlock()
	return len(n.pendingMessages) > 0
}

func (n *Node) flushPending() {
	n.pendingMu.Lock()
	queue := make([]Message, len(n.pendingMessages))
	copy(queue, n.pendingMessages)
	n.pendingMu.Unlock()

	if len(queue) == 0 {
		return
	}

	log.Printf("[QUEUE] flushing %d pending messages", len(queue))
	// Переотправляем всё. Из очереди НЕ удаляем.
	// Удаление — только при получении [DELIVERED] (removePendingByRef).
	for _, msg := range queue {
		n.tryReplicate(msg)
	}
	log.Printf("[QUEUE] flushed, remaining: %d", len(queue))
}

// removePendingByRef — удаляет сообщение из очереди по ID.
// Вызывается при получении [DELIVERED] от получателя —
// только тогда доставка считается состоявшейся.
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

func (n *Node) tryReplicate(msg Message) bool {
	if n.host == nil {
		return false
	}
	msg.ReplicatedFrom = msg.Sender
	msg.IsOwn = false
	data, err := json.Marshal(msg)
	if err != nil {
		return false
	}

	if msg.Recipient != "" {
		return n.sendToRecipient(msg, data)
	}

	peers := n.host.Network().Peers()
	var alive []peer.ID
	for _, p := range peers {
		if p.String() == msg.Sender {
			continue
		}
		if !n.isPeerDead(p.String()) {
			alive = append(alive, p)
		}
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

// ============================================================
// АДРЕСНАЯ МАРШРУТИЗАЦИЯ (этап 4.2)
// ============================================================

func (n *Node) sendToRecipient(msg Message, data []byte) bool {
	targetID, err := peer.Decode(msg.Recipient)
	if err != nil {
		log.Printf("[REPLICA] invalid Recipient %q: %v", msg.Recipient, err)
		return false
	}

	for _, p := range n.host.Network().Peers() {
		if p == targetID && !n.isPeerDead(p.String()) {
			go n.sendReplicaToPeer(targetID, data)
			return true
		}
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
		go n.sendReplicaToPeer(pi.ID, data)
		return true
	}

	log.Printf("[STATUS] recipient not found: %s", msg.Recipient)
	return false
}

func (n *Node) sendReplicaToPeer(targetID peer.ID, data []byte) {
	randomDelay(10, 30)
	// Глобальный семафор — не более 10 одновременных stream'ов.
	n.sendSem <- struct{}{}
	defer func() { <-n.sendSem }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s, err := n.host.NewStream(ctx, targetID, protocolID)
	if err != nil {
		log.Printf("[REPLICA] NewStream to %s failed: %v", targetID, err)
		return
	}
	defer s.Close()
	fmt.Fprintf(s, "%s%s\n", REPLICA_PREFIX, string(data))
}

// sendServiceViaBootstrap — отправка сервисного сообщения через bootstrap (relay).
// Используется для [CONTACT_HELLO] / [CONTACT_HELLO_ACK].
// Синхронный — возвращает ошибку, чтобы UI знал результат.
// Без randomDelay — handshake важнее антидетекта.
// Всегда через bootstrap — не зависит от circuit (может быть stale).
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

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		s, err := n.host.NewStream(ctx, pi.ID, protocolID)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("NewStream to %s failed: %w", pi.ID, err)
			continue
		}
		_, err = fmt.Fprintf(s, "%s%s\n", REPLICA_PREFIX, string(data))
		s.Close()
		cancel()
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

// ============================================================
// NOTIFIEE
// ============================================================

type nodeNotifiee struct {
	node *Node
}

func (nn *nodeNotifiee) Connected(net network.Network, conn network.Conn) {
	remote := conn.RemotePeer().String()
	log.Printf("[NOTIFY] connected to %s", remote)
	go func() {
		time.Sleep(1 * time.Second)

		if nn.node.hasPending() {
			log.Printf("[NOTIFY] flushing pending after connect to %s", remote)
			nn.node.flushPending()
		}

		// Если reconnect с relay-сервером (VPS) — немедленно
		// пересоздать резервацию и анонсировать новый адрес.
		if nn.node.isRelayAddr(remote) {
			nn.node.refreshRelayAndAnnounce()
		}
	}()
}

func (nn *nodeNotifiee) Disconnected(net network.Network, conn network.Conn) {
	log.Printf("[NOTIFY] disconnected from %s", conn.RemotePeer())
}

func (nn *nodeNotifiee) Listen(net network.Network, addr ma.Multiaddr)      {}
func (nn *nodeNotifiee) ListenClose(net network.Network, addr ma.Multiaddr) {}

// isRelayAddr — true, если remote — наш relay-сервер (bootstrap).
// Используется в Notifiee, чтобы реагировать только на relay-reconnect.
func (n *Node) isRelayAddr(remotePeerID string) bool {
	for _, addr := range n.loadBootstrapPeers() {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		if pi.ID.String() == remotePeerID {
			return true
		}
	}
	return false
}

// refreshRelayAndAnnounce — немедленно пересоздаёт резервацию на VPS
// и анонсирует новый адрес. Вызывается из Notifiee при reconnect с VPS.
// Это закрывает «первое окно»: резервация теряется при disconnect,
// а relayLoop её обновляет только через N минут.
// SendAnnounce вызывается только если есть announceMultiaddrs — иначе
// первый ANNOUNCE придёт от Flutter (_sendAnnounce).
func (n *Node) refreshRelayAndAnnounce() {
	n.relayMu.Lock()
	// Throttle: не чаще одного раза в 10 секунд.
	// НО: если предыдущая резервация упала — throttle не блокирует.
	// Иначе при быстром reconnect refresh пропускается, резервация не восстанавливается.
	if time.Since(n.lastRelayRefresh) < 10*time.Second && n.lastReservationErr == nil {
		n.relayMu.Unlock()
		return
	}
	n.lastRelayRefresh = time.Now()
	relayInfo := n.relayPeerInfo
	n.relayMu.Unlock()

	if relayInfo.ID == "" {
		// Нет relay-инфо — нечего пересоздавать.
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := n.ReserveRelaySlot(ctx, relayInfo)
	cancel()
	if err != nil {
		log.Printf("[RELAY] refresh on reconnect failed: %v", err)
		return
	}
	log.Printf("[RELAY] refreshed on reconnect")

	// Анонсируем немедленно, если есть что. Это закрывает «второе окно»:
	// при смене relay-адреса другие узлы узнают о нём сразу, а не через 4 минуты.
	if len(n.announceMultiaddrs) > 0 {
		n.SendAnnounce(n.announceMultiaddrs)
	}
}

// ============================================================
// ANNOUNCE — справочник пиров (VPS)
// ============================================================

func (n *Node) announcePeer(peerID string, multiaddrs []string) {
	if len(multiaddrs) == 0 {
		return
	}
	n.announcedMu.Lock()
	defer n.announcedMu.Unlock()
	if n.announcedPeers == nil {
		n.announcedPeers = make(map[string]announcedPeer)
	}
	n.announcedPeers[peerID] = announcedPeer{
		Multiaddrs: multiaddrs,
		LastSeen:   time.Now(),
	}
	log.Printf("[ANNOUNCE] %s → %d addrs (%v)", peerID, len(multiaddrs), multiaddrs)
}

func (n *Node) lookupPeer(peerID string) ([]string, bool) {
	n.announcedMu.Lock()
	defer n.announcedMu.Unlock()
	p, ok := n.announcedPeers[peerID]
	if !ok {
		return nil, false
	}
	if time.Since(p.LastSeen) > ANNOUNCE_TTL {
		delete(n.announcedPeers, peerID)
		return nil, false
	}
	return p.Multiaddrs, true
}

func (n *Node) cleanupAnnounced() {
	connected := make(map[string]bool)
	if n.host != nil {
		for _, p := range n.host.Network().Peers() {
			connected[p.String()] = true
		}
	}

	n.announcedMu.Lock()
	defer n.announcedMu.Unlock()
	now := time.Now()
	for id, p := range n.announcedPeers {
		if !connected[id] {
			delete(n.announcedPeers, id)
			log.Printf("[ANNOUNCE] peer %s not connected — removed", id)
			continue
		}
		if now.Sub(p.LastSeen) > ANNOUNCE_TTL {
			delete(n.announcedPeers, id)
			log.Printf("[ANNOUNCE] TTL expired: %s", id)
		}
	}
}

func (n *Node) SendAnnounce(multiaddrs []string) {
	if n.host == nil || len(multiaddrs) == 0 {
		return
	}

	n.announceMultiaddrs = multiaddrs

	relayAddrs := n.GetRelayAddrs()
	allAddrs := make([]string, 0, len(multiaddrs)+len(relayAddrs))
	allAddrs = append(allAddrs, multiaddrs...)
	allAddrs = append(allAddrs, relayAddrs...)

	var sb strings.Builder
	sb.WriteString(ANNOUNCE_PREFIX)
	sb.WriteString("\n")
	for _, a := range allAddrs {
		sb.WriteString(a)
		sb.WriteString("\n")
	}
	sb.WriteString(END_PREFIX)
	sb.WriteString("\n")
	payload := sb.String()

	bootstrapPeers := n.loadBootstrapPeers()
	for _, addr := range bootstrapPeers {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		go func(pid peer.ID) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s, err := n.host.NewStream(ctx, pid, protocolID)
			if err != nil {
				log.Printf("[ANNOUNCE] send failed to %s: %v", pid, err)
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s", payload)
			log.Printf("[ANNOUNCE] sent to %s: %d addrs", pid, len(allAddrs))
		}(pi.ID)
	}

	for _, p := range n.host.Network().Peers() {
		isBootstrap := false
		for _, addr := range bootstrapPeers {
			if pi, err := peer.AddrInfoFromString(addr); err == nil && pi.ID == p {
				isBootstrap = true
				break
			}
		}
		if isBootstrap {
			continue
		}
		go func(pid peer.ID) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s, err := n.host.NewStream(ctx, pid, protocolID)
			if err != nil {
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s", payload)
		}(p)
	}
}

func (n *Node) FindPeerByID(targetID string) ([]string, error) {
	if n.host == nil {
		return nil, fmt.Errorf("node not started")
	}

	if addrs, ok := n.lookupPeer(targetID); ok {
		log.Printf("[FIND] local hit: %s → %d addrs: %v", targetID, len(addrs), addrs)
		return addrs, nil
	}

	bootstrapPeers := n.loadBootstrapPeers()
	for _, addr := range bootstrapPeers {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		s, err := n.host.NewStream(ctx, pi.ID, protocolID)
		if err != nil {
			cancel()
			log.Printf("[FIND] cannot open stream to %s: %v", pi.ID, err)
			continue
		}
		fmt.Fprintf(s, "%s%s\n", FIND_PREFIX, targetID)
		buf := make([]byte, 64*1024)
		s.SetReadDeadline(time.Now().Add(5 * time.Second))
		nr, _ := s.Read(buf)
		s.Close()
		cancel()

		response := strings.TrimSpace(string(buf[:nr]))
		addrs := parseMultiaddrsResponse(response, FOUND_PREFIX)
		if len(addrs) > 0 {
			log.Printf("[FIND] %s → %d addrs (via %s): %v", targetID, len(addrs), pi.ID, addrs)
			return addrs, nil
		}
		if strings.HasPrefix(response, NOT_FOUND_PREFIX) {
			log.Printf("[FIND] %s not found on %s", targetID, pi.ID)
		}
	}

	return nil, fmt.Errorf("not found")
}

func parseMultiaddrsResponse(response, prefix string) []string {
	lines := strings.Split(response, "\n")
	if len(lines) == 0 {
		return nil
	}
	if strings.TrimSpace(lines[0]) != prefix {
		return nil
	}
	var addrs []string
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" || line == END_PREFIX {
			continue
		}
		addrs = append(addrs, line)
	}
	return addrs
}

// ============================================================

func (n *Node) handleStream(stream network.Stream) {
	defer stream.Close()

	// Читаем до \n или до EOF — libp2p может фрагментировать сообщение.
	stream.SetReadDeadline(time.Now().Add(15 * time.Second))

	var full []byte
	buf := make([]byte, 64*1024)
	for {
		nr, err := stream.Read(buf)
		if nr > 0 {
			full = append(full, buf[:nr]...)
			if bytes.Contains(buf[:nr], []byte("\n")) {
				break
			}
		}
		if err != nil {
			break
		}
		if len(full) > 4*1024*1024 {
			break
		}
	}
	msg := strings.TrimSpace(string(full))

	if strings.HasPrefix(msg, ANNOUNCE_PREFIX) {
		remoteID := stream.Conn().RemotePeer().String()
		addrs := parseMultiaddrsResponse(msg, ANNOUNCE_PREFIX)
		if len(addrs) > 0 {
			n.announcePeer(remoteID, addrs)
		}
		return
	}

	if strings.HasPrefix(msg, FIND_PREFIX) {
		targetID := strings.TrimPrefix(msg, FIND_PREFIX)
		targetID = strings.TrimSpace(targetID)

		addrs, ok := n.lookupPeer(targetID)

		if !ok && n.host != nil {
			targetPID, err := peer.Decode(targetID)
			if err == nil {
				connected := false
				for _, p := range n.host.Network().Peers() {
					if p == targetPID {
						connected = true
						break
					}
				}
				if connected {
					relayAddr := n.buildRelayAddrFor(targetID)
					if relayAddr != "" {
						addrs = []string{relayAddr}
						ok = true
						log.Printf("[FIND] fallback to relay for connected peer %s", targetID)
					}
				}
			}
		}

		if ok && len(addrs) > 0 {
			var sb strings.Builder
			sb.WriteString(FOUND_PREFIX)
			sb.WriteString("\n")
			for _, a := range addrs {
				sb.WriteString(a)
				sb.WriteString("\n")
			}
			sb.WriteString(END_PREFIX)
			sb.WriteString("\n")
			stream.Write([]byte(sb.String()))
		} else {
			stream.Write([]byte(NOT_FOUND_PREFIX + "\n"))
		}
		return
	}

	if strings.HasPrefix(msg, REPLICA_PREFIX) {
		payload := strings.TrimPrefix(msg, REPLICA_PREFIX)
		var replicaMsg Message
		if err := json.Unmarshal([]byte(payload), &replicaMsg); err == nil {
			// Сервисные сообщения (DELIVERED/READ/CONTACT_*):
			// мне → обработать, relay → форвард, иначе → drop.
			if n.isServiceType(replicaMsg.Type) {
				myID := n.host.ID().String()
				if replicaMsg.Recipient == myID {
					n.handleServiceMessage(replicaMsg)
				} else if n.isRelay {
					go n.replicateMessage(replicaMsg)
				}
				return
			}
			// Деобфускация только для Version < 2 (история, broadcast).
			if replicaMsg.Version < 2 {
				if plaintext, ok := n.deobfuscate(replicaMsg.Text); ok {
					replicaMsg.Text = plaintext
				}
			}
			replicaMsg.ReplicatedAt = time.Now()
			replicaMsg.IsOwn = false

			// Version 2 — E2E-шифрованное. Маршрутизация по Recipient.
			if replicaMsg.Version == MESSAGE_VERSION_E2E {
				myID := n.host.ID().String()
				if replicaMsg.Recipient == myID {
					// Нам — расшифровываем.
					plaintext, err := n.decryptFromSender(replicaMsg.Sender, replicaMsg.Text)
					if err != nil {
						log.Printf("[REPLICA] decrypt failed from %s: %v", replicaMsg.Sender, err)
						return
					}
					replicaMsg.Text = plaintext
					replicaMsg.SenderName = n.nameForPeer(replicaMsg.Sender)
					if n.memory.Add(replicaMsg) {
						// Батч [DELIVERED] — копим refs, отправляем по таймеру/размеру.
						n.queueDelivered(replicaMsg.Sender, replicaMsg.ID)
						if n.messageHook != nil {
							data, _ := json.Marshal(replicaMsg)
							n.messageHook(string(data))
						}
					}
					return
				}
				// Не нам.
				if n.isRelay {
					go n.replicateMessage(replicaMsg)
				}
				return
			}

			// Version 0 — старая логика.
			if replicaMsg.Recipient != "" && n.host != nil {
				myID := n.host.ID().String()
				if replicaMsg.Recipient != myID {
					if n.isRelay {
						log.Printf("[REPLICA] relay forward to %s", replicaMsg.Recipient)
						go n.replicateMessage(replicaMsg)
					} else {
						log.Printf("[REPLICA] not for us (recipient=%s), dropping", replicaMsg.Recipient)
					}
					return
				}
				replicaMsg.SenderName = n.nameForPeer(replicaMsg.Sender)
				if n.memory.Add(replicaMsg) {
					log.Printf("[REPLICA] received addressed message for us: %s", replicaMsg.ID)
					if n.messageHook != nil {
						data, _ := json.Marshal(replicaMsg)
						n.messageHook(string(data))
					}
				}
				return
			}

			if n.memory.Add(replicaMsg) {
				go n.replicateMessage(replicaMsg)
				if n.messageHook != nil {
					data, _ := json.Marshal(replicaMsg)
					n.messageHook(string(data))
				}
			}
		}
		return
	}

	if strings.HasPrefix(msg, RESTORE_PREFIX) {
		nodeID := strings.TrimPrefix(msg, RESTORE_PREFIX)
		replicas := n.memory.GetReplicasFor(nodeID)
		if len(replicas) > 0 {
			for _, r := range replicas {
				data, _ := json.Marshal(r)
				stream.Write([]byte(REPLICA_PREFIX + string(data) + "\n"))
			}
		}
		return
	}

	if strings.HasPrefix(msg, "PEERS:") {
		payload := strings.TrimPrefix(msg, "PEERS:")
		var peerAddrs []string
		if err := json.Unmarshal([]byte(payload), &peerAddrs); err == nil {
			for _, addr := range peerAddrs {
				peerInfo, err := peer.AddrInfoFromString(addr)
				if err == nil {
					n.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour*24)
				}
			}
			allAddrs := n.GetKnownPeers()
			myAddrs := n.GetMultiaddrs()
			for _, addr := range myAddrs {
				if !strings.Contains(addr, "127.0.0.1") {
					allAddrs = append(allAddrs, addr)
				}
			}
			data, _ := json.Marshal(allAddrs)
			stream.Write([]byte("PEERS:" + string(data) + "\n"))
		}
		return
	}

	if strings.HasPrefix(msg, "RELAY:") {
		parts := strings.SplitN(strings.TrimPrefix(msg, "RELAY:"), ":", 2)
		if len(parts) == 2 {
			targetPeerID := parts[0]
			actualMsg := parts[1]
			if plaintext, ok := n.deobfuscate(actualMsg); ok {
				actualMsg = plaintext
			}
			targetPID, err := peer.Decode(targetPeerID)
			if err == nil {
				go func() {
					ctx := context.Background()
					s, err := n.host.NewStream(ctx, targetPID, protocolID)
					if err != nil {
						return
					}
					defer s.Close()
					fmt.Fprintf(s, "%s\n", n.obfuscate(actualMsg))
				}()
			}
		}
		return
	}

	if strings.HasPrefix(msg, STEGO_PREFIX) {
		return
	}

	if plaintext, ok := n.deobfuscate(msg); ok {
		msg = plaintext
	}

	if msg == n.ethHash {
		stream.Write([]byte("ACCEPTED\n"))
		return
	}

	remoteID := stream.Conn().RemotePeer().String()

	if strings.HasPrefix(msg, "CHAIN:") {
		parts := strings.SplitN(msg, ":", 3)
		if len(parts) == 3 {
			nextRelays := parts[1]
			actualMsg := parts[2]
			if nextRelays != "" {
				relays := strings.Split(nextRelays, ",")
				n.sendViaRelayChain(relays, actualMsg)
			} else {
				n.processMessageRelayed(actualMsg, remoteID)
			}
			return
		}
	}

	n.processMessage(msg, remoteID, false)
}

// findMyMessageByID — находит моё сообщение по ID в памяти.
// Возвращает копию и true, если найдено.
func (n *Node) findMyMessageByID(id string) (Message, bool) {
	if id == "" {
		return Message{}, false
	}
	all := n.memory.GetAll()
	for _, msg := range all {
		if msg.ID == id {
			return msg, true
		}
	}
	return Message{}, false
}

// isServiceType — классификация. Только тип, без побочных эффектов.
// Сервисные сообщения не идут в UI.
func (n *Node) isServiceType(t MessageType) bool {
	switch t {
	case TypeDelivered, TypeRead, TypeContactRequest, TypeContactAccept, TypeContactReject,
		TypeContactHello, TypeContactHelloAck, TypeTtlUpdate:
		return true
	}
	return false
}

// handleServiceMessage — обработка сервисного сообщения.
// Вызывается только если Recipient == myID (проверка в handleStream).
func (n *Node) handleServiceMessage(m Message) {
	switch m.Type {
	case TypeDelivered:
		// Батч: обрабатываем все Refs. Одиночный: Ref.
		refs := m.Refs
		if len(refs) == 0 && m.Ref != "" {
			refs = []string{m.Ref}
		}
		// Обновляем read_enabled контакта один раз на всё сообщение.
		if m.ReadEnabled != nil && n.contacts != nil {
			_ = n.SetContactReadEnabled(m.Sender, *m.ReadEnabled)
		}
		for _, ref := range refs {
			n.handleDeliveredRef(ref, m.Sender)
		}

	case TypeRead:
		// Батч (Refs не пуст) — обрабатываем циклом.
		// Одиночный (Refs пуст, Ref заполнен) — как раньше.
		refs := m.Refs
		if len(refs) == 0 && m.Ref != "" {
			refs = []string{m.Ref}
		}
		for _, ref := range refs {
			// Если я не делюсь статусом прочтения — не показываю чужой [READ].
			// Иконка становится ✓✓🔒 (StatusHidden), а не ✓✓ (цвет).
			statusToSet := StatusRead
			if !n.myReadEnabled {
				statusToSet = StatusHidden
			}
			// Файловый чанк: [READ] приходит по последнему чанку.
			// Распространяем статус на все чанки этого файла.
			if msg, ok := n.findMyMessageByID(ref); ok {
				if msg.MediaType == "file" && msg.ChunkTotal > 0 {
					all := n.memory.GetAll()
					for _, cm := range all {
						if cm.MediaID == msg.MediaID {
							n.setMessageStatus(cm.ID, statusToSet)
						}
					}
					log.Printf("[SERVICE] read ack (file %s, %d chunks) ref=%s from=%s",
						msg.MediaID, msg.ChunkTotal, ref, m.Sender)
				} else {
					n.setMessageStatus(ref, statusToSet)
					log.Printf("[SERVICE] read ack ref=%s from=%s", ref, m.Sender)
				}
			} else {
				n.setMessageStatus(ref, statusToSet)
				log.Printf("[SERVICE] read ack ref=%s (not found in memory) from=%s", ref, m.Sender)
			}

			// Режим after_read: сообщение получателя прочитано — запускаем
			// таймер удаления на отправителе (у нас). Если ExpiresAt ещё не
			// установлен — ставим now + ttlPeriodSeconds.
			if msg, ok := n.findMyMessageByID(ref); ok {
				if msg.TtlMode == "after_read" && msg.ExpiresAt.IsZero() && msg.TtlPeriodSeconds > 0 {
					expiresAt := time.Now().Add(time.Duration(msg.TtlPeriodSeconds) * time.Second)
					if n.memory.SetExpiresAt(ref, expiresAt) {
						log.Printf("[TTL] after_read: set ExpiresAt for %s (+%ds)", ref, msg.TtlPeriodSeconds)
						n.scheduleSaveState()
						// Push в Dart: сообщаем об обновлении ExpiresAt.
						if updated, ok := n.findMyMessageByID(ref); ok {
							if data, err := json.Marshal(updated); err == nil {
								if n.messageHook != nil {
									n.messageHook(string(data))
								}
							}
						}
					}
				}
			}
		}

	case TypeContactRequest:
		n.handleContactRequest(m)

	case TypeContactAccept:
		n.handleContactAccept(m)

	case TypeContactReject:
		n.handleContactReject(m)

	case TypeContactHello:
		n.handleContactHello(m)

	case TypeContactHelloAck:
		n.handleContactHelloAck(m)

	case TypeTtlUpdate:
		n.handleTtlUpdate(m)
	}
}

// handleTtlUpdate — обновляет ExpiresAt для сообщения по ref.
// Приходит от отправителя, когда он перешёл на hard (auto-hard).
// Payload: m.Text — expires_in_seconds (целое, строкой).
func (n *Node) handleTtlUpdate(m Message) {
	if m.Ref == "" {
		return
	}
	seconds, err := strconv.Atoi(m.Text)
	if err != nil || seconds <= 0 {
		log.Printf("[TTL_UPDATE] invalid seconds %q ref=%s", m.Text, m.Ref)
		return
	}
	expiresAt := time.Now().Add(time.Duration(seconds) * time.Second)
	if n.memory.SetExpiresAt(m.Ref, expiresAt) {
		log.Printf("[TTL_UPDATE] set ExpiresAt for %s (+%ds)", m.Ref, seconds)
		n.scheduleSaveState()
		if updated, ok := n.findMyMessageByID(m.Ref); ok {
			if data, err := json.Marshal(updated); err == nil {
				if n.messageHook != nil {
					n.messageHook(string(data))
				}
			}
		}
	} else {
		log.Printf("[TTL_UPDATE] message not found ref=%s", m.Ref)
	}
}

// handleContactRequest — обрабатывает входящий запрос на контакт.
// Payload шифрован E2E. Расшифровываем, парсим, сохраняем в requests store.
func (n *Node) handleContactRequest(m Message) {
	if m.Version != MESSAGE_VERSION_E2E {
		log.Printf("[SERVICE] contact_request without E2E, dropped")
		return
	}
	plaintext, err := n.decryptFromSender(m.Sender, m.Text)
	if err != nil {
		log.Printf("[SERVICE] contact_request decrypt failed: %v", err)
		return
	}
	var payload struct {
		Name        string `json:"name"`
		Ed25519Pub  string `json:"ed25519_pub"`
		X25519Pub   string `json:"x25519_pub"`
		Signature   string `json:"signature"`
		ReadEnabled bool   `json:"read_enabled"`
	}
	if err := json.Unmarshal([]byte(plaintext), &payload); err != nil {
		log.Printf("[SERVICE] contact_request parse failed: %v", err)
		return
	}
	if n.requests == nil {
		log.Printf("[SERVICE] requests store not initialized")
		return
	}
	req := ContactRequest{
		ID:          m.ID,
		PeerID:      m.Sender,
		Name:        payload.Name,
		Ed25519Pub:  payload.Ed25519Pub,
		X25519Pub:   payload.X25519Pub,
		Signature:   payload.Signature,
		ReadEnabled: payload.ReadEnabled,
		Status:      RequestStatusPending,
	}
	if err := n.requests.Add(req); err != nil {
		log.Printf("[SERVICE] contact_request add failed: %v", err)
		return
	}
	// A попал в requests store — временный контакт больше не нужен.
	n.RemoveTempContact(m.Sender)
	log.Printf("[SERVICE] contact_request saved from %s", m.Sender)

	// Уведомляем UI о новом запросе (push, не polling).
	if n.messageHook != nil {
		data, _ := json.Marshal(m)
		n.messageHook(string(data))
	}
}

// handleContactAccept — обрабатывает принятие нашего запроса.
// Находим свой исходящий запрос (по Ref) — сохраняем контакт, шлём [DELIVERED].
func (n *Node) handleContactAccept(m Message) {
	if m.Version != MESSAGE_VERSION_E2E {
		log.Printf("[SERVICE] contact_accept without E2E, dropped")
		return
	}
	plaintext, err := n.decryptFromSender(m.Sender, m.Text)
	if err != nil {
		log.Printf("[SERVICE] contact_accept decrypt failed: %v", err)
		return
	}
	var payload struct {
		RequestID   string `json:"request_id"`
		Name        string `json:"name"`
		Ed25519Pub  string `json:"ed25519_pub"`
		X25519Pub   string `json:"x25519_pub"`
		Signature   string `json:"signature"`
		ReadEnabled bool   `json:"read_enabled"`
	}
	if err := json.Unmarshal([]byte(plaintext), &payload); err != nil {
		log.Printf("[SERVICE] contact_accept parse failed: %v", err)
		return
	}
	if n.contacts == nil {
		log.Printf("[SERVICE] contacts store not initialized")
		return
	}
	if err := n.AddContact(m.Sender, payload.Ed25519Pub, payload.X25519Pub, payload.Signature, "", payload.Name, payload.ReadEnabled); err != nil {
		log.Printf("[SERVICE] contact_accept add_contact failed: %v", err)
		return
	}
	if n.contacts != nil {
		_ = n.contacts.SetConfirmed(m.Sender)
	}
	log.Printf("[SERVICE] contact_accept saved contact %s (ref=%s)", m.Sender, m.Ref)

	// Уведомляем UI — перечитать контакты (confirmed обновился).
	if n.messageHook != nil {
		data, _ := json.Marshal(m)
		n.messageHook(string(data))
	}
}

// handleContactReject — обрабатывает отклонение нашего запроса.
func (n *Node) handleContactReject(m Message) {
	log.Printf("[SERVICE] contact_reject ref=%s from=%s", m.Ref, m.Sender)
}

// handleContactHello — обрабатывает [CONTACT_HELLO] от A.
// Открытое. Payload: peerID + публичные ключи + подпись A.
// B делает AddTempContact(A), чтобы потом расшифровать [CONTACT_REQUEST] (E2E).
// Если A уже в постоянных контактах — temp не создаётся.
func (n *Node) handleContactHello(m Message) {
	if m.Sender == "" {
		log.Printf("[SERVICE] contact_hello: empty sender, dropped")
		return
	}
	if n.host == nil {
		log.Printf("[SERVICE] contact_hello: host not started")
		return
	}
	myID := n.host.ID().String()
	if m.Sender == myID {
		log.Printf("[SERVICE] contact_hello: self, dropped")
		return
	}
	if m.Text == "" {
		log.Printf("[SERVICE] contact_hello: empty payload from %s", m.Sender)
		return
	}

	var payload struct {
		PeerID     string `json:"peerID"`
		Ed25519Pub string `json:"ed25519_pub"`
		X25519Pub  string `json:"x25519_pub"`
		Signature  string `json:"signature"`
	}
	if err := json.Unmarshal([]byte(m.Text), &payload); err != nil {
		log.Printf("[SERVICE] contact_hello: parse failed from %s: %v", m.Sender, err)
		return
	}
	if payload.PeerID != m.Sender {
		log.Printf("[SERVICE] contact_hello: peerID mismatch (%s != %s), dropped", payload.PeerID, m.Sender)
		return
	}
	if payload.Ed25519Pub == "" || payload.X25519Pub == "" {
		log.Printf("[SERVICE] contact_hello: missing keys from %s", m.Sender)
		return
	}

	// Проверяем подпись — если валидна, verified=true.
	verified := false
	if payload.Signature != "" {
		if verifyContactSignature(payload.PeerID, payload.Ed25519Pub, payload.X25519Pub, payload.Signature) {
			verified = true
		}
	}

	// Если A уже в постоянных контактах — не создаём temp.
	if _, ok := n.GetContact(m.Sender); !ok {
		n.AddTempContact(Contact{
			PeerID:      payload.PeerID,
			Ed25519Pub:  payload.Ed25519Pub,
			X25519Pub:   payload.X25519Pub,
			Signature:   payload.Signature,
			Verified:    verified,
			Confirmed:   false,
			ReadEnabled: true,
		})
	}

	log.Printf("[SERVICE] contact_hello from %s — sending ack", m.Sender)
	go func(sender string) {
		if err := n.SendContactHelloAck(sender); err != nil {
			log.Printf("[SERVICE] contact_hello_ack send failed to %s: %v", sender, err)
		}
	}(m.Sender)
}

// handleContactHelloAck — обрабатывает [CONTACT_HELLO_ACK] от B.
// B прислал свои публичные ключи. Логируем. Дальнейшая логика — в Dart:
// после получения ack UI шлёт [CONTACT_REQUEST] (E2E).
func (n *Node) handleContactHelloAck(m Message) {
	if m.Sender == "" {
		log.Printf("[SERVICE] contact_hello_ack: empty sender, dropped")
		return
	}
	if m.Version != 0 {
		log.Printf("[SERVICE] contact_hello_ack: unexpected version %d, dropped", m.Version)
		return
	}
	plaintext := m.Text
	if plaintext == "" {
		log.Printf("[SERVICE] contact_hello_ack: empty payload from %s", m.Sender)
		return
	}
	var payload struct {
		PeerID     string `json:"peerID"`
		Ed25519Pub string `json:"ed25519_pub"`
		X25519Pub  string `json:"x25519_pub"`
		Signature  string `json:"signature"`
	}
	if err := json.Unmarshal([]byte(plaintext), &payload); err != nil {
		log.Printf("[SERVICE] contact_hello_ack: parse failed from %s: %v", m.Sender, err)
		return
	}
	if payload.PeerID != m.Sender {
		log.Printf("[SERVICE] contact_hello_ack: peerID mismatch (%s != %s), dropped", payload.PeerID, m.Sender)
		return
	}
	log.Printf("[SERVICE] contact_hello_ack from %s (keys: ed25519=%v, x25519=%v)",
		m.Sender, payload.Ed25519Pub != "", payload.X25519Pub != "")

	// Уведомляем UI через hook — чтобы Dart знал и мог отправить [CONTACT_REQUEST].
	if n.messageHook != nil {
		data, _ := json.Marshal(m)
		n.messageHook(string(data))
	}
}

func (n *Node) buildRelayAddrFor(targetID string) string {
	if n.host == nil {
		return ""
	}
	myID := n.host.ID().String()
	for _, a := range n.host.Addrs() {
		s := a.String()
		if strings.Contains(s, "127.0.0.1") {
			continue
		}
		if strings.Contains(s, "/ws") {
			return s + "/p2p/" + myID + "/p2p-circuit/p2p/" + targetID
		}
	}
	for _, a := range n.host.Addrs() {
		s := a.String()
		if strings.Contains(s, "127.0.0.1") {
			continue
		}
		return s + "/p2p/" + myID + "/p2p-circuit/p2p/" + targetID
	}
	return ""
}

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
					go n.replicateMessage(replicaMsg)
					if n.messageHook != nil {
						data, _ := json.Marshal(replicaMsg)
						n.messageHook(string(data))
					}
				}
			}
		}
	}
}

func (n *Node) handlePingStream(stream network.Stream) {
	defer stream.Close()
	buf := make([]byte, 64)
	nr, _ := stream.Read(buf)
	msg := strings.TrimSpace(string(buf[:nr]))
	if msg == "PING" {
		stream.Write([]byte("PONG\n"))
	}
}

func (n *Node) pingPeers() {
	go func() {
		for {
			time.Sleep(15 * time.Second)
			if n.host == nil {
				continue
			}
			peers := n.host.Network().Peers()
			for _, p := range peers {
				go func(peerID peer.ID) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					s, err := n.host.NewStream(ctx, peerID, pingProtocolID)
					if err != nil {
						n.markPeerDead(peerID.String())
						return
					}
					defer s.Close()
					s.Write([]byte("PING\n"))
					buf := make([]byte, 64)
					s.SetReadDeadline(time.Now().Add(5 * time.Second))
					nr, err := s.Read(buf)
					if err != nil || strings.TrimSpace(string(buf[:nr])) != "PONG" {
						n.markPeerDead(peerID.String())
						return
					}
					if n.markPeerAlive(peerID.String()) {
						go func() {
							if n.hasPending() {
								log.Printf("[PING] peer %s alive again, flushing pending", peerID)
								n.flushPending()
							}
						}()
					}
				}(p)
			}
		}
	}()
}

func (n *Node) reconnectLoop() {
	go func() {
		backoff := time.Second
		const maxBackoff = 30 * time.Second

		for {
			time.Sleep(backoff)
			if n.host == nil {
				continue
			}
			peers := n.host.Network().Peers()
			if len(peers) > 0 {
				if backoff != time.Second {
					backoff = time.Second
				}
				continue
			}
			log.Printf("[RECONNECT] Нет пиров, переподключаюсь к bootstrap (backoff=%s)", backoff)

			bootstrapPeers := n.loadBootstrapPeers()
			connected := false
			for _, addr := range bootstrapPeers {
				peerInfo, err := peer.AddrInfoFromString(addr)
				if err != nil {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				err = n.host.Connect(ctx, *peerInfo)
				cancel()
				if err != nil {
					log.Printf("[RECONNECT] Ошибка подключения к %s: %v", addr, err)
					continue
				}
				log.Printf("[RECONNECT] Подключён к bootstrap: %s", addr)
				go func(peerID peer.ID) {
					time.Sleep(2 * time.Second)
					n.ExchangePeers(peerID.String())
					n.flushPending()
				}(peerInfo.ID)
				connected = true
				break
			}

			if connected {
				backoff = time.Second
			} else {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		}
	}()
}

func (n *Node) announceLoop() {
	go func() {
		for {
			time.Sleep(4 * time.Minute)
			if len(n.announceMultiaddrs) > 0 {
				n.SendAnnounce(n.announceMultiaddrs)
			}
			n.flushPending()
		}
	}()
}

func (n *Node) cleanupLoop() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			n.cleanupAnnounced()
			if removed := n.memory.DeleteExpired(); removed > 0 {
				log.Printf("[TTL] purged %d expired messages", removed)
				n.scheduleSaveState()
			}
		}
	}()
}

func (n *Node) markPeerAlive(peerID string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastPing == nil {
		n.lastPing = make(map[string]time.Time)
	}
	if n.deadPeers == nil {
		n.deadPeers = make(map[string]bool)
	}
	wasDead := n.deadPeers[peerID]
	n.lastPing[peerID] = time.Now()
	if wasDead {
		delete(n.deadPeers, peerID)
	}
	return wasDead
}

func (n *Node) markPeerDead(peerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.deadPeers == nil {
		n.deadPeers = make(map[string]bool)
	}
	if !n.deadPeers[peerID] {
		n.deadPeers[peerID] = true
	}
}

func (n *Node) isPeerDead(peerID string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.deadPeers[peerID]
}

func (n *Node) getPeerWeight(peerID string) float64 {
	msgs := n.memory.GetMessagesFrom(peerID)
	if len(msgs) == 0 {
		return 0.5
	}
	total := 0.0
	for _, m := range msgs {
		total += m.Weight
	}
	return total / float64(len(msgs))
}

func (n *Node) processMessageRelayed(msg string, senderID string) {
	id := generateMsgID(msg)
	newMsg := Message{
		ID:       id,
		Text:     msg,
		Sender:   senderID,
		Time:     time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:    false,
		Score:    0,
		Weight:   0.5,
		Priority: 0,
		Mode:     1,
		Relayed:  true,
	}
	newMsg.SenderName = n.nameForPeer(newMsg.Sender)
	if n.memory.Add(newMsg) {
		n.replicateMessage(newMsg)
		if n.messageHook != nil {
			data, _ := json.Marshal(newMsg)
			n.messageHook(string(data))
		}
	}
}

func (n *Node) replicateMessage(msg Message) {
	if n.host == nil {
		return
	}
	msg.ReplicatedFrom = msg.Sender
	msg.IsOwn = false
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	if msg.Recipient != "" {
		// Адресное сообщение.
		n.sendToRecipient(msg, data)
		// В offline-очередь — только НЕ-сервисные и только на клиенте.
		// Relay (VPS) не отправитель — pending не нужен, иначе очередь
		// растёт бесконечно (68 МБ за часы) и вызывает OOM при старте.
		// Сервисные ([CONTACT_*], [DELIVERED], [READ], [TTL_UPDATE])
		// тоже не кладём: получатель их не подтверждает через [DELIVERED].
		if !n.isRelay && !n.isServiceType(msg.Type) {
			n.enqueuePending(msg)
		}
		return
	}

	peers := n.host.Network().Peers()
	log.Printf("[REPLICA] broadcast msg id=%s sender=%s: %d peers", msg.ID, msg.Sender, len(peers))

	var alive []peer.ID
	for _, p := range peers {
		dead := n.isPeerDead(p.String())
		isSender := p.String() == msg.Sender
		if isSender {
			continue
		}
		if !dead {
			alive = append(alive, p)
		}
	}
	log.Printf("[REPLICA] alive=%d", len(alive))

	if len(alive) == 0 {
		if n.isRelay {
			// Relay не отправитель — pending не нужен.
			return
		}
		log.Printf("[REPLICA] SKIP: no alive peers — enqueue")
		n.enqueuePending(msg)
		return
	}
	replicaCount := 2
	if len(alive) < replicaCount {
		replicaCount = len(alive)
	}
	for i := 0; i < replicaCount; i++ {
		go n.sendReplicaToPeer(alive[i], data)
	}
}

func (n *Node) requestRestore() {
	if n.host == nil {
		return
	}
	myID := n.host.ID().String()[:8]
	peers := n.host.Network().Peers()
	for _, p := range peers {
		go func(peerID peer.ID) {
			randomDelay(100, 500)
			ctx := context.Background()
			s, err := n.host.NewStream(ctx, peerID, protocolID)
			if err != nil {
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s%s\n", RESTORE_PREFIX, myID)
			buf := make([]byte, 2*1024*1024)
			s.SetReadDeadline(time.Now().Add(5 * time.Second))
			nr, _ := s.Read(buf)
			if nr > 0 {
				n.handleReplicaData(strings.TrimSpace(string(buf[:nr])))
			}
		}(p)
	}
}

func (n *Node) processMessageWithTTL(msg string, senderID string, isOwn bool, expiresAt time.Time) {
	n.processMessageInternal(msg, senderID, isOwn, expiresAt, "", "", 0, "", TypeMessage, false, 0, "")
}

func (n *Node) processMessageWithID(msg string, senderID string, isOwn bool, expiresAt time.Time, id string) {
	n.processMessageInternal(msg, senderID, isOwn, expiresAt, id, "", 0, "", TypeMessage, false, 0, "")
}

// processMessageWithRecipient — отправляет адресное сообщение конкретному получателю.
// version — 2 для E2E-шифрованных.
func (n *Node) processMessageWithRecipient(msg string, senderID string, isOwn bool, expiresAt time.Time, id string, recipient string, version int) {
	n.processMessageInternal(msg, senderID, isOwn, expiresAt, id, recipient, version, "", TypeMessage, false, 0, "")
}

func (n *Node) processMessageWithModeAndTTL(msg string, senderID string, isOwn bool, mode int, expiresAt time.Time) {
	if mode == 0 || n.host == nil {
		n.processMessageInternal(msg, senderID, isOwn, expiresAt, "", "", 0, "", TypeMessage, false, 0, "")
		return
	}
	relayCount := 4
	if mode == 2 {
		relayCount = 5
		time.Sleep(10 * time.Second)
	}
	relays := n.selectRelays(relayCount)
	if len(relays) < relayCount {
		n.processMessageInternal(msg, senderID, isOwn, expiresAt, "", "", 0, "", TypeMessage, false, 0, "")
		return
	}
	n.sendViaRelayChain(relays, msg)
}

func (n *Node) processMessageWithMode(msg string, senderID string, isOwn bool, mode int) {
	n.processMessageWithModeAndTTL(msg, senderID, isOwn, mode, time.Time{})
}

func (n *Node) selectRelays(count int) []string {
	peers := n.host.Network().Peers()
	var trusted []peer.ID
	var fallback []peer.ID
	for _, p := range peers {
		if n.isPeerDead(p.String()) {
			continue
		}
		weight := n.getPeerWeight(p.String())
		if weight > 0.7 {
			trusted = append(trusted, p)
		} else {
			fallback = append(fallback, p)
		}
	}
	if len(trusted) >= count {
		perm := mathrand.Perm(len(trusted))
		result := make([]string, count)
		for i := 0; i < count; i++ {
			result[i] = trusted[perm[i]].String()
		}
		return result
	}
	selected := make([]string, 0)
	for _, p := range trusted {
		selected = append(selected, p.String())
	}
	perm := mathrand.Perm(len(fallback))
	for i := 0; len(selected) < count && i < len(fallback); i++ {
		selected = append(selected, fallback[perm[i]].String())
	}
	return selected
}

func (n *Node) sendViaRelayChain(relays []string, msg string) {
	if len(relays) == 0 {
		return
	}
	obfuscated := n.obfuscate(msg)
	chainMsg := fmt.Sprintf("CHAIN:%s:%s", strings.Join(relays[1:], ","), obfuscated)
	relayPeer, err := peer.Decode(relays[0])
	if err != nil {
		return
	}
	randomDelay(10, 50)
	ctx := context.Background()
	s, err := n.host.NewStream(ctx, relayPeer, protocolID)
	if err != nil {
		return
	}
	defer s.Close()
	s.Write([]byte(chainMsg + "\n"))
}

func (n *Node) processMessage(msg string, senderID string, isOwn bool) {
	n.processMessageInternal(msg, senderID, isOwn, time.Time{}, "", "", 0, "", TypeMessage, false, 0, "")
}

func (n *Node) processMessageInternal(msg string, senderID string, isOwn bool, expiresAt time.Time, providedID string, recipient string, version int, plainText string, msgType MessageType, viaBootstrap bool, ttlPeriodSeconds int, ttlMode string) {
	inputVector := textToVector(msg)
	outputVector, _ := forward(inputVector, n.layers)
	answer := vectorToText(outputVector)

	n.mu.Lock()
	similar := n.memory.FindSimilar(msg, 0.7)
	var contextVector []float64
	if len(similar) > 0 {
		contextVector = make([]float64, VectorDim)
		totalWeight := 0.0
		for _, s := range similar {
			sv := textToVector(s.Text)
			w := s.Weight
			for i := range sv {
				contextVector[i] += sv[i] * w
			}
			totalWeight += w
		}
		for i := range contextVector {
			contextVector[i] /= totalWeight
		}
		for i := range inputVector {
			inputVector[i] = inputVector[i]*0.7 + contextVector[i]*0.3
		}
	}
	lr := 0.01
	if n.adaptive != nil {
		lr = n.adaptive.LearningRate
	}
	n.layers = train(n.layers, inputVector, outputVector, inputVector, lr)
	n.layersDirty = true
	n.mu.Unlock()

	go n.broadcastLayers()

	ethicsVec := hashToVector(n.ethHash)
	msgVec := textToVector(msg)
	ethicsScore := cosineSimilarity(ethicsVec, msgVec)
	initialWeight := 0.3 + ethicsScore*0.5

	if n.preHash != "" {
		preVec := hashToVector(n.preHash)
		preScore := cosineSimilarity(preVec, msgVec)
		preWeight := 0.2 + preScore*0.6
		initialWeight = initialWeight*0.4 + preWeight*0.6
	}
	if n.antiHash != "" {
		antiVec := hashToVector(n.antiHash)
		antiScore := cosineSimilarity(antiVec, msgVec)
		antiWeight := 0.3 + antiScore*0.5
		initialWeight = initialWeight * (1.0 - antiWeight*0.8)
		if initialWeight < 0.1 {
			initialWeight = 0.1
		}
	}

	priority := 0
	if isOwn {
		nodeWeight := 0.5
		activeMsgs := n.memory.GetActiveMessages(0.5)
		if len(activeMsgs) > 0 {
			totalWeight := 0.0
			for _, m := range activeMsgs {
				totalWeight += m.Weight
			}
			nodeWeight = totalWeight / float64(len(activeMsgs))
		}
		if nodeWeight > 0.7 {
			priority = 100
		}
	}

	id := providedID
	if id == "" {
		id = generateMsgID(msg)
	}

	readEnabled := n.myReadEnabled
	newMsg := Message{
		ID:               id,
		Text:             msg,
		PlainText:        plainText,
		Sender:           senderID,
		Recipient:        recipient,
		Version:          version,
		Type:             msgType,
		Time:             time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:            isOwn,
		Score:            0,
		Weight:           initialWeight,
		Priority:         priority,
		Mode:             0,
		ExpiresAt:        expiresAt,
		ReadEnabled:      &readEnabled,
		TtlPeriodSeconds: ttlPeriodSeconds,
		TtlMode:          ttlMode,
	}
	if n.memory.Add(newMsg) {
		// Если это наше E2E-сообщение — фиксируем статус StatusSent.
		if isOwn && version == MESSAGE_VERSION_E2E && newMsg.Type == TypeMessage {
			n.setMessageStatus(id, StatusSent)
		}
		if viaBootstrap {
			// Служебные контакт-протокола — всегда через bootstrap.
			// Circuit может быть stale — direct таймаутит.
			if err := n.sendServiceViaBootstrap(newMsg); err != nil {
				log.Printf("[SERVICE] viaBootstrap failed for %s: %v", newMsg.ID, err)
			}
		} else {
			n.replicateMessage(newMsg)
		}
		if n.messageHook != nil {
			data, _ := json.Marshal(newMsg)
			n.messageHook(string(data))
		}
	}

	if answer != "" && recipient == "" {
		answerMsg := Message{
			ID:       generateMsgID(answer),
			Text:     answer,
			Sender:   "🌐 Сеть",
			Time:     time.Now().UTC().Format("2006-01-02T15:04:05"),
			IsOwn:    false,
			Score:    0,
			Weight:   0.5,
			Priority: 0,
			Mode:     0,
		}
		if n.memory.Add(answerMsg) {
			n.replicateMessage(answerMsg)
		}
	}

	go n.saveState()
}

func migrateTime(t string) string {
	if strings.Contains(t, "T") {
		return t
	}
	if len(t) == 8 && t[2] == ':' {
		return time.Now().Format("2006-01-02") + "T" + t
	}
	return time.Now().Format("2006-01-02T15:04:05")
}

func (n *Node) loadBootstrapPeers() []string {
	if len(n.configBootstrap) > 0 {
		return n.configBootstrap
	}
	var peers []string
	if envPeers := os.Getenv("ISOTOPE_BOOTSTRAP_PEERS"); envPeers != "" {
		for _, p := range strings.Split(envPeers, ",") {
			if p = strings.TrimSpace(p); p != "" {
				peers = append(peers, p)
			}
		}
	}
	if data, err := os.ReadFile("bootstrap.txt"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				peers = append(peers, line)
			}
		}
	}
	return peers
}

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
	// Миграция: старые state хранили статусы в отдельном map.
	// Переносим их в Message.Status (новый единый источник истины).
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

	// Однократная очистка своих E2E-сообщений без PlainText (баг до фикса).
	n.runE2ECleanupOnce()

	if n.pendingFile == "" {
		if n.stateFile != "" {
			n.pendingFile = n.stateFile + ".queue"
		}
	}
	n.loadPendingQueue()

	// Инициализация E2E-ключей (Ed25519 + X25519).
	n.loadOrGenerateE2EKeys()

	// Инициализация хранилища контактов (отдельно от state).
	if n.contactsFile == "" && n.stateFile != "" {
		n.contactsFile = filepath.Join(filepath.Dir(n.stateFile), "isotope_contacts.json")
	}
	if n.contactsFile != "" {
		_ = os.MkdirAll(filepath.Dir(n.contactsFile), 0700)
		n.contacts = NewContactsStore(n.contactsFile)
	}

	// Инициализация хранилища входящих запросов (отдельно от state).
	if n.requestsFile == "" && n.stateFile != "" {
		n.requestsFile = filepath.Join(filepath.Dir(n.stateFile), "isotope_requests.json")
	}
	if n.requestsFile != "" {
		_ = os.MkdirAll(filepath.Dir(n.requestsFile), 0700)
		n.requests = NewRequestsStore(n.requestsFile)
	}

	// Инициализация списка удалённых (отдельно от state).
	if n.deletedFile == "" && n.stateFile != "" {
		n.deletedFile = filepath.Join(filepath.Dir(n.stateFile), "isotope_deleted.json")
	}
	if n.deletedFile != "" {
		_ = os.MkdirAll(filepath.Dir(n.deletedFile), 0700)
		n.deleted = NewDeletedStore(n.deletedFile)
	}

	// Инициализация пользовательских настроек (отдельно от state).
	// isotope_settings.json — рядом со state.
	settingsFile := filepath.Join(filepath.Dir(n.stateFile), "isotope_settings.json")
	_ = os.MkdirAll(filepath.Dir(settingsFile), 0700)
	n.settingsStore = NewSettingsStore(settingsFile)
	// Синхронизация кэша с загруженным значением.
	n.myReadEnabled = n.settingsStore.GetMyReadEnabled()

	var priv crypto.PrivKey
	keyBytes, err := n.loadPrivateKey()
	if err != nil {
		priv, _, _ = crypto.GenerateKeyPair(crypto.RSA, 2048)
		keyBytes, _ = crypto.MarshalPrivateKey(priv)
		n.savePrivateKey(keyBytes)
	} else {
		priv, _ = crypto.UnmarshalPrivateKey(keyBytes)
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

	// Автоочистка: удалить self-contact (баг старых версий — свой QR).
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

	n.pingPeers()
	if !n.isRelay {
		n.reconnectLoop()
	}
	n.announceLoop()
	n.cleanupLoop()
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

// GetMessages — возвращает все НЕ истёкшие сообщения.
func (n *Node) GetMessages() []Message {
	all := n.memory.GetAll()
	now := time.Now()
	alive := make([]Message, 0, len(all))
	for _, m := range all {
		if !m.ExpiresAt.IsZero() && now.After(m.ExpiresAt) {
			continue
		}
		alive = append(alive, m)
	}
	return alive
}

// GetPeers — возвращает список активных пиров
func (n *Node) GetPeers() []string {
	if n.host == nil {
		return []string{}
	}
	peers := n.host.Network().Peers()
	result := make([]string, 0, len(peers))
	for _, p := range peers {
		result = append(result, p.String())
	}
	return result
}

// GetWeight — возвращает вес узла
func (n *Node) GetWeight() float64 {
	msgs := n.memory.GetActiveMessages(0.5)
	if len(msgs) == 0 {
		return 0.5
	}
	total := 0.0
	for _, m := range msgs {
		total += m.Weight
	}
	return total / float64(len(msgs))
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

// GetMultiaddrs — возвращает все адреса узла.
func (n *Node) GetMultiaddrs() []string {
	if n.host == nil {
		return []string{}
	}
	myID := n.host.ID().String()

	ifaceAddrs, err := n.host.Network().InterfaceListenAddresses()
	if err != nil {
		ifaceAddrs = nil
	}

	fallbackAddrs := n.host.Addrs()

	seen := make(map[string]bool)
	var result []string

	addAddr := func(addr string) {
		if seen[addr] {
			return
		}
		seen[addr] = true
		result = append(result, addr+"/p2p/"+myID)
	}

	for _, a := range ifaceAddrs {
		s := a.String()
		if strings.Contains(s, "127.0.0.1") {
			continue
		}
		addAddr(s)
	}

	for _, a := range fallbackAddrs {
		s := a.String()
		if strings.Contains(s, "127.0.0.1") {
			continue
		}
		addAddr(s)
	}

	return result
}

// ConnectToPeer — подключение к пиру
func (n *Node) ConnectToPeer(multiaddr string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	peerInfo, err := peer.AddrInfoFromString(multiaddr)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := n.host.Connect(ctx, *peerInfo); err != nil {
		return err
	}
	go func() {
		time.Sleep(2 * time.Second)
		n.ExchangePeers(peerInfo.ID.String())
	}()
	return nil
}

// ConnectToPeerWithFallback — пробует все multiaddr параллельно.
func (n *Node) ConnectToPeerWithFallback(multiaddrs []string) (string, error) {
	if len(multiaddrs) == 0 {
		return "", fmt.Errorf("empty multiaddrs list")
	}
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}

	type result struct {
		addr string
		err  error
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan result, len(multiaddrs))

	for _, ma := range multiaddrs {
		go func(addr string) {
			peerInfo, err := peer.AddrInfoFromString(addr)
			if err != nil {
				results <- result{addr: addr, err: err}
				return
			}

			dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
			defer dialCancel()

			if err := n.host.Connect(dialCtx, *peerInfo); err != nil {
				results <- result{addr: addr, err: err}
				return
			}
			results <- result{addr: addr, err: nil}
		}(ma)
	}

	var lastErr error
	for i := 0; i < len(multiaddrs); i++ {
		r := <-results
		if r.err == nil {
			cancel()
			log.Printf("[CONNECT] success %s", r.addr)
			go func() {
				time.Sleep(2 * time.Second)
				peerInfo, err := peer.AddrInfoFromString(r.addr)
				if err == nil {
					n.ExchangePeers(peerInfo.ID.String())
				}
			}()
			return r.addr, nil
		}
		lastErr = r.err
		log.Printf("[CONNECT] failed %s: %v", r.addr, r.err)
	}

	return "", fmt.Errorf("all dials failed: %v", lastErr)
}

// SendMessage — отправляет сообщение всем пирам (broadcast).
func (n *Node) SendMessage(text string, ttlPeriod string, ttlMode string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	id := generateMsgID(text)
	n.processMessageInternal(text, n.host.ID().String(), true, expiresAt, id, "", 0, "", TypeMessage, false, ttlSeconds, ttlMode)
	return id, nil
}

// SendToPeer — отправляет сообщение конкретному пиру (адресно).
// Требуется контакт с x25519_pub. Шифрует через box.Seal (Version=2).
func (n *Node) SendToPeer(peerID string, text string, ttlPeriod string, ttlMode string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	if peerID == "" {
		return "", fmt.Errorf("peerID is required")
	}

	encrypted, err := n.encryptForRecipient(peerID, text)
	if err != nil {
		return "", fmt.Errorf("encrypt failed: %w", err)
	}

	ttlSeconds := parsePeriod(ttlPeriod)
	var expiresAt time.Time
	if ttlSeconds > 0 && ttlMode == "hard" {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	id := generateMsgID(text)
	n.processMessageInternal(encrypted, n.host.ID().String(), true, expiresAt, id, peerID, MESSAGE_VERSION_E2E, text, TypeMessage, false, ttlSeconds, ttlMode)
	return id, nil
}

// SendDelivered — отправляет подтверждение доставки сообщения отправителю.
// См. SendRead.
func (n *Node) SendDelivered(ref, recipient string) error {
	return n.sendConfirmation(TypeDelivered, ref, recipient)
}

// SendDeliveredBatch — отправляет батч подтверждений доставки.
// Одно сообщение [DELIVERED] с массивом Refs вместо N отдельных.
func (n *Node) SendDeliveredBatch(refs []string, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if recipient == "" {
		return fmt.Errorf("recipient is required")
	}
	if len(refs) == 0 {
		return nil
	}
	id := generateMsgID(fmt.Sprintf("delivered-batch:%d", len(refs)))
	readEnabled := n.myReadEnabled
	msg := Message{
		ID:          id,
		Text:        "",
		Sender:      n.host.ID().String(),
		Recipient:   recipient,
		Type:        TypeDelivered,
		Refs:        refs,
		Version:     0,
		Time:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		ReadEnabled: &readEnabled,
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		log.Printf("[CONFIRM] delivered-batch (%d refs) to %s failed: %v", len(refs), recipient, err)
		return err
	}
	log.Printf("[CONFIRM] delivered-batch (%d refs) to %s", len(refs), recipient)
	return nil
}

// queueDelivered — добавляет ref в буфер. Если буфер достиг 10 — flush.
// Иначе flush произойдёт по таймеру (500 мс).
func (n *Node) queueDelivered(sender, ref string) {
	if sender == "" || ref == "" {
		return
	}
	n.pendingDeliveredMu.Lock()
	n.pendingDelivered[sender] = append(n.pendingDelivered[sender], ref)
	count := len(n.pendingDelivered[sender])
	n.pendingDeliveredMu.Unlock()

	if count >= 10 {
		go n.flushDelivered(sender)
		return
	}
	time.AfterFunc(500*time.Millisecond, func() {
		n.flushDelivered(sender)
	})
}

// flushDelivered — отправляет батч [DELIVERED] для указанного sender.
func (n *Node) flushDelivered(sender string) {
	n.pendingDeliveredMu.Lock()
	refs, ok := n.pendingDelivered[sender]
	if !ok || len(refs) == 0 {
		n.pendingDeliveredMu.Unlock()
		return
	}
	delete(n.pendingDelivered, sender)
	n.pendingDeliveredMu.Unlock()

	if err := n.SendDeliveredBatch(refs, sender); err != nil {
		log.Printf("[CONFIRM] delivered-batch flush failed for %s: %v", sender, err)
	}
}

// handleDeliveredRef — обработка одного ref из [DELIVERED] (одиночного или батча).
func (n *Node) handleDeliveredRef(ref, sender string) {
	if ref == "" {
		return
	}
	n.removePendingByRef(ref)

	peerReadEnabled := n.getPeerReadEnabled(sender)
	if !n.myReadEnabled || !peerReadEnabled {
		n.setMessageStatus(ref, StatusHidden)
		log.Printf("[SERVICE] delivered ack (hidden) ref=%s from=%s", ref, sender)

		if msg, ok := n.findMyMessageByID(ref); ok {
			if msg.TtlMode == "after_read" && msg.ExpiresAt.IsZero() && msg.TtlPeriodSeconds > 0 {
				expiresAt := time.Now().Add(time.Duration(msg.TtlPeriodSeconds) * time.Second)
				if n.memory.SetExpiresAt(ref, expiresAt) {
					log.Printf("[TTL] after_read → hard (hidden): set ExpiresAt for %s (+%ds)", ref, msg.TtlPeriodSeconds)
					n.scheduleSaveState()
					if updated, ok := n.findMyMessageByID(ref); ok {
						if data, err := json.Marshal(updated); err == nil {
							if n.messageHook != nil {
								n.messageHook(string(data))
							}
						}
					}
					if err := n.SendTtlUpdate(ref, msg.TtlPeriodSeconds, sender); err != nil {
						log.Printf("[TTL_UPDATE] send failed ref=%s: %v", ref, err)
					}
				}
			}
		}
	} else {
		n.setMessageStatus(ref, StatusDelivered)
		log.Printf("[SERVICE] delivered ack ref=%s from=%s", ref, sender)
	}
}

// SendTtlUpdate — отправляет получателю [TTL_UPDATE] с expires_in_seconds.
// Используется при auto-hard (отправитель знает, что прочтения не будет).
func (n *Node) SendTtlUpdate(ref string, seconds int, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if ref == "" || recipient == "" || seconds <= 0 {
		return fmt.Errorf("ref, recipient, seconds required")
	}
	text := strconv.Itoa(seconds)
	return n.sendTtlUpdate(ref, text, recipient)
}

// sendTtlUpdate — общая логика отправки [TTL_UPDATE].
// Version=0, Type=TypeTtlUpdate, Ref=msg_id, Text=seconds. Через bootstrap.
func (n *Node) sendTtlUpdate(ref, text, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	id := generateMsgID(ref + ":" + recipient)
	msg := Message{
		ID:        id,
		Text:      text,
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Version:   0,
		Type:      TypeTtlUpdate,
		Ref:       ref,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:     true,
		Weight:    0.5,
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("sendServiceViaBootstrap: %w", err)
	}
	return nil
}

// SendRead — отправляет подтверждение прочтения сообщения отправителю.
// См. SendDelivered.
// Плюс: локально запускает TTL у получателя — если это after_read,
// ставим ExpiresAt = now + period для входящего сообщения.
// Файловые чанки (MediaType=file, ChunkTotal>0) не отправляют [READ] —
// только [DELIVERED]. Меньше трафика при передаче файлов.
func (n *Node) SendRead(ref, recipient string) error {
	if msg, ok := n.findMyMessageByID(ref); ok {
		if msg.MediaType == "file" && msg.ChunkTotal > 0 {
			// [READ] для файла — только по последнему чанку.
			if msg.ChunkIndex != msg.ChunkTotal-1 {
				return nil
			}
		}
	}
	// Локальный TTL: получатель прочитал — запускаем таймер удаления.
	if msg, ok := n.findMyMessageByID(ref); ok {
		if msg.TtlMode == "after_read" && msg.ExpiresAt.IsZero() && msg.TtlPeriodSeconds > 0 {
			expiresAt := time.Now().Add(time.Duration(msg.TtlPeriodSeconds) * time.Second)
			if n.memory.SetExpiresAt(ref, expiresAt) {
				log.Printf("[TTL] after_read (recipient): set ExpiresAt for %s (+%ds)", ref, msg.TtlPeriodSeconds)
				n.scheduleSaveState()
				// Push в Dart: сообщаем об обновлении ExpiresAt.
				if updated, ok := n.findMyMessageByID(ref); ok {
					if data, err := json.Marshal(updated); err == nil {
						if n.messageHook != nil {
							n.messageHook(string(data))
						}
					}
				}
			}
		}
	}
	return n.sendConfirmation(TypeRead, ref, recipient)
}

// SendReadBatch — отправляет батч подтверждений прочтения.
// Одно сообщение [READ] с массивом Refs вместо N отдельных.
// Version=0, Type=TypeRead, Refs=msg_ids. Отправка — через bootstrap.
// Локально у получателя — TTL after_read для каждого ref (как в SendRead).
// Если !myReadEnabled — не отправляем.
func (n *Node) SendReadBatch(refs []string, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if recipient == "" {
		return fmt.Errorf("recipient is required")
	}
	if len(refs) == 0 {
		return nil
	}
	// [READ] отправляется только если делюсь статусом прочтения.
	if !n.myReadEnabled {
		return nil
	}

	// Фильтр: файловые чанки — оставляем только ПОСЛЕДНИЙ чанк каждого файла.
	// Остальные отбрасываем, чтобы [READ] был один на файл.
	seenMedia := make(map[string]bool)
	filtered := make([]string, 0, len(refs))
	for _, ref := range refs {
		if msg, ok := n.findMyMessageByID(ref); ok {
			if msg.MediaType == "file" && msg.ChunkTotal > 0 {
				// Оставляем только последний чанк.
				if msg.ChunkIndex != msg.ChunkTotal-1 {
					continue
				}
				if seenMedia[msg.MediaID] {
					continue
				}
				seenMedia[msg.MediaID] = true
			}
		}
		filtered = append(filtered, ref)
	}
	refs = filtered
	if len(refs) == 0 {
		return nil
	}

	// Локальный TTL after_read — для каждого входящего.
	for _, ref := range refs {
		if msg, ok := n.findMyMessageByID(ref); ok {
			if msg.TtlMode == "after_read" && msg.ExpiresAt.IsZero() && msg.TtlPeriodSeconds > 0 {
				expiresAt := time.Now().Add(time.Duration(msg.TtlPeriodSeconds) * time.Second)
				if n.memory.SetExpiresAt(ref, expiresAt) {
					log.Printf("[TTL] after_read (recipient batch): set ExpiresAt for %s (+%ds)", ref, msg.TtlPeriodSeconds)
					n.scheduleSaveState()
					if updated, ok := n.findMyMessageByID(ref); ok {
						if data, err := json.Marshal(updated); err == nil {
							if n.messageHook != nil {
								n.messageHook(string(data))
							}
						}
					}
				}
			}
		}
	}

	id := generateMsgID(fmt.Sprintf("read-batch:%d", len(refs)))
	readEnabled := n.myReadEnabled
	msg := Message{
		ID:          id,
		Text:        "",
		Sender:      n.host.ID().String(),
		Recipient:   recipient,
		Type:        TypeRead,
		Refs:        refs,
		Version:     0,
		Time:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		ReadEnabled: &readEnabled,
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		log.Printf("[CONFIRM] relay read-batch (%d refs) to %s failed: %v", len(refs), recipient, err)
		return err
	}
	log.Printf("[CONFIRM] relay read-batch (%d refs) to %s", len(refs), recipient)
	return nil
}

// MarkReadLocally — помечает входящие сообщения как прочитанные локально.
// Вызывается из Dart при открытии чата (перед sendReadBatch).
// Идемпотентно. Push в Dart не делает — UI обновит _unreadByPeer сам.
// Возвращает количество изменённых.
func (n *Node) MarkReadLocally(refs []string) int {
	if len(refs) == 0 {
		return 0
	}
	changed := n.memory.MarkReadLocally(refs)
	if changed > 0 {
		log.Printf("[READ] marked %d messages as read locally", changed)
		// Немедленное сохранение — отметка прочтения должна выжить
		// при убийстве приложения Android. scheduleSaveState (5 сек)
		// не успевает — окно потери.
		go func() {
			if err := n.saveState(); err != nil {
				log.Printf("[STATE] save failed: %v", err)
			}
		}()
	}
	return changed
}

// sendConfirmation — общая логика отправки [DELIVERED]/[READ].
// Version=0, Type=delivered|read, Ref=msg_id. Прямо или через relay.
func (n *Node) sendConfirmation(msgType MessageType, ref, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if ref == "" || recipient == "" {
		return fmt.Errorf("ref and recipient are required")
	}
	// [READ] отправляется только если делюсь статусом прочтения.
	// [DELIVERED] — всегда.
	if msgType == TypeRead && !n.myReadEnabled {
		return nil
	}

	id := generateMsgID(fmt.Sprintf("%s:%s", ref, msgTypeString(msgType)))
	readEnabled := n.myReadEnabled
	msg := Message{
		ID:          id,
		Text:        "",
		Sender:      n.host.ID().String(),
		Recipient:   recipient,
		Type:        msgType,
		Ref:         ref,
		Version:     0,
		Time:        time.Now().UTC().Format("2006-01-02T15:04:05"),
		ReadEnabled: &readEnabled,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Пытаемся напрямую — если peerstore знает рабочий адрес.
	targetID, err := peer.Decode(recipient)
	if err == nil {
		for _, p := range n.host.Network().Peers() {
			if p == targetID && !n.isPeerDead(p.String()) {
				go n.sendReplicaToPeer(targetID, data)
				log.Printf("[CONFIRM] direct %s ref=%s to %s", msgTypeString(msgType), ref, recipient)
				return nil
			}
		}
	}

	// Fallback — через relay (bootstrap).
	if n.sendToRecipient(msg, data) {
		log.Printf("[CONFIRM] relay %s ref=%s to %s", msgTypeString(msgType), ref, recipient)
		return nil
	}
	return fmt.Errorf("no route for confirmation")
}

// msgTypeString — строковое имя типа для логов.
func msgTypeString(t MessageType) string {
	switch t {
	case TypeDelivered:
		return "delivered"
	case TypeRead:
		return "read"
	case TypeContactRequest:
		return "contact_request"
	case TypeContactAccept:
		return "contact_accept"
	case TypeContactReject:
		return "contact_reject"
	case TypeContactHello:
		return "contact_hello"
	case TypeContactHelloAck:
		return "contact_hello_ack"
	case TypeTtlUpdate:
		return "ttl_update"
	default:
		return "message"
	}
}

// SendContactRequest — отправляет запрос на добавление в контакты.
// recipient — PeerID получателя (должен быть в контактах, чтобы был x25519_pub).
// displayName — представление отправителя. Если пусто — берётся MyDisplayName.
// Payload (имя, ключи, подпись, read_enabled) шифруется E2E.
// Version=2.
func (n *Node) SendContactRequest(recipient, displayName string) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	if recipient == "" {
		return "", fmt.Errorf("recipient is required")
	}

	if displayName == "" {
		displayName = n.GetMyDisplayName()
	}
	readEnabled := n.myReadEnabled

	payload := map[string]interface{}{
		"name":         displayName,
		"ed25519_pub":  base64.StdEncoding.EncodeToString(n.ed25519Pub),
		"x25519_pub":   base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		"signature":    n.signMyX25519(),
		"read_enabled": readEnabled,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encrypted, err := n.encryptForRecipient(recipient, string(payloadJSON))
	if err != nil {
		return "", fmt.Errorf("encrypt failed: %w", err)
	}

	id := generateMsgID(fmt.Sprintf("req:%s:%d", recipient, time.Now().UnixNano()))
	n.processMessageInternal(encrypted, n.host.ID().String(), true, time.Time{}, id, recipient, MESSAGE_VERSION_E2E, "", TypeContactRequest, true, 0, "")

	return id, nil
}

// SendContactHello — отправляет [CONTACT_HELLO] получателю.
// Открытое (Version=0). Payload: peerID + публичные ключи + подпись A.
// Нужно, чтобы B мог AddTempContact(A) → расшифровать [CONTACT_REQUEST] (E2E).
func (n *Node) SendContactHello(recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if recipient == "" {
		return fmt.Errorf("recipient is required")
	}

	payload := map[string]interface{}{
		"peerID":      n.host.ID().String(),
		"ed25519_pub": base64.StdEncoding.EncodeToString(n.ed25519Pub),
		"x25519_pub":  base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		"signature":   n.signMyX25519(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	id := generateMsgID(fmt.Sprintf("hello:%s:%d", recipient, time.Now().UnixNano()))
	msg := Message{
		ID:        id,
		Text:      string(payloadJSON),
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Type:      TypeContactHello,
		Version:   0,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send hello failed: %w", err)
	}
	log.Printf("[REQUESTS] contact_hello sent to %s (id=%s)", recipient, id)
	return nil
}

// SendContactHelloAck — отправляет [CONTACT_HELLO_ACK] отправителю.
// Открытое (Version=0). Payload: peerID + публичные ключи + подпись.
func (n *Node) SendContactHelloAck(recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if recipient == "" {
		return fmt.Errorf("recipient is required")
	}

	payload := map[string]interface{}{
		"peerID":      n.host.ID().String(),
		"ed25519_pub": base64.StdEncoding.EncodeToString(n.ed25519Pub),
		"x25519_pub":  base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		"signature":   n.signMyX25519(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	id := generateMsgID(fmt.Sprintf("hello_ack:%s:%d", recipient, time.Now().UnixNano()))
	msg := Message{
		ID:        id,
		Text:      string(payloadJSON),
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Type:      TypeContactHelloAck,
		Version:   0,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send hello_ack failed: %w", err)
	}
	log.Printf("[REQUESTS] contact_hello_ack sent to %s (id=%s)", recipient, id)
	return nil
}

// SendContactAccept — отправляет принятие запроса на контакт.
// requestID — ID исходного запроса (Ref). recipient — PeerID запросившего.
// Version=2, E2E.
func (n *Node) SendContactAccept(requestID, recipient string) error {
	return n.sendContactControl(TypeContactAccept, requestID, recipient)
}

// SendContactReject — отправляет отклонение запроса на контакт.
func (n *Node) SendContactReject(requestID, recipient string) error {
	return n.sendContactControl(TypeContactReject, requestID, recipient)
}

// sendContactControl — общая логика accept/reject.
// Шифруем E2E. Version=2.
// Payload: request_id, display_name (моё представление), read_enabled.
// Ключи (ed25519_pub, x25519_pub, signature) — для случая, если у A
// ещё нет контакта B (например, при [CONTACT_REJECT] из ниоткуда).
func (n *Node) sendContactControl(msgType MessageType, requestID, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if requestID == "" || recipient == "" {
		return fmt.Errorf("requestID and recipient are required")
	}

	readEnabled := n.myReadEnabled
	payload := map[string]interface{}{
		"request_id":   requestID,
		"name":         n.GetMyDisplayName(),
		"ed25519_pub":  base64.StdEncoding.EncodeToString(n.ed25519Pub),
		"x25519_pub":   base64.StdEncoding.EncodeToString(n.x25519Pub[:]),
		"signature":    n.signMyX25519(),
		"read_enabled": readEnabled,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	encrypted, err := n.encryptForRecipient(recipient, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("encrypt failed: %w", err)
	}

	id := generateMsgID(fmt.Sprintf("%s:%s:%d", msgTypeString(msgType), requestID, time.Now().UnixNano()))
	n.processMessageInternal(encrypted, n.host.ID().String(), true, time.Time{}, id, recipient, MESSAGE_VERSION_E2E, "", msgType, true, 0, "")

	return nil
}

// setMessageStatus — устанавливает статус сообщения по ID.
// Правила приоритета (в Memory.SetStatus):
//   - hidden (3) — терминальное. Не повышается.
//   - остальные — не понижаются (только повышение).
func (n *Node) setMessageStatus(id string, status MessageStatus) {
	if id == "" {
		return
	}
	if n.memory.SetStatus(id, status) {
		log.Printf("[STATUS] %s → %d", id, status)
		// Throttled save — статусы сохраняются раз в 5 сек.
		n.scheduleSaveState()
	}
}

// scheduleSaveState — отложенная запись state (throttle 5 сек).
// Защита от лишних I/O при частых setMessageStatus.
func (n *Node) scheduleSaveState() {
	n.mu.Lock()
	if n.saveStateScheduled {
		n.mu.Unlock()
		return
	}
	n.saveStateScheduled = true
	n.mu.Unlock()

	go func() {
		time.Sleep(5 * time.Second)
		n.mu.Lock()
		n.saveStateScheduled = false
		n.mu.Unlock()
		if err := n.saveState(); err != nil {
			log.Printf("[STATE] save failed: %v", err)
		}
	}()
}

// getMessageStatus — возвращает статус сообщения по ID.
// 0 — неизвестен.
func (n *Node) getMessageStatus(id string) MessageStatus {
	if msg, ok := n.findMyMessageByID(id); ok {
		return msg.Status
	}
	return 0
}

// GetMessageStatuses — возвращает map статусов для UI.
// Формат: map[msg_id]status (1/2/3/4). Статус 0 не включается.
// Пробегает memory.GetAll() — статус теперь в Message.Status.
func (n *Node) GetMessageStatuses() map[string]MessageStatus {
	all := n.memory.GetAll()
	result := make(map[string]MessageStatus, len(all))
	for _, msg := range all {
		if msg.Status > 0 {
			result[msg.ID] = msg.Status
		}
	}
	return result
}

// GetRequests — возвращает все pending-запросы на контакт.
func (n *Node) GetRequests() []ContactRequest {
	if n.requests == nil {
		return []ContactRequest{}
	}
	return n.requests.GetPending()
}

// AcceptRequestByID — принимает входящий запрос по ID.
// Добавляет контакт, отправляет [CONTACT_ACCEPT], удаляет запрос.
func (n *Node) AcceptRequestByID(id string) error {
	if n.requests == nil {
		return fmt.Errorf("requests store not initialized")
	}
	req, ok := n.requests.Get(id)
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	if err := n.AddContact(req.PeerID, req.Ed25519Pub, req.X25519Pub, req.Signature, "", req.Name, req.ReadEnabled); err != nil {
		return fmt.Errorf("add contact failed: %w", err)
	}

	if n.contacts != nil {
		_ = n.contacts.SetConfirmed(req.PeerID)
	}

	if err := n.SendContactAccept(id, req.PeerID); err != nil {
		log.Printf("[REQUESTS] accept send failed: %v", err)
	}

	if err := n.requests.Remove(id); err != nil {
		log.Printf("[REQUESTS] remove failed: %v", err)
	}
	log.Printf("[REQUESTS] accepted %s (peer=%s)", id, req.PeerID)
	return nil
}

// RejectRequestByID — отклоняет входящий запрос по ID.
// Отправляет [CONTACT_REJECT], удаляет запрос.
func (n *Node) RejectRequestByID(id string) error {
	if n.requests == nil {
		return fmt.Errorf("requests store not initialized")
	}
	req, ok := n.requests.Get(id)
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	if err := n.SendContactReject(id, req.PeerID); err != nil {
		log.Printf("[REQUESTS] reject send failed: %v", err)
	}

	if err := n.requests.Remove(id); err != nil {
		log.Printf("[REQUESTS] remove failed: %v", err)
	}
	log.Printf("[REQUESTS] rejected %s (peer=%s)", id, req.PeerID)
	return nil
}

// GetKnownPeers — возвращает список известных multiaddr
func (n *Node) GetKnownPeers() []string {
	if n.host == nil {
		return []string{}
	}
	var addrs []string
	peers := n.host.Peerstore().Peers()
	for _, p := range peers {
		peerInfo := n.host.Peerstore().PeerInfo(p)
		for _, addr := range peerInfo.Addrs {
			addrStr := addr.String() + "/p2p/" + p.String()
			if strings.Contains(addrStr, "127.0.0.1") {
				continue
			}
			addrs = append(addrs, addrStr)
		}
	}
	return addrs
}

// ExchangePeers — обменивается списками узлов
func (n *Node) ExchangePeers(peerID string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	pid, err := peer.Decode(peerID)
	if err != nil {
		return err
	}
	allKnownAddrs := n.GetKnownPeers()
	myAddrs := n.GetMultiaddrs()
	for _, addr := range myAddrs {
		if !strings.Contains(addr, "127.0.0.1") {
			found := false
			for _, known := range allKnownAddrs {
				if known == addr {
					found = true
					break
				}
			}
			if !found {
				allKnownAddrs = append(allKnownAddrs, addr)
			}
		}
	}
	ctx := context.Background()
	s, err := n.host.NewStream(ctx, pid, protocolID)
	if err != nil {
		return err
	}
	defer s.Close()
	data, _ := json.Marshal(allKnownAddrs)
	fmt.Fprintf(s, "PEERS:%s\n", string(data))
	buf := make([]byte, 256*1024)
	s.SetReadDeadline(time.Now().Add(15 * time.Second))
	nr, _ := s.Read(buf)
	response := strings.TrimSpace(string(buf[:nr]))
	if strings.HasPrefix(response, "PEERS:") {
		payload := strings.TrimPrefix(response, "PEERS:")
		var peerAddrs []string
		if err := json.Unmarshal([]byte(payload), &peerAddrs); err == nil {
			for _, addr := range peerAddrs {
				peerInfo, err := peer.AddrInfoFromString(addr)
				if err == nil {
					n.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour*24)
				}
			}
		}
	}
	return nil
}

// AddContact — добавляет или обновляет контакт.
// 4.4: проверяет подпись над peerID || x25519_pub.
// localName — как я называю контакт (не передаётся в сеть).
// remoteName — представление контакта о себе (пришло в payload/QR).
// Результат:
//   - подпись валидна       → Verified: true
//   - подписи нет           → Verified: false
//   - подпись невалидна     → Verified: false + лог
//   - Ed25519Pub пустой при непустой Signature → Verified: false + лог
//
// Отправка блокируется не здесь, а в encryptForRecipient (требует X25519Pub).
func (n *Node) AddContact(peerID, ed25519Pub, x25519Pub, signature, localName, remoteName string, readEnabled bool) error {
	if n.contacts == nil {
		return fmt.Errorf("contacts store not initialized")
	}

	// Защита на уровне ядра: нельзя добавить себя в контакты.
	if n.host != nil && peerID == n.host.ID().String() {
		return fmt.Errorf("cannot add self as contact")
	}

	// Если пользователь явно добавляет контакт (QR) — убрать из удалённых.
	if n.deleted != nil {
		if err := n.deleted.RemoveFromDeleted(peerID); err != nil {
			log.Printf("[DELETED] removeFromDeleted failed for %s: %v", peerID, err)
		}
	}

	verified := false

	if signature != "" {
		if ed25519Pub == "" || x25519Pub == "" {
			log.Printf("[CONTACT] incomplete keys for %s (ed25519=%v, x25519=%v) — contact saved as unverified",
				peerID, ed25519Pub != "", x25519Pub != "")
		} else if verifyContactSignature(peerID, ed25519Pub, x25519Pub, signature) {
			verified = true
			log.Printf("[CONTACT] signature verified for %s", peerID)
		} else {
			log.Printf("[CONTACT] signature invalid for %s — contact saved as unverified", peerID)
		}
	}

	return n.contacts.Add(Contact{
		PeerID:      peerID,
		Ed25519Pub:  ed25519Pub,
		X25519Pub:   x25519Pub,
		Signature:   signature,
		Verified:    verified,
		Name:        localName,
		RemoteName:  remoteName,
		ReadEnabled: readEnabled,
	})
}

// SetContactReadEnabled — устанавливает read_enabled для контакта.
// Использует UpdateReadEnabled — явное значение (не «не понижает»).
func (n *Node) SetContactReadEnabled(peerID string, enabled bool) error {
	if n.contacts == nil {
		return fmt.Errorf("contacts store not initialized")
	}
	return n.contacts.UpdateReadEnabled(peerID, enabled)
}

// RenameContact — устанавливает локальное имя контакта (Name).
// Локальное имя — как я называю контакт. Не передаётся в сеть.
func (n *Node) RenameContact(peerID, localName string) error {
	if n.contacts == nil {
		return fmt.Errorf("contacts store not initialized")
	}
	return n.contacts.RenameContact(peerID, localName)
}

// RemoveContact — удаляет контакт у меня. У собеседника остаётся.
// Отправка сообщений контакту после удаления невозможна (encryptForRecipient
// не найдёт x25519_pub). Также удаляет всю переписку с этим контактом —
// сообщения в памяти и их статусы. И добавляет peerID в список удалённых —
// чтобы он не вернулся после перезапуска (из P2PService history / UI).
func (n *Node) RemoveContact(peerID string) error {
	if n.contacts == nil {
		return fmt.Errorf("contacts store not initialized")
	}

	// 1. Удаляем контакт из isotope_contacts.json (если есть).
	// Если контакта нет — не ошибка (удаляем «старый узел»).
	_ = n.contacts.Remove(peerID)

	// 2. Добавляем в список удалённых.
	if n.deleted != nil {
		if err := n.deleted.Add(peerID); err != nil {
			log.Printf("[DELETED] add failed for %s: %v", peerID, err)
		}
	}

	// 3. Удаляем все сообщения с этим peerID (sender или recipient).
	all := n.memory.GetAll()
	for _, m := range all {
		if m.Sender == peerID || m.Recipient == peerID {
			n.memory.Remove(m.ID)
			// Статус уходит вместе с сообщением (Message.Status).
		}
	}

	// 4. Сохраняем state (messageStatus изменился).
	n.scheduleSaveState()

	log.Printf("[CONTACTS] removed %s + messages purged + added to deleted", peerID)
	return nil
}

// getPeerReadEnabled — возвращает read_enabled контакта (из isotope_contacts.json).
// Если контакта нет или ошибка — дефолт true (обратная совместимость).
func (n *Node) getPeerReadEnabled(peerID string) bool {
	if n.contacts == nil {
		return true
	}
	c, ok := n.contacts.Get(peerID)
	if !ok {
		return true
	}
	return c.ReadEnabled
}

// SetMyReadEnabled — устанавливает мою настройку "делюсь ли статусом прочтения".
// Сохраняет в isotope_settings.json. Обновляет кэш в Node.
// Используется UI (Приватность).
func (n *Node) SetMyReadEnabled(enabled bool) {
	n.myReadEnabled = enabled
	if n.settingsStore != nil {
		if err := n.settingsStore.SetMyReadEnabled(enabled); err != nil {
			log.Printf("[SETTINGS] save my_read_enabled failed: %v", err)
		}
	}
}

// GetMyReadEnabled — возвращает текущую настройку (из кэша).
func (n *Node) GetMyReadEnabled() bool {
	return n.myReadEnabled
}

// SetMyDisplayName — устанавливает представление по умолчанию.
// Сохраняется в isotope_settings.json. Используется в QR и [CONTACT_REQUEST].
func (n *Node) SetMyDisplayName(name string) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	return n.settingsStore.SetMyDisplayName(name)
}

// GetShowNotificationContent — показывать ли содержимое в уведомлениях.
// Источник истины — settingsStore.
func (n *Node) GetShowNotificationContent() bool {
	if n.settingsStore == nil {
		return true
	}
	return n.settingsStore.GetShowNotificationContent()
}

// SetShowNotificationContent — устанавливает настройку и сохраняет.
func (n *Node) SetShowNotificationContent(enabled bool) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	return n.settingsStore.SetShowNotificationContent(enabled)
}

// GetDeletedPeers — возвращает список удалённых peerID (для Dart).
// Dart фильтрует _discoveredNodes по этому списку — удалённые не показываются.
func (n *Node) GetDeletedPeers() []string {
	if n.deleted == nil {
		return []string{}
	}
	return n.deleted.GetAll()
}

// RemoveFromDeleted — убирает peerID из списка удалённых.
// Вызывается при QR-возврате контакта.
func (n *Node) RemoveFromDeleted(peerID string) error {
	if n.deleted == nil {
		return fmt.Errorf("deleted store not initialized")
	}
	return n.deleted.RemoveFromDeleted(peerID)
}

// IsDeleted — true, если peerID в списке удалённых.
func (n *Node) IsDeleted(peerID string) bool {
	if n.deleted == nil {
		return false
	}
	return n.deleted.IsDeleted(peerID)
}

// GetMyDisplayName — возвращает представление по умолчанию.
func (n *Node) GetMyDisplayName() string {
	if n.settingsStore == nil {
		return ""
	}
	return n.settingsStore.GetMyDisplayName()
}

// SetTtl — устанавливает период и режим удаления сообщений.
// period: "10s" | "30s" | "1m" | "5m" | "15m" | "30m" | "1h" | "4h" | "24h" | "never".
// mode: nil (при never) | "after_read" | "hard".
func (n *Node) SetTtl(period string, mode string) error {
	if n.settingsStore == nil {
		return fmt.Errorf("settings store not initialized")
	}
	var modePtr *string
	if mode != "" && period != "never" && period != "forever" {
		m := mode
		modePtr = &m
	}
	return n.settingsStore.SetTtl(period, modePtr)
}

// GetTtl — возвращает период и режим удаления сообщений.
// period: "10s" | "30s" | "1m" | ... | "never".
// mode: "" (при never) | "after_read" | "hard".
func (n *Node) GetTtl() (string, string) {
	if n.settingsStore == nil {
		return "never", ""
	}
	period, modePtr := n.settingsStore.GetTtl()
	mode := ""
	if modePtr != nil {
		mode = *modePtr
	}
	return period, mode
}

// GetContacts — возвращает все контакты.
func (n *Node) GetContacts() []Contact {
	if n.contacts == nil {
		return []Contact{}
	}
	return n.contacts.GetAll()
}

// GetContact — возвращает контакт по PeerID.
func (n *Node) GetContact(peerID string) (Contact, bool) {
	if n.contacts == nil {
		return Contact{}, false
	}
	return n.contacts.Get(peerID)
}

// nameForPeer — возвращает имя отправителя по приоритету:
//   - свой peerID → MyDisplayName;
//   - контакт → Name → RemoteName;
//   - иначе — пусто (UI возьмёт короткий PeerID).
func (n *Node) nameForPeer(peerID string) string {
	if peerID == "" {
		return ""
	}
	if n.host != nil && peerID == n.host.ID().String() {
		if n.settingsStore != nil {
			return n.settingsStore.GetMyDisplayName()
		}
		return ""
	}
	if n.contacts != nil {
		if c, ok := n.contacts.Get(peerID); ok {
			if c.Name != "" {
				return c.Name
			}
			if c.RemoteName != "" {
				return c.RemoteName
			}
		}
	}
	return ""
}

// AddTempContact — добавляет временный контакт (только в памяти).
// Используется при получении [CONTACT_HELLO] от A — чтобы B мог
// расшифровать [CONTACT_REQUEST] от A (E2E). Не сохраняется на диск.
func (n *Node) AddTempContact(c Contact) {
	if c.PeerID == "" {
		return
	}
	n.tempContactsMu.Lock()
	defer n.tempContactsMu.Unlock()
	if n.tempContacts == nil {
		n.tempContacts = make(map[string]Contact)
	}
	n.tempContacts[c.PeerID] = c
	log.Printf("[CONTACTS] temp added %s (verified=%v)", c.PeerID, c.Verified)
}

// GetContactTemp — возвращает контакт по PeerID: сначала постоянный,
// потом временный. Используется в decryptFromSender — прозрачно.
func (n *Node) GetContactTemp(peerID string) (Contact, bool) {
	if c, ok := n.GetContact(peerID); ok {
		return c, true
	}
	n.tempContactsMu.Lock()
	defer n.tempContactsMu.Unlock()
	c, ok := n.tempContacts[peerID]
	return c, ok
}

// RemoveTempContact — удаляет временный контакт по PeerID.
// Вызывается после requests.Add (A попал в requests store).
func (n *Node) RemoveTempContact(peerID string) {
	n.tempContactsMu.Lock()
	defer n.tempContactsMu.Unlock()
	if _, ok := n.tempContacts[peerID]; ok {
		delete(n.tempContacts, peerID)
		log.Printf("[CONTACTS] temp removed %s", peerID)
	}
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

// node/node.go
